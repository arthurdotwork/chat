package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/infrastructure/redis"
)

type LocalBroadcaster interface {
	Broadcast(ctx context.Context, m domain.Message) error
}

type Subscriber struct {
	redisClient             *redis.Client
	localBroadcasterService LocalBroadcaster
}

func NewSubscriber(redisClient *redis.Client, localBroadcasterService LocalBroadcaster) *Subscriber {
	return &Subscriber{
		redisClient:             redisClient,
		localBroadcasterService: localBroadcasterService,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context, channel string) error {
	subscriber := s.redisClient.Subscribe(ctx, channel)

	if err := subscriber(func(msg redis.Message) error {
		var m domain.Message
		if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
			return fmt.Errorf("json.Unmarshal: %w", err)
		}

		if err := s.localBroadcasterService.Broadcast(ctx, m); err != nil {
			return fmt.Errorf("localBroadcasterService.Broadcast: %w", err)
		}

		return nil
	}); err != nil {
		slog.ErrorContext(ctx, "error subscribing to redis", "error", err)
		return fmt.Errorf("subscriber: %w", err)
	}

	return nil
}
