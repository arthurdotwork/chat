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
}
