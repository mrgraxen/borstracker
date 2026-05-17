package yahoo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatVenue(t *testing.T) {
	assert.Equal(t, "Stockholm (STO)", FormatVenue("Stockholm", "STO"))
	assert.Equal(t, "NasdaqGS", FormatVenue("", "NasdaqGS"))
	assert.Equal(t, "Stockholm", FormatVenue("Stockholm", ""))
}

func TestSearchSymbols(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "finance/search")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"quotes": [
				{
					"symbol": "VOLV-B.ST",
					"shortname": "Volvo AB ser. B",
					"exchange": "STO",
					"exchDisp": "Stockholm",
					"quoteType": "EQUITY"
				},
				{
					"symbol": "VOLAF",
					"shortname": "Volvo AB",
					"exchange": "PNK",
					"exchDisp": "OTC Markets",
					"quoteType": "EQUITY"
				}
			]
		}`))
	}))
	defer srv.Close()

	client := NewClientWithHTTP(srv.Client(), srv.URL+"/", srv.URL+"/v1/finance/search")
	results, err := client.SearchSymbols(context.Background(), "volvo", 10)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "VOLV-B.ST", results[0].Symbol)
	assert.Equal(t, "Stockholm (STO)", results[0].Venue)
	assert.Equal(t, "OTC Markets (PNK)", results[1].Venue)
}
