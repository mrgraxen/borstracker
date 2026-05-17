package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	client *redis.Client
	window time.Duration
}

func New(client *redis.Client) *Limiter {
	return &Limiter{
		client: client,
		window: time.Minute,
	}
}

// Allow checks a per-key counter against limit within a rolling minute window.
func (l *Limiter) Allow(ctx context.Context, key string, limit int) (bool, error) {
	if l.client == nil || limit <= 0 {
		return true, nil
	}
	pipe := l.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return true, err
	}
	count, err := incr.Result()
	if err != nil {
		return true, err
	}
	return count <= int64(limit), nil
}

func Key(sessionID, bucket string) string {
	return fmt.Sprintf("ratelimit:%s:%s", sessionID, bucket)
}
