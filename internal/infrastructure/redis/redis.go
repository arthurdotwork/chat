package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type Client struct {
	*redis.Client
}

func NewClient(addr string) *Client {
	return &Client{Client: redis.NewClient(&redis.Options{Addr: addr})}
}

type Message = redis.Message

var ErrFailedToReceiveMessage = errors.New("failed to receive message")

func (c *Client) Subscribe(ctx context.Context, channel string) func(handler func(Message) error) error {
	pubsub := c.Client.Subscribe(ctx, channel)

	return func(handler func(Message) error) error {
		defer pubsub.Close()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				msg, err := pubsub.Receive(ctx)
				if err != nil {
					return fmt.Errorf("pubsub.Receive: %w: %w", ErrFailedToReceiveMessage, err)
				}

				switch m := msg.(type) {
				case *Message:
					if err := handler(*m); err != nil {
						return fmt.Errorf("handler: %w", err)
					}
				}
			}
		}
	}
}

func (c *Client) Publish(ctx context.Context, channel string, message any) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	if err := c.Client.Publish(ctx, channel, msgBytes).Err(); err != nil {
		return fmt.Errorf("client.Publish: %w", err)
	}

	return nil
}
