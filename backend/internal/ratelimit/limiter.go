package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

func New(client *redis.Client, perMin int) *Limiter {
	return &Limiter{
		client: client,
		limit:  perMin,
		window: time.Minute,
	}
}

func (l *Limiter) Allow(ctx context.Context, sessionID string) (bool, error) {
	if l.client == nil {
		return true, nil
	}
	key := fmt.Sprintf("ratelimit:%s", sessionID)
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
	return count <= int64(l.limit), nil
}
