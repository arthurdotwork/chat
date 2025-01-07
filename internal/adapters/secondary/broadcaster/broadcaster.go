package broadcaster

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/infrastructure/pubsub"
)

type Broadcaster struct {
	publisher *pubsub.Client
}

func NewBroadcaster(publisher *pubsub.Client) *Broadcaster {
	return &Broadcaster{publisher: publisher}
}

func (b *Broadcaster) Broadcast(ctx context.Context, channel string, message domain.Message) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	if err := b.publisher.Publish(ctx, channel, &pubsub.Message{Data: payload}); err != nil {
		return fmt.Errorf("publisher.Publish: %w", err)
	}

	return nil
}
