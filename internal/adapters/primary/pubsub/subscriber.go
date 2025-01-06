package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/infrastructure/pubsub"
)

type LocalBroadcaster interface {
	Broadcast(ctx context.Context, m domain.Message) error
}

type Subscriber struct {
	subscriber              *pubsub.Subscriber
	localBroadcasterService LocalBroadcaster
}

func NewSubscriber(subscriber *pubsub.Subscriber, localBroadcasterService LocalBroadcaster) *Subscriber {
	return &Subscriber{
		subscriber:              subscriber,
		localBroadcasterService: localBroadcasterService,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context, channel string) {
	s.subscriber.Subscribe(ctx, channel, func(ctx context.Context, t pubsub.Task) error {
		var m domain.Message
		if err := json.Unmarshal(t.Payload, &m); err != nil {
			slog.ErrorContext(ctx, "json.Unmarshal", "error", err)
			return fmt.Errorf("json.Unmarshal: %w", err)
		}

		if err := s.localBroadcasterService.Broadcast(ctx, m); err != nil {
			return fmt.Errorf("localBroadcasterService.Broadcast: %w", err)
		}

		return nil
	})
}
