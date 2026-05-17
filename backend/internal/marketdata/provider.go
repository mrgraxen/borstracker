package marketdata

import (
	"context"

	"github.com/graxe/borstracker/internal/models"
)

// Provider fetches market quotes and chart data from an external source.
type Provider interface {
	FetchQuote(ctx context.Context, symbol string) (*models.Quote, error)
	FetchChart(ctx context.Context, symbol, interval, rangeParam string) ([]models.ChartPoint, *models.Quote, error)
}
