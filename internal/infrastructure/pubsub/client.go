package pubsub

import (
	"context"
	"log/slog"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Message = pubsub.Message

type Client struct {
	*pubsub.Client
}

func NewClient(ctx context.Context, projectID string, credentialsFilePath string) (*Client, error) {
	client, err := pubsub.NewClient(ctx, projectID, option.WithCredentialsFile(credentialsFilePath))
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		time.Sleep(time.Second * 5)
		if err := client.Close(); err != nil {
			slog.ErrorContext(ctx, "client.Close", "error", err)
		}
	}()

	return &Client{
		client,
	}, nil
}

type SubscriberHandler func(ctx context.Context, m *pubsub.Message) error

func (c *Client) Subscribe(ctx context.Context, topic string, sub string, handler SubscriberHandler) error {
	go func() {
		<-ctx.Done()
		if err := c.Subscription(sub).Delete(context.WithoutCancel(ctx)); err != nil {
			slog.ErrorContext(ctx, "Subscription.Delete", "error", err)
		}
	}()

	s, err := c.CreateSubscription(ctx, sub, pubsub.SubscriptionConfig{
		Topic:                     c.Topic(topic),
		EnableExactlyOnceDelivery: true,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() != codes.AlreadyExists {
			slog.DebugContext(ctx, "CreateSubscription", "error", err)
			return err
		}

		if !ok {
			slog.DebugContext(ctx, "CreateSubscription", "error", err)
			return err
		}

		s = c.Subscription(sub)
	}

	slog.DebugContext(ctx, "subscribed", "topic", topic, "sub", sub)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := s.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
				if err := handler(ctx, m); err != nil {
					slog.ErrorContext(ctx, "handler", "error", err)
					m.Nack()
					return
				}

				m.Ack()
			}); err != nil {
				slog.DebugContext(ctx, "Receive", "error", err)
				return err
			}

			return nil
		}
	}
}

func (c *Client) Publish(ctx context.Context, topic string, m *pubsub.Message) error {
	t := c.Topic(topic)
	r := t.Publish(ctx, m)
	if _, err := r.Get(ctx); err != nil {
		slog.ErrorContext(ctx, "Publish", "error", err)
		return err
	}

	return nil
}
