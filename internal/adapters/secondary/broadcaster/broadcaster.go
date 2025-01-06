package broadcaster

import (
	"context"
	"fmt"

	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/infrastructure/pubsub"
)

type Broadcaster struct {
	publisher *pubsub.Publisher
}

func NewBroadcaster(publisher *pubsub.Publisher) *Broadcaster {
	return &Broadcaster{publisher: publisher}
}

func (b *Broadcaster) Broadcast(ctx context.Context, channel string, message domain.Message) error {
	if err := b.publisher.Publish(ctx, channel, message); err != nil {
		return fmt.Errorf("publisher.Publish: %w", err)
	}

	return nil
}
