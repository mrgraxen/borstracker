package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/graxe/borstracker/internal/cache"
	"github.com/graxe/borstracker/internal/config"
	"github.com/graxe/borstracker/internal/db"
	"github.com/graxe/borstracker/internal/httpx"
	"github.com/graxe/borstracker/internal/metrics"
	"github.com/graxe/borstracker/internal/pubsub"
	"github.com/graxe/borstracker/internal/ratelimit"
	"github.com/graxe/borstracker/internal/session"
	"github.com/graxe/borstracker/internal/ws"
	"github.com/graxe/borstracker/internal/yahoo"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}))

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool, "migrations"); err != nil {
		logger.Warn("migrations", "err", err)
	}

	redisClient, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		logger.Error("redis connect", "err", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	priceCache := cache.NewPriceCache(redisClient, cfg.PriceCacheTTL)
	hub := ws.NewHub()

	sessions := db.NewSessionRepo(pool)
	watchlist := db.NewWatchlistRepo(pool)
	alerts := db.NewAlertRepo(pool)
	prices := db.NewPriceRepo(pool)

	srv := &httpx.Server{
		Sessions:  sessions,
		Watchlist: watchlist,
		Alerts:    alerts,
		Prices:    prices,
		Cache:     priceCache,
		Hub:       hub,
		Yahoo:     yahoo.NewClient(),
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || origin == cfg.FrontendOrigin
			},
		},
	}

	sessMW := session.NewMiddleware(cfg, sessions)
	limiter := ratelimit.New(redisClient)

	go startRedisSubscriber(ctx, redisClient, hub, logger)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	r.GET("/healthz", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "db": err.Error()})
			return
		}
		if err := priceCache.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "redis": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	v1 := r.Group("/api/v1")
	v1.Use(sessMW.Handle())
	v1.Use(rateLimitMiddleware(limiter, cfg))
	{
		v1.GET("/settings", srv.GetSettings)
		v1.PATCH("/settings", srv.PatchSettings)
		v1.GET("/symbols/search", srv.SearchSymbols)
		v1.GET("/watchlist", srv.GetWatchlist)
		v1.POST("/watchlist", srv.AddWatchlist)
		v1.DELETE("/watchlist/:symbol", srv.RemoveWatchlist)
		v1.GET("/alerts", srv.ListAlerts)
		v1.POST("/alerts", srv.CreateAlert)
		v1.PATCH("/alerts/:id", srv.PatchAlert)
		v1.DELETE("/alerts/:id", srv.DeleteAlert)
		v1.GET("/alerts/history", srv.AlertHistory)
		v1.GET("/chart/:symbol", srv.GetChart)
		v1.GET("/ws", srv.WebSocket)
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			n, err := sessions.CountActive(context.Background(), 24*time.Hour)
			if err == nil {
				metrics.ActiveSessions.Set(float64(n))
			}
			metrics.WebSocketConnections.Set(float64(hub.Count()))
		}
	}()

	httpServer := &http.Server{Addr: cfg.APIAddr, Handler: r}
	go func() {
		logger.Info("api listening", "addr", cfg.APIAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("api server", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func startRedisSubscriber(ctx context.Context, client *redis.Client, hub *ws.Hub, _ *slog.Logger) {
	redisSub := client.PSubscribe(ctx, "channel:price:*", "channel:alert:*")
	defer redisSub.Close()

	for msg := range redisSub.Channel() {
		if msg.Payload == "" {
			continue
		}
		channel := msg.Channel
		payload := []byte(msg.Payload)

		if len(channel) > len("channel:price:") && channel[:len("channel:price:")] == "channel:price:" {
			symbol := pubsub.SymbolFromPriceChannel(channel)
			var cp cache.CachedPrice
			if err := json.Unmarshal(payload, &cp); err != nil {
				continue
			}
			pm := ws.PriceMessage{
				Symbol:   symbol,
				Price:    cp.Price,
				Open:     cp.Open,
				Stale:    cp.Stale,
				Currency: cp.Currency,
				Ts:       cp.Time.Format(time.RFC3339),
			}
			hub.BroadcastPrice(symbol, ws.MarshalPrice(pm))
			continue
		}

		if len(channel) > len("channel:alert:") && channel[:len("channel:alert:")] == "channel:alert:" {
			sessionID := pubsub.SessionFromAlertChannel(channel)
			hub.BroadcastToSession(sessionID, payload)
		}
	}
}

func rateLimitMiddleware(l *ratelimit.Limiter, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/api/v1/ws" {
			c.Next()
			return
		}
		sid := session.ID(c)
		if sid == "" {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		var bucket string
		var limit int
		switch {
		case strings.HasPrefix(path, "/api/v1/symbols/search"):
			bucket = "search"
			limit = cfg.RateLimitSearchPerMin
		case c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead:
			bucket = "read"
			limit = cfg.RateLimitReadPerMin
		default:
			bucket = "write"
			limit = cfg.RateLimitWritePerMin
		}

		ok, err := l.Allow(c.Request.Context(), ratelimit.Key(sid, bucket), limit)
		if err != nil || ok {
			c.Next()
			return
		}
		c.Header("Retry-After", "60")
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
	}
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
