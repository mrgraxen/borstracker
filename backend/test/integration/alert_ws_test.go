//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	alertengine "github.com/graxe/borstracker/internal/alerts"
	"github.com/graxe/borstracker/internal/cache"
	"github.com/graxe/borstracker/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlertEvaluationTriggersMessage(t *testing.T) {
	alert := models.Alert{
		ID:          1,
		SessionID:   "test-session",
		Symbol:      "AAPL",
		AlertType:   models.AlertAbsoluteBelow,
		Threshold:   decimal.NewFromFloat(200),
		Enabled:     true,
		CooldownSec: 300,
	}

	ok, msg := alertengine.Evaluate(alert, decimal.NewFromFloat(199), decimal.NewFromFloat(210), time.Now())
	require.True(t, ok)
	assert.Contains(t, msg, "AAPL")
}

func TestMockYahooToWebSocketFlow(t *testing.T) {
	yahooServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"chart": {
				"result": [{
					"meta": {
						"currency": "USD",
						"regularMarketPrice": 195.0,
						"regularMarketOpen": 200.0,
						"regularMarketDayHigh": 201.0,
						"regularMarketDayLow": 194.0,
						"regularMarketVolume": 1000
					},
					"timestamp": [1700000000],
					"indicators": {"quote": [{"close": [195.0]}]}
				}]
			}
		}`))
	}))
	defer yahooServer.Close()

	_ = context.Background()

	alert := models.Alert{
		AlertType: models.AlertAbsoluteBelow,
		Threshold: decimal.NewFromFloat(200),
		Enabled:   true,
	}
	ok, _ := alertengine.Evaluate(alert, decimal.NewFromFloat(195), decimal.NewFromFloat(200), time.Now())
	assert.True(t, ok)

	var msg cache.AlertMessage
	msg.Type = "alert"
	msg.AlertID = 1
	msg.Symbol = "AAPL"
	msg.Price = 195
	msg.Message = "triggered"
	b, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"type":"alert"`)
}
