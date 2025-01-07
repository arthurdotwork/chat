package domain

import (
	"context"
	"fmt"
	"log/slog"
)

type ChatService struct {
	roomStore   RoomStore
	broadcaster Broadcaster
}

func NewChatService(roomStore RoomStore, broadcaster Broadcaster) *ChatService {
	return &ChatService{
		roomStore:   roomStore,
		broadcaster: broadcaster,
	}
}

func (s *ChatService) Join(ctx context.Context, user User) (User, error) {
	if err := s.roomStore.Connect(ctx, user); err != nil {
		return User{}, fmt.Errorf("roomStore.Connect: %w", err)
	}

	if err := s.broadcaster.Broadcast(ctx, "chat-events", Message{
		Content: fmt.Sprintf("%s has joined the room", user.Name),
		Sender:  user,
	}); err != nil {
		return User{}, fmt.Errorf("broadcaster.Broadcast: %w", err)
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

func (s *ChatService) Close(ctx context.Context) error {
	connectedUsers, err := s.roomStore.GetConnectedUsers(ctx)
	if err != nil {
		return fmt.Errorf("roomStore.GetConnectedUsers: %w", err)
	}

	for _, u := range connectedUsers {
		if err := u.Messenger.SendServerClosingNotification(ctx); err != nil {
			slog.ErrorContext(ctx, "error sending server closing notification", "error", err)
		}
	}

	return nil
}

func (s *ChatService) Broadcast(ctx context.Context, message Message) error {
	connectedUsers, err := s.roomStore.GetConnectedUsers(ctx)
	if err != nil {
		return fmt.Errorf("roomStore.GetConnectedUsers: %w", err)
	}

	for _, u := range connectedUsers {
		if err := u.Messenger.SendMessage(ctx, message); err != nil {
			slog.ErrorContext(ctx, "error broadcasting message", "error", err)
		}
	}

	return nil
}
