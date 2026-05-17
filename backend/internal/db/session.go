package db

import (
	"context"
	"time"

	"github.com/graxe/borstracker/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Upsert(ctx context.Context, id string) (*models.Session, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO sessions (id, last_seen_at)
		VALUES ($1, now())
		ON CONFLICT (id) DO UPDATE SET last_seen_at = now()
		RETURNING id, language, sound_id, sound_enabled, created_at, last_seen_at
	`, id)

	var s models.Session
	err := row.Scan(&s.ID, &s.Language, &s.SoundID, &s.SoundEnabled, &s.CreatedAt, &s.LastSeenAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) Get(ctx context.Context, id string) (*models.Session, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, language, sound_id, sound_enabled, created_at, last_seen_at
		FROM sessions WHERE id = $1
	`, id)

	var s models.Session
	err := row.Scan(&s.ID, &s.Language, &s.SoundID, &s.SoundEnabled, &s.CreatedAt, &s.LastSeenAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) UpdateSettings(ctx context.Context, id, language string, soundID int16, soundEnabled bool) (*models.Session, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE sessions
		SET language = $2, sound_id = $3, sound_enabled = $4, last_seen_at = now()
		WHERE id = $1
		RETURNING id, language, sound_id, sound_enabled, created_at, last_seen_at
	`, id, language, soundID, soundEnabled)

	var s models.Session
	err := row.Scan(&s.ID, &s.Language, &s.SoundID, &s.SoundEnabled, &s.CreatedAt, &s.LastSeenAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) CountActive(ctx context.Context, since time.Duration) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM sessions WHERE last_seen_at > now() - ($1 || ' seconds')::interval
	`, int64(since.Seconds())).Scan(&n)
	return n, err
}
