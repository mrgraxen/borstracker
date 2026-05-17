package db

import (
	"context"
	"time"

	"github.com/graxe/borstracker/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type PriceRepo struct {
	pool *pgxpool.Pool
}

func NewPriceRepo(pool *pgxpool.Pool) *PriceRepo {
	return &PriceRepo{pool: pool}
}

func (r *PriceRepo) InsertTick(ctx context.Context, tick models.PriceTick) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO price_ticks (time, symbol, price, open, high, low, volume, currency, stale)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, tick.Time, tick.Symbol, tick.Price, tick.Open, tick.High, tick.Low, tick.Volume, tick.Currency, tick.Stale)
	return err
}

func (r *PriceRepo) Latest(ctx context.Context, symbol string) (*models.PriceTick, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT time, symbol, price::text, open::text, high::text, low::text, volume, currency, stale
		FROM price_ticks WHERE symbol = $1 ORDER BY time DESC LIMIT 1
	`, symbol)

	var tick models.PriceTick
	var priceStr string
	var openStr, highStr, lowStr *string
	var volume *int64
	err := row.Scan(&tick.Time, &tick.Symbol, &priceStr, &openStr, &highStr, &lowStr, &volume, &tick.Currency, &tick.Stale)
	if err != nil {
		return nil, err
	}
	tick.Price, _ = decimal.NewFromString(priceStr)
	if openStr != nil {
		o, _ := decimal.NewFromString(*openStr)
		tick.Open = &o
	}
	if highStr != nil {
		h, _ := decimal.NewFromString(*highStr)
		tick.High = &h
	}
	if lowStr != nil {
		l, _ := decimal.NewFromString(*lowStr)
		tick.Low = &l
	}
	tick.Volume = volume
	return &tick, nil
}

func (r *PriceRepo) Chart(ctx context.Context, symbol string, since time.Time) ([]models.ChartPoint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT time, price::float8 FROM price_ticks
		WHERE symbol = $1 AND time >= $2
		ORDER BY time ASC
	`, symbol, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []models.ChartPoint
	for rows.Next() {
		var p models.ChartPoint
		if err := rows.Scan(&p.Time, &p.Price); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
