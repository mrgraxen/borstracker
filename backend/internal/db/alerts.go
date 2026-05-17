package db

import (
	"context"
	"time"

	"github.com/graxe/borstracker/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type AlertRepo struct {
	pool *pgxpool.Pool
}

func NewAlertRepo(pool *pgxpool.Pool) *AlertRepo {
	return &AlertRepo{pool: pool}
}

func scanAlert(row interface {
	Scan(dest ...any) error
}) (*models.Alert, error) {
	var a models.Alert
	var threshold string
	var lastTriggered *time.Time
	err := row.Scan(
		&a.ID, &a.SessionID, &a.Symbol, &a.AlertType, &threshold,
		&a.Enabled, &a.CooldownSec, &lastTriggered, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	a.Threshold, _ = decimal.NewFromString(threshold)
	a.LastTriggered = lastTriggered
	return &a, nil
}

func (r *AlertRepo) List(ctx context.Context, sessionID, symbol string) ([]models.Alert, error) {
	q := `
		SELECT id, session_id, symbol, alert_type, threshold::text, enabled, cooldown_sec, last_triggered, created_at
		FROM alerts WHERE session_id = $1`
	args := []any{sessionID}
	if symbol != "" {
		q += ` AND symbol = $2`
		args = append(args, symbol)
	}
	q += ` ORDER BY id`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, *a)
	}
	return alerts, rows.Err()
}

func (r *AlertRepo) ListEnabledBySymbol(ctx context.Context, symbol string) ([]models.Alert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, session_id, symbol, alert_type, threshold::text, enabled, cooldown_sec, last_triggered, created_at
		FROM alerts WHERE symbol = $1 AND enabled = true
	`, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, *a)
	}
	return alerts, rows.Err()
}

func (r *AlertRepo) Create(ctx context.Context, sessionID, symbol string, alertType models.AlertType, threshold decimal.Decimal, cooldownSec int) (*models.Alert, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO alerts (session_id, symbol, alert_type, threshold, cooldown_sec)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, session_id, symbol, alert_type, threshold::text, enabled, cooldown_sec, last_triggered, created_at
	`, sessionID, symbol, alertType, threshold, cooldownSec)
	return scanAlert(row)
}

func (r *AlertRepo) Update(ctx context.Context, sessionID string, id int64, enabled *bool, threshold *decimal.Decimal, cooldownSec *int) (*models.Alert, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE alerts SET
			enabled = COALESCE($3, enabled),
			threshold = COALESCE($4, threshold),
			cooldown_sec = COALESCE($5, cooldown_sec)
		WHERE id = $1 AND session_id = $2
		RETURNING id, session_id, symbol, alert_type, threshold::text, enabled, cooldown_sec, last_triggered, created_at
	`, id, sessionID, enabled, threshold, cooldownSec)
	return scanAlert(row)
}

func (r *AlertRepo) Delete(ctx context.Context, sessionID string, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM alerts WHERE id = $1 AND session_id = $2`, id, sessionID)
	return err
}

func (r *AlertRepo) MarkTriggered(ctx context.Context, alertID int64, at time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE alerts SET last_triggered = $2 WHERE id = $1`, alertID, at)
	return err
}

func (r *AlertRepo) InsertEvent(ctx context.Context, sessionID string, alertID int64, symbol string, price decimal.Decimal, message string) (*models.AlertEvent, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO alert_events (session_id, alert_id, symbol, price, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, session_id, alert_id, symbol, price, message, triggered_at
	`, sessionID, alertID, symbol, price, message)

	var e models.AlertEvent
	var priceStr string
	err := row.Scan(&e.ID, &e.SessionID, &e.AlertID, &e.Symbol, &priceStr, &e.Message, &e.TriggeredAt)
	if err != nil {
		return nil, err
	}
	e.Price, _ = decimal.NewFromString(priceStr)
	return &e, nil
}

func (r *AlertRepo) TrimHistory(ctx context.Context, sessionID string, keep int) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM alert_events
		WHERE session_id = $1 AND id NOT IN (
			SELECT id FROM alert_events
			WHERE session_id = $1
			ORDER BY triggered_at DESC
			LIMIT $2
		)
	`, sessionID, keep)
	return err
}

func (r *AlertRepo) History(ctx context.Context, sessionID string, limit int) ([]models.AlertEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, session_id, alert_id, symbol, price::text, message, triggered_at
		FROM alert_events WHERE session_id = $1
		ORDER BY triggered_at DESC LIMIT $2
	`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.AlertEvent
	for rows.Next() {
		var e models.AlertEvent
		var priceStr string
		if err := rows.Scan(&e.ID, &e.SessionID, &e.AlertID, &e.Symbol, &priceStr, &e.Message, &e.TriggeredAt); err != nil {
			return nil, err
		}
		e.Price, _ = decimal.NewFromString(priceStr)
		events = append(events, e)
	}
	return events, rows.Err()
}
