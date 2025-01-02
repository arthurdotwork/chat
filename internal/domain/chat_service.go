package domain

import (
	"context"
	"fmt"
	"log/slog"
)

type ChatService struct {
	roomStore RoomStore
}

func NewChatService(roomStore RoomStore) *ChatService {
	return &ChatService{roomStore: roomStore}
}

func (s *ChatService) Join(ctx context.Context, user User) (User, error) {
	if err := s.roomStore.Connect(ctx, user); err != nil {
		return User{}, fmt.Errorf("roomStore.Connect: %w", err)
	}

	connectedUsers, err := s.roomStore.GetConnectedUsers(ctx)
	if err != nil {
		return User{}, fmt.Errorf("roomStore.GetConnectedUsers: %w", err)
	}

	slog.DebugContext(ctx, "dispatching message to connected users", "connected_users", len(connectedUsers))

	for _, u := range connectedUsers {
		if u.ID == user.ID {
			continue
		}

		slog.DebugContext(ctx, "dispatching message to user", "user", u.Name)

		if err := u.Messenger.SendMessage(ctx, Message{
			Content: fmt.Sprintf("%s has joined the room", user.Name),
			Sender:  user,
		}); err != nil {
			return User{}, fmt.Errorf("messenger.SendMessage: %w", err)
		}
	}

	return user, nil
}

func (s *ChatService) SendMessage(ctx context.Context, message Message) error {
	sender, err := s.roomStore.GetUser(ctx, message.Sender.ID)
	if err != nil {
		return fmt.Errorf("roomStore.GetUser: %w", err)
	}

	connectedUsers, err := s.roomStore.GetConnectedUsers(ctx)
	if err != nil {
		return fmt.Errorf("roomStore.GetConnectedUsers: %w", err)
	}

	for _, u := range connectedUsers {
		if u.ID == sender.ID {
			continue
		}

		if err := u.Messenger.SendMessage(ctx, message); err != nil {
			return fmt.Errorf("messenger.SendMessage: %w", err)
		}
	}

	return nil
}

func (s *ChatService) Disconnect(ctx context.Context, user User) error {
	if err := s.roomStore.Disconnect(ctx, user); err != nil {
		return fmt.Errorf("roomStore.Disconnect: %w", err)
	}

	connectedUsers, err := s.roomStore.GetConnectedUsers(ctx)
	if err != nil {
		return fmt.Errorf("roomStore.GetConnectedUsers: %w", err)
	}

	for _, u := range connectedUsers {
		if u.ID == user.ID {
			continue
		}

		if err := u.Messenger.SendMessage(ctx, Message{
			Content: fmt.Sprintf("%s has left the room", user.Name),
			Sender:  user,
		}); err != nil {
			return fmt.Errorf("messenger.SendMessage: %w", err)
		}
	}

	return nil
}
