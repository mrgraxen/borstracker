package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/graxe/borstracker/internal/cache"
	"github.com/graxe/borstracker/internal/config"
	"github.com/graxe/borstracker/internal/db"
	"github.com/graxe/borstracker/internal/fetcher"
	"github.com/graxe/borstracker/internal/yahoo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	publisher := cache.NewPublisher(redisClient)

	loop := fetcher.NewLoop(
		cfg,
		db.NewWatchlistRepo(pool),
		db.NewAlertRepo(pool),
		db.NewPriceRepo(pool),
		priceCache,
		publisher,
		yahoo.NewClient(),
		logger,
	)

	go loop.Run(ctx)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	httpServer := &http.Server{Addr: cfg.PriceFetcherAddr, Handler: r}
	go func() {
		logger.Info("price-fetcher listening", "addr", cfg.PriceFetcherAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
