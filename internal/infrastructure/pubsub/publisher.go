package pubsub

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

type Publisher struct {
	client *asynq.Client
}

func NewPublisher(redisAddr string) *Publisher {
	return &Publisher{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}),
	}
}

func (p *Publisher) Publish(ctx context.Context, t string, message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	task := asynq.NewTask(t, payload)
	if _, err := p.client.Enqueue(task); err != nil {
		return err
	}

	return nil
}

func (p *Publisher) Close() error {
	return p.client.Close()
}
