package fetcher

import (
	"context"
	"log/slog"
	"sync"
	"time"

	alertengine "github.com/graxe/borstracker/internal/alerts"
	"github.com/graxe/borstracker/internal/cache"
	"github.com/graxe/borstracker/internal/config"
	"github.com/graxe/borstracker/internal/db"
	"github.com/graxe/borstracker/internal/metrics"
	"github.com/graxe/borstracker/internal/models"
	"github.com/graxe/borstracker/internal/yahoo"
)

type Loop struct {
	cfg       config.Config
	watchlist *db.WatchlistRepo
	alerts    *db.AlertRepo
	prices    *db.PriceRepo
	cache     *cache.PriceCache
	publisher *cache.Publisher
	yahoo     *yahoo.Client
	logger    *slog.Logger
}

func NewLoop(
	cfg config.Config,
	watchlist *db.WatchlistRepo,
	alerts *db.AlertRepo,
	prices *db.PriceRepo,
	cache *cache.PriceCache,
	publisher *cache.Publisher,
	yahooClient *yahoo.Client,
	logger *slog.Logger,
) *Loop {
	return &Loop{
		cfg:       cfg,
		watchlist: watchlist,
		alerts:    alerts,
		prices:    prices,
		cache:     cache,
		publisher: publisher,
		yahoo:     yahooClient,
		logger:    logger,
	}
}

func (l *Loop) Run(ctx context.Context) {
	ticker := time.NewTicker(l.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.tick(ctx)
		}
	}
}

func (l *Loop) tick(ctx context.Context) {
	symbols, err := l.watchlist.DistinctSymbols(ctx)
	if err != nil {
		l.logger.Error("distinct symbols", "err", err)
		return
	}
	metrics.SymbolsPolled.Set(float64(len(symbols)))
	if len(symbols) == 0 {
		return
	}

	sem := make(chan struct{}, l.cfg.YahooMaxConcurrent)
	var wg sync.WaitGroup
	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			l.processSymbol(ctx, sym)
		}(symbol)
	}
	wg.Wait()
}

func (l *Loop) processSymbol(ctx context.Context, symbol string) {
	start := time.Now()
	quote, err := l.yahoo.FetchQuote(ctx, symbol)
	if err != nil {
		metrics.YahooErrors.Inc()
		l.logger.Warn("yahoo fetch failed", "symbol", symbol, "err", err)
		cached, cerr := l.cache.Get(ctx, symbol)
		if cerr != nil || cached == nil {
			if tick, derr := l.prices.Latest(ctx, symbol); derr == nil && tick != nil {
				quote = tickToQuote(tick, true)
			} else {
				return
			}
		} else {
			cached.Stale = true
			quote = cached
		}
		metrics.StalePrices.Inc()
	} else {
		metrics.YahooRequestDuration.Observe(time.Since(start).Seconds())
	}

	if err := l.cache.Set(ctx, *quote); err != nil {
		l.logger.Error("cache set", "symbol", symbol, "err", err)
	}

	tick := models.PriceTick{
		Time:     quote.Time,
		Symbol:   quote.Symbol,
		Price:    quote.Price,
		Open:     &quote.Open,
		High:     &quote.High,
		Low:      &quote.Low,
		Volume:   &quote.Volume,
		Currency: quote.Currency,
		Stale:    quote.Stale,
	}
	if err := l.prices.InsertTick(ctx, tick); err != nil {
		l.logger.Error("insert tick", "symbol", symbol, "err", err)
	}

	if err := l.publisher.PublishPrice(ctx, *quote); err != nil {
		l.logger.Error("publish price", "symbol", symbol, "err", err)
	}

	l.evaluateAlerts(ctx, *quote)
}

func (l *Loop) evaluateAlerts(ctx context.Context, quote models.Quote) {
	alerts, err := l.alerts.ListEnabledBySymbol(ctx, quote.Symbol)
	if err != nil {
		l.logger.Error("list alerts", "err", err)
		return
	}
	now := time.Now().UTC()
	for _, a := range alerts {
		ok, msg := alertengine.Evaluate(a, quote.Price, quote.Open, now)
		if !ok {
			continue
		}
		if err := l.alerts.MarkTriggered(ctx, a.ID, now); err != nil {
			l.logger.Error("mark triggered", "err", err)
			continue
		}
		ev, err := l.alerts.InsertEvent(ctx, a.SessionID, a.ID, quote.Symbol, quote.Price, msg)
		if err != nil {
			l.logger.Error("insert event", "err", err)
			continue
		}
		_ = l.alerts.TrimHistory(ctx, a.SessionID, 20)
		metrics.AlertsTriggered.Inc()

		_ = l.publisher.PublishAlert(ctx, a.SessionID, cache.AlertMessage{
			AlertID: a.ID,
			Symbol:  quote.Symbol,
			Price:   quote.Price.InexactFloat64(),
			Message: ev.Message,
		})
	}
}

func tickToQuote(t *models.PriceTick, stale bool) *models.Quote {
	open := t.Price
	if t.Open != nil {
		open = *t.Open
	}
	return &models.Quote{
		Symbol:   t.Symbol,
		Price:    t.Price,
		Open:     open,
		Currency: t.Currency,
		Stale:    stale,
		Time:     t.Time,
	}
}
