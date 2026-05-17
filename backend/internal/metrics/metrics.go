package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ActiveSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "borstracker_active_sessions",
		Help: "Number of active sessions",
	})
	WebSocketConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "borstracker_websocket_connections",
		Help: "Active WebSocket connections",
	})
	AlertsTriggered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "borstracker_alerts_triggered_total",
		Help: "Total alerts triggered",
	})
	YahooRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "borstracker_yahoo_request_duration_seconds",
		Help:    "Yahoo Finance request latency",
		Buckets: prometheus.DefBuckets,
	})
	YahooErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "borstracker_yahoo_errors_total",
		Help: "Yahoo Finance request errors",
	})
	SymbolsPolled = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "borstracker_symbols_polled",
		Help: "Symbols polled in last cycle",
	})
	StalePrices = promauto.NewCounter(prometheus.CounterOpts{
		Name: "borstracker_stale_prices_total",
		Help: "Stale price responses served",
	})
)

func Handler() http.Handler {
	return promhttp.Handler()
}
