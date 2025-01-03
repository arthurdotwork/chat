package domain

import (
	"context"

	"github.com/google/uuid"
)

type RoomStore interface {
	GetConnectedUsers(ctx context.Context) ([]User, error)
	Connect(ctx context.Context, user User) error
	Disconnect(ctx context.Context, user User) error
	GetUser(ctx context.Context, userID uuid.UUID) (User, error)
}

type Messenger interface {
	SendMessage(ctx context.Context, message Message) error
	SendServerClosingNotification(ctx context.Context) error
}

type Broadcaster interface {
	Broadcast(ctx context.Context, channel string, message Message) error
}
