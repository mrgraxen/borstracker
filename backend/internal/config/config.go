package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env                string
	LogLevel           string
	DatabaseURL        string
	RedisURL           string
	FrontendOrigin     string
	CookieSecure       bool
	SessionMaxAge      time.Duration
	SessionCookieName  string
	APIAddr            string
	PriceFetcherAddr   string
	RateLimitReadPerMin   int
	RateLimitWritePerMin  int
	RateLimitSearchPerMin int
	PollInterval       time.Duration
	YahooMaxConcurrent int
	PriceCacheTTL      time.Duration
}

func Load() Config {
	return Config{
		Env:                getEnv("ENV", "development"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://borstracker:borstracker@localhost:5432/borstracker?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379/0"),
		FrontendOrigin:     getEnv("FRONTEND_ORIGIN", "http://localhost"),
		CookieSecure:       getEnvBool("COOKIE_SECURE", false),
		SessionMaxAge:      time.Duration(getEnvInt("SESSION_MAX_AGE_DAYS", 30)) * 24 * time.Hour,
		SessionCookieName:  getEnv("SESSION_COOKIE_NAME", "session_id"),
		APIAddr:            getEnv("API_ADDR", ":8080"),
		PriceFetcherAddr:   getEnv("PRICE_FETCHER_ADDR", ":8081"),
		RateLimitReadPerMin:   getEnvInt("API_RATE_LIMIT_READ_PER_MIN", 300),
		RateLimitWritePerMin:  rateLimitWritePerMin(),
		RateLimitSearchPerMin: getEnvInt("API_RATE_LIMIT_SEARCH_PER_MIN", 120),
		PollInterval:       time.Duration(getEnvInt("POLL_INTERVAL_SEC", 5)) * time.Second,
		YahooMaxConcurrent: getEnvInt("YAHOO_MAX_CONCURRENCY", 10),
		PriceCacheTTL:      time.Duration(getEnvInt("PRICE_CACHE_TTL_SEC", 5)) * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func rateLimitWritePerMin() int {
	if v := os.Getenv("API_RATE_LIMIT_WRITE_PER_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return getEnvInt("API_RATE_LIMIT_PER_MIN", 60)
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}
