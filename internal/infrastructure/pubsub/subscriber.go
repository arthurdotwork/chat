package pubsub

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
)

type Subscriber struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

func NewSubscriber(redisAddr string) *Subscriber {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			TaskCheckInterval: time.Millisecond * 100,
			// Number of concurrent workers
			Concurrency: 100,

			// Graceful shutdown timeout
			ShutdownTimeout: 30 * time.Second,

			// Queue priorities
			Queues: map[string]int{
				"default": 10,
			},
			Logger: NewNilLogger(),
		},
	)

	return &Subscriber{
		srv: srv,
		mux: asynq.NewServeMux(),
	}
}

type TaskHandlerFunc func(ctx context.Context, t Task) error

func (c *Subscriber) Subscribe(ctx context.Context, topic string, handler func(ctx context.Context, t Task) error) {
	c.mux.HandleFunc(topic, func(ctx context.Context, t *asynq.Task) error {
		if err := handler(ctx, Task{Type: t.Type(), Payload: t.Payload()}); err != nil {
			return err
		}

		return nil
	})
}

func (c *Subscriber) Start() error {
	return c.srv.Start(c.mux)
}

func (c *Subscriber) Stop() {
	c.srv.Stop()
}
