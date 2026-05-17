package yahoo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchQuote_RetrySuccess(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"chart": {"result": [{
				"meta": {
					"currency": "USD",
					"regularMarketPrice": 150.5,
					"regularMarketOpen": 148.0,
					"regularMarketDayHigh": 151.0,
					"regularMarketDayLow": 147.0,
					"regularMarketVolume": 1000
				},
				"timestamp": [1700000000],
				"indicators": {"quote": [{"close": [150.5]}]}
			}]}
		}`))
	}))
	defer srv.Close()

	client := NewClientWithHTTP(srv.Client(), srv.URL+"/")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	quote, err := client.FetchQuote(ctx, "AAPL")
	require.NoError(t, err)
	assert.Equal(t, "AAPL", quote.Symbol)
	assert.True(t, quote.Price.IsPositive())
	assert.GreaterOrEqual(t, attempts, 2)
}

func TestNormalizeSymbol(t *testing.T) {
	s, err := NormalizeSymbol("  volv-b.st ")
	require.NoError(t, err)
	assert.Equal(t, "VOLV-B.ST", s)

	_, err = NormalizeSymbol("bad symbol!")
	assert.Error(t, err)
}
