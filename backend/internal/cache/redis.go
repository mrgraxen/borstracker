package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/graxe/borstracker/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

type PriceCache struct {
	client *redis.Client
	ttl    time.Duration
}

type CachedPrice struct {
	Symbol   string    `json:"symbol"`
	Price    float64   `json:"price"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Volume   int64     `json:"volume"`
	Currency string    `json:"currency"`
	Stale    bool      `json:"stale"`
	Time     time.Time `json:"time"`
}

func NewRedisClient(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opt), nil
}

func NewPriceCache(client *redis.Client, ttl time.Duration) *PriceCache {
	return &PriceCache{client: client, ttl: ttl}
}

func priceKey(symbol string) string {
	return "price:" + symbol
}

func (c *PriceCache) Set(ctx context.Context, q models.Quote) error {
	cp := CachedPrice{
		Symbol:   q.Symbol,
		Price:    q.Price.InexactFloat64(),
		Open:     q.Open.InexactFloat64(),
		High:     q.High.InexactFloat64(),
		Low:      q.Low.InexactFloat64(),
		Volume:   q.Volume,
		Currency: q.Currency,
		Stale:    q.Stale,
		Time:     q.Time,
	}
	b, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, priceKey(q.Symbol), b, c.ttl).Err()
}

func (c *PriceCache) Get(ctx context.Context, symbol string) (*models.Quote, error) {
	b, err := c.client.Get(ctx, priceKey(symbol)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cp CachedPrice
	if err := json.Unmarshal(b, &cp); err != nil {
		return nil, err
	}
	return &models.Quote{
		Symbol:   cp.Symbol,
		Price:    decimal.NewFromFloat(cp.Price),
		Open:     decimal.NewFromFloat(cp.Open),
		High:     decimal.NewFromFloat(cp.High),
		Low:      decimal.NewFromFloat(cp.Low),
		Volume:   cp.Volume,
		Currency: cp.Currency,
		Stale:    cp.Stale,
		Time:     cp.Time,
	}, nil
}

func (c *PriceCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func PriceChannel(symbol string) string {
	return fmt.Sprintf("channel:price:%s", symbol)
}

func AlertChannel(sessionID string) string {
	return fmt.Sprintf("channel:alert:%s", sessionID)
}

type Publisher struct {
	client *redis.Client
}

func NewPublisher(client *redis.Client) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) PublishPrice(ctx context.Context, q models.Quote) error {
	cp := CachedPrice{
		Symbol:   q.Symbol,
		Price:    q.Price.InexactFloat64(),
		Open:     q.Open.InexactFloat64(),
		High:     q.High.InexactFloat64(),
		Low:      q.Low.InexactFloat64(),
		Volume:   q.Volume,
		Currency: q.Currency,
		Stale:    q.Stale,
		Time:     q.Time,
	}
	b, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	return p.client.Publish(ctx, PriceChannel(q.Symbol), b).Err()
}

type AlertMessage struct {
	Type    string  `json:"type"`
	AlertID int64   `json:"alertId"`
	Symbol  string  `json:"symbol"`
	Price   float64 `json:"price"`
	Message string  `json:"message"`
}

func (p *Publisher) PublishAlert(ctx context.Context, sessionID string, msg AlertMessage) error {
	msg.Type = "alert"
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.client.Publish(ctx, AlertChannel(sessionID), b).Err()
}
