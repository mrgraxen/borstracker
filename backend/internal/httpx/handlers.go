package httpx

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/graxe/borstracker/internal/cache"
	"github.com/graxe/borstracker/internal/db"
	"github.com/graxe/borstracker/internal/models"
	"github.com/graxe/borstracker/internal/session"
	"github.com/graxe/borstracker/internal/ws"
	"github.com/graxe/borstracker/internal/yahoo"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

type Server struct {
	Sessions  *db.SessionRepo
	Watchlist *db.WatchlistRepo
	Alerts    *db.AlertRepo
	Prices    *db.PriceRepo
	Cache     *cache.PriceCache
	Hub       *ws.Hub
	Yahoo     *yahoo.Client
	Upgrader  websocket.Upgrader
}

type addWatchlistReq struct {
	Symbol string `json:"symbol" binding:"required"`
}

type createAlertReq struct {
	Symbol      string  `json:"symbol" binding:"required"`
	AlertType   string  `json:"alertType" binding:"required"`
	Threshold   float64 `json:"threshold" binding:"required"`
	CooldownSec *int    `json:"cooldownSec"`
}

type patchAlertReq struct {
	Enabled     *bool    `json:"enabled"`
	Threshold   *float64 `json:"threshold"`
	CooldownSec *int     `json:"cooldownSec"`
}

type patchSettingsReq struct {
	Language     *string `json:"language"`
	SoundID      *int16  `json:"soundId"`
	SoundEnabled *bool   `json:"soundEnabled"`
}

func (s *Server) SearchSymbols(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query too short"})
		return
	}
	if len(q) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query too long"})
		return
	}

	results, err := s.Yahoo.SearchSymbols(c.Request.Context(), q, 12)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "symbol search failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": jsonSlice(results)})
}

func (s *Server) GetWatchlist(c *gin.Context) {
	sid := session.ID(c)
	items, err := s.Watchlist.List(c.Request.Context(), sid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type row struct {
		Symbol   string   `json:"symbol"`
		Price    *float64 `json:"price,omitempty"`
		Open     *float64 `json:"open,omitempty"`
		Currency string   `json:"currency,omitempty"`
		Stale    bool     `json:"stale"`
	}
	out := make([]row, 0, len(items))
	for _, it := range items {
		r := row{Symbol: it.Symbol}
		if q, err := s.Cache.Get(c.Request.Context(), it.Symbol); err == nil && q != nil {
			p := q.Price.InexactFloat64()
			o := q.Open.InexactFloat64()
			r.Price = &p
			r.Open = &o
			r.Currency = q.Currency
			r.Stale = q.Stale
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (s *Server) AddWatchlist(c *gin.Context) {
	var req addWatchlistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	symbol, err := yahoo.NormalizeSymbol(req.Symbol)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sid := session.ID(c)
	if _, err := s.Watchlist.Add(c.Request.Context(), sid, symbol); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"symbol": symbol})
}

func (s *Server) RemoveWatchlist(c *gin.Context) {
	symbol, err := yahoo.NormalizeSymbol(c.Param("symbol"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sid := session.ID(c)
	if err := s.Watchlist.Remove(c.Request.Context(), sid, symbol); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) ListAlerts(c *gin.Context) {
	sid := session.ID(c)
	symbol := c.Query("symbol")
	alerts, err := s.Alerts.List(c.Request.Context(), sid, symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": jsonSlice(alerts)})
}

func (s *Server) CreateAlert(c *gin.Context) {
	var req createAlertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	symbol, err := yahoo.NormalizeSymbol(req.Symbol)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	at := models.AlertType(req.AlertType)
	if !validAlertType(at) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert type"})
		return
	}
	cooldown := 300
	if req.CooldownSec != nil {
		cooldown = *req.CooldownSec
	}
	sid := session.ID(c)
	a, err := s.Alerts.Create(c.Request.Context(), sid, symbol, at, decimal.NewFromFloat(req.Threshold), cooldown)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (s *Server) PatchAlert(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req patchAlertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var threshold *decimal.Decimal
	if req.Threshold != nil {
		t := decimal.NewFromFloat(*req.Threshold)
		threshold = &t
	}
	sid := session.ID(c)
	a, err := s.Alerts.Update(c.Request.Context(), sid, id, req.Enabled, threshold, req.CooldownSec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (s *Server) DeleteAlert(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	sid := session.ID(c)
	if err := s.Alerts.Delete(c.Request.Context(), sid, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) AlertHistory(c *gin.Context) {
	sid := session.ID(c)
	events, err := s.Alerts.History(c.Request.Context(), sid, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": jsonSlice(events)})
}

func (s *Server) PatchSettings(c *gin.Context) {
	var req patchSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sid := session.ID(c)
	sess, err := s.Sessions.Get(c.Request.Context(), sid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	lang := sess.Language
	if req.Language != nil {
		if *req.Language != "sv" && *req.Language != "en" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "language must be sv or en"})
			return
		}
		lang = *req.Language
	}
	soundID := sess.SoundID
	if req.SoundID != nil {
		if *req.SoundID < 1 || *req.SoundID > 4 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "soundId must be 1-4"})
			return
		}
		soundID = *req.SoundID
	}
	soundEnabled := sess.SoundEnabled
	if req.SoundEnabled != nil {
		soundEnabled = *req.SoundEnabled
	}
	updated, err := s.Sessions.UpdateSettings(c.Request.Context(), sid, lang, soundID, soundEnabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (s *Server) GetSettings(c *gin.Context) {
	sid := session.ID(c)
	sess, err := s.Sessions.Get(c.Request.Context(), sid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sess)
}

func (s *Server) GetChart(c *gin.Context) {
	symbol, err := yahoo.NormalizeSymbol(c.Param("symbol"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rng := c.DefaultQuery("range", "1d")
	since := chartSince(rng)

	points, err := s.Prices.Chart(c.Request.Context(), symbol, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(points) == 0 && s.Yahoo != nil {
		interval, rangeParam := yahoo.RangeParams(rng)
		chartPoints, _, err := s.Yahoo.FetchChart(c.Request.Context(), symbol, interval, rangeParam)
		if err == nil {
			points = chartPoints
		}
	}
	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "range": rng, "points": jsonSlice(points)})
}

func chartSince(rng string) time.Time {
	now := time.Now().UTC()
	switch rng {
	case "1w":
		return now.AddDate(0, 0, -7)
	case "1m":
		return now.AddDate(0, -1, 0)
	case "3m":
		return now.AddDate(0, -3, 0)
	case "1y":
		return now.AddDate(-1, 0, 0)
	default:
		return now.AddDate(0, 0, -1)
	}
}

func (s *Server) WebSocket(c *gin.Context) {
	sid := session.ID(c)
	conn, err := s.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	items, _ := s.Watchlist.List(c.Request.Context(), sid)
	symbols := make([]string, len(items))
	for i, it := range items {
		symbols[i] = it.Symbol
	}

	client := ws.NewClient(sid, s.Hub, conn)
	client.SetSymbols(symbols)
	s.Hub.Register(client)

	go client.WritePump()
	client.ReadPump(func() {
		s.Hub.Unregister(client)
		conn.Close()
	})
}

func validAlertType(at models.AlertType) bool {
	switch at {
	case models.AlertAbsoluteBelow, models.AlertAbsoluteAbove, models.AlertPctBelowOpen, models.AlertPctAboveOpen:
		return true
	}
	return false
}
