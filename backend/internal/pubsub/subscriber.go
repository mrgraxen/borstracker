package pubsub

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/graxe/borstracker/internal/cache"
	"github.com/redis/go-redis/v9"
)

type MessageHandler func(channel string, payload []byte)

type Subscriber struct {
	client  *redis.Client
	handler MessageHandler
}

func NewSubscriber(client *redis.Client, handler MessageHandler) *Subscriber {
	return &Subscriber{client: client, handler: handler}
}

func (s *Subscriber) Subscribe(ctx context.Context, patterns ...string) error {
	if len(patterns) == 0 {
		return nil
	}
	pubsub := s.client.PSubscribe(ctx, patterns...)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if msg.Payload == "" {
				continue
			}
			s.handler(msg.Channel, []byte(msg.Payload))
		}
	}
}

func ParsePricePayload(payload []byte) (*cache.CachedPrice, error) {
	var cp cache.CachedPrice
	if err := json.Unmarshal(payload, &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

func SymbolFromPriceChannel(channel string) string {
	// channel:price:AAPL
	parts := strings.Split(channel, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func SessionFromAlertChannel(channel string) string {
	parts := strings.Split(channel, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func LogHandler(logger *slog.Logger) MessageHandler {
	return func(channel string, payload []byte) {
		logger.Debug("pubsub message", "channel", channel, "len", len(payload))
	}
}
