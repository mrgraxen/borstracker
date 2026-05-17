package db

import (
	"context"

	"github.com/graxe/borstracker/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WatchlistRepo struct {
	pool *pgxpool.Pool
}

func NewWatchlistRepo(pool *pgxpool.Pool) *WatchlistRepo {
	return &WatchlistRepo{pool: pool}
}

func (r *WatchlistRepo) List(ctx context.Context, sessionID string) ([]models.WatchlistItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, session_id, symbol FROM watchlist_items
		WHERE session_id = $1 ORDER BY symbol
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.WatchlistItem
	for rows.Next() {
		var w models.WatchlistItem
		if err := rows.Scan(&w.ID, &w.SessionID, &w.Symbol); err != nil {
			return nil, err
		}
		items = append(items, w)
	}
	return items, rows.Err()
}

func (r *WatchlistRepo) Add(ctx context.Context, sessionID, symbol string) (*models.WatchlistItem, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO watchlist_items (session_id, symbol)
		VALUES ($1, $2)
		ON CONFLICT (session_id, symbol) DO UPDATE SET symbol = EXCLUDED.symbol
		RETURNING id, session_id, symbol
	`, sessionID, symbol)

	var w models.WatchlistItem
	err := row.Scan(&w.ID, &w.SessionID, &w.Symbol)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WatchlistRepo) Remove(ctx context.Context, sessionID, symbol string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM watchlist_items WHERE session_id = $1 AND symbol = $2
	`, sessionID, symbol)
	return err
}

func (r *WatchlistRepo) DistinctSymbols(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT symbol FROM watchlist_items
		UNION
		SELECT DISTINCT symbol FROM alerts WHERE enabled = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		symbols = append(symbols, s)
	}
	return symbols, rows.Err()
}
