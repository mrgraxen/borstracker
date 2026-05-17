package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/graxe/borstracker/internal/models"
	"github.com/shopspring/decimal"
)

const baseURL = "https://query1.finance.yahoo.com/v8/finance/chart/"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    baseURL,
	}
}

func NewClientWithHTTP(c *http.Client, base string) *Client {
	if base == "" {
		base = baseURL
	}
	return &Client{httpClient: c, baseURL: base}
}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency             string  `json:"currency"`
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				RegularMarketOpen    float64 `json:"regularMarketOpen"`
				RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
				RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
				RegularMarketVolume  int64   `json:"regularMarketVolume"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

func (c *Client) FetchQuote(ctx context.Context, symbol string) (*models.Quote, error) {
	u := c.baseURL + url.PathEscape(symbol) + "?interval=1m&range=1d"
	var lastErr error
	backoffs := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}

	for attempt := 0; attempt <= len(backoffs); attempt++ {
		quote, err := c.doFetch(ctx, u, symbol)
		if err == nil {
			return quote, nil
		}
		lastErr = err
		if attempt < len(backoffs) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffs[attempt]):
			}
		}
	}
	return nil, lastErr
}

func (c *Client) FetchChart(ctx context.Context, symbol, interval, rangeParam string) ([]models.ChartPoint, *models.Quote, error) {
	u := fmt.Sprintf("%s%s?interval=%s&range=%s", c.baseURL, url.PathEscape(symbol), interval, rangeParam)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, nil, err
	}
	return parseChartBody(body, symbol)
}

func (c *Client) doFetch(ctx context.Context, u, symbol string) (*models.Quote, error) {
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	points, quote, err := parseChartBody(body, symbol)
	if err != nil {
		return nil, err
	}
	_ = points
	if quote.Price.IsZero() && len(points) > 0 {
		quote.Price = decimal.NewFromFloat(points[len(points)-1].Price)
	}
	return quote, nil
}

func (c *Client) get(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func parseChartBody(body []byte, symbol string) ([]models.ChartPoint, *models.Quote, error) {
	var cr chartResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, nil, err
	}
	if cr.Chart.Error != nil {
		return nil, nil, fmt.Errorf("yahoo error: %s", cr.Chart.Error.Description)
	}
	if len(cr.Chart.Result) == 0 {
		return nil, nil, fmt.Errorf("no chart data for %s", symbol)
	}

	r := cr.Chart.Result[0]
	meta := r.Meta

	quote := &models.Quote{
		Symbol:   strings.ToUpper(symbol),
		Price:    decimal.NewFromFloat(meta.RegularMarketPrice),
		Open:     decimal.NewFromFloat(meta.RegularMarketOpen),
		High:     decimal.NewFromFloat(meta.RegularMarketDayHigh),
		Low:      decimal.NewFromFloat(meta.RegularMarketDayLow),
		Volume:   meta.RegularMarketVolume,
		Currency: meta.Currency,
		Stale:    false,
		Time:     time.Now().UTC(),
	}

	var points []models.ChartPoint
	if len(r.Indicators.Quote) > 0 && len(r.Timestamp) > 0 {
		closes := r.Indicators.Quote[0].Close
		for i, ts := range r.Timestamp {
			if i >= len(closes) || closes[i] == 0 {
				continue
			}
			points = append(points, models.ChartPoint{
				Time:  time.Unix(ts, 0).UTC(),
				Price: closes[i],
			})
		}
	}
	return points, quote, nil
}

func RangeParams(chartRange string) (interval, rangeParam string) {
	switch chartRange {
	case "1w":
		return "15m", "5d"
	case "1m":
		return "1h", "1mo"
	case "3m":
		return "1d", "3mo"
	case "1y":
		return "1d", "1y"
	default:
		return "1m", "1d"
	}
}

func NormalizeSymbol(s string) (string, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return "", fmt.Errorf("empty symbol")
	}
	for _, c := range s {
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '.' && c != '-' {
			return "", fmt.Errorf("invalid symbol: %s", s)
		}
	}
	return s, nil
}
