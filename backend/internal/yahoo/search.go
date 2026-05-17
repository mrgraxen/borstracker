package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const defaultSearchURL = "https://query2.finance.yahoo.com/v1/finance/search"

// SearchResult is a symbol match with venue metadata for the UI.
type SearchResult struct {
	Symbol          string `json:"symbol"`
	Name            string `json:"name"`
	Exchange        string `json:"exchange"`
	ExchangeDisplay string `json:"exchangeDisplay"`
	Venue           string `json:"venue"`
	QuoteType       string `json:"quoteType"`
}

type searchResponse struct {
	Quotes []struct {
		Symbol    string `json:"symbol"`
		ShortName string `json:"shortname"`
		LongName  string `json:"longname"`
		Exchange  string `json:"exchange"`
		ExchDisp  string `json:"exchDisp"`
		QuoteType string `json:"quoteType"`
		TypeDisp  string `json:"typeDisp"`
	} `json:"quotes"`
}

var allowedQuoteTypes = map[string]bool{
	"EQUITY": true,
	"ETF":    true,
	"INDEX":  true,
	"MUTUALFUND": true,
}

func (c *Client) SearchSymbols(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	base := c.searchURL
	if base == "" {
		base = defaultSearchURL
	}
	u := fmt.Sprintf("%s?q=%s&quotesCount=%d&newsCount=0&listsCount=0&enableFuzzyQuery=true",
		base, url.QueryEscape(query), limit)

	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}

	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, err
	}

	out := make([]SearchResult, 0, len(sr.Quotes))
	seen := make(map[string]struct{})
	for _, q := range sr.Quotes {
		if q.Symbol == "" {
			continue
		}
		if !allowedQuoteTypes[strings.ToUpper(q.QuoteType)] {
			continue
		}
		sym := strings.ToUpper(q.Symbol)
		if _, ok := seen[sym]; ok {
			continue
		}
		seen[sym] = struct{}{}

		name := q.ShortName
		if name == "" {
			name = q.LongName
		}
		if name == "" {
			name = sym
		}

		out = append(out, SearchResult{
			Symbol:          sym,
			Name:            name,
			Exchange:        strings.ToUpper(q.Exchange),
			ExchangeDisplay: q.ExchDisp,
			Venue:           FormatVenue(q.ExchDisp, q.Exchange),
			QuoteType:       q.QuoteType,
		})
	}
	return out, nil
}

// FormatVenue builds a human-readable trading venue label.
func FormatVenue(exchangeDisplay, exchange string) string {
	exchangeDisplay = strings.TrimSpace(exchangeDisplay)
	exchange = strings.TrimSpace(exchange)
	switch {
	case exchangeDisplay != "" && exchange != "" && !strings.EqualFold(exchangeDisplay, exchange):
		return exchangeDisplay + " (" + exchange + ")"
	case exchangeDisplay != "":
		return exchangeDisplay
	case exchange != "":
		return exchange
	default:
		return ""
	}
}
