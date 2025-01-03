package broadcaster

import (
	"context"

	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/infrastructure/redis"
)

type Broadcaster struct {
	redisClient *redis.Client
}

func NewBroadcaster(redisClient *redis.Client) *Broadcaster {
	return &Broadcaster{redisClient: redisClient}
}

func (b *Broadcaster) Broadcast(ctx context.Context, channel string, message domain.Message) error {
	return b.redisClient.Publish(ctx, channel, message)
}
