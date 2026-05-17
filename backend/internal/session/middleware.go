package session

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/graxe/borstracker/internal/config"
	"github.com/graxe/borstracker/internal/db"
)

const ContextKey = "session_id"

type Middleware struct {
	cfg  config.Config
	repo *db.SessionRepo
}

func NewMiddleware(cfg config.Config, repo *db.SessionRepo) *Middleware {
	return &Middleware{cfg: cfg, repo: repo}
}

func (m *Middleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(m.cfg.SessionCookieName)
		if err != nil || sessionID == "" {
			sessionID = uuid.New().String()
		}
		setSessionCookie(c, m.cfg, sessionID)

		if _, err := m.repo.Upsert(c.Request.Context(), sessionID); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "session error"})
			return
		}

		c.Set(ContextKey, sessionID)
		c.Next()
	}
}

func ID(c *gin.Context) string {
	v, _ := c.Get(ContextKey)
	s, _ := v.(string)
	return s
}

func setSessionCookie(c *gin.Context, cfg config.Config, sessionID string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(cfg.SessionMaxAge / time.Second),
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
