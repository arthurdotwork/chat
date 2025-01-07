package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/arthurdotwork/chat/internal/domain"
	p "github.com/arthurdotwork/chat/internal/infrastructure/pubsub"
	"github.com/google/uuid"
)

type LocalBroadcaster interface {
	Broadcast(ctx context.Context, m domain.Message) error
}

type Subscriber struct {
	subscriber              *p.Client
	localBroadcasterService LocalBroadcaster
}

func NewSubscriber(subscriber *p.Client, localBroadcasterService LocalBroadcaster) *Subscriber {
	return &Subscriber{
		subscriber:              subscriber,
		localBroadcasterService: localBroadcasterService,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context, channel string) error {
	if err := s.subscriber.Subscribe(ctx, channel, "ps"+uuid.NewString(), func(ctx context.Context, m *p.Message) error {
		var msg domain.Message
		if err := json.Unmarshal(m.Data, &msg); err != nil {
			slog.ErrorContext(ctx, "json.Unmarshal", "error", err)
			return fmt.Errorf("json.Unmarshal: %w", err)
		}

		if err := s.localBroadcasterService.Broadcast(ctx, msg); err != nil {
			return fmt.Errorf("localBroadcasterService.Broadcast: %w", err)
		}

		return nil
	}); err != nil {
		slog.ErrorContext(ctx, "subscriber.Subscribe", "error", err)
		return fmt.Errorf("subscriber.Subscribe: %w", err)
	}

	return nil
}
