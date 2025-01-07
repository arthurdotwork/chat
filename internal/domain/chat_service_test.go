package domain_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChatService_Join(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomStore := mocks.NewMockRoomStore(t)
	broadcaster := mocks.NewMockBroadcaster(t)
	chatService := domain.NewChatService(roomStore, broadcaster)

	user := domain.User{
		ID:   uuid.New(),
		Name: "arthur",
	}

	t.Run("it should return an error if it can not add the user to the current room", func(t *testing.T) {
		roomStore.On("Connect", ctx, user).Return(fmt.Errorf("error")).Once()

		_, err := chatService.Join(ctx, user)
		require.Error(t, err)
	})

	t.Run("it should return an error if it can not broadcast the message", func(t *testing.T) {
		roomStore.On("Connect", ctx, user).Return(nil).Once()
		broadcaster.On("Broadcast", ctx, "chat-events", domain.Message{
			Content: "arthur has joined the room",
			Sender:  user,
		}).Return(fmt.Errorf("error")).Once()

		_, err := chatService.Join(ctx, user)
		require.Error(t, err)
	})

	t.Run("it should allow one user to join", func(t *testing.T) {
		roomStore.On("Connect", ctx, user).Return(nil).Once()
		broadcaster.On("Broadcast", ctx, "chat-events", domain.Message{
			Content: "arthur has joined the room",
			Sender:  user,
		}).Return(nil).Once()

		connectedUser, err := chatService.Join(ctx, user)
		require.NoError(t, err)
		require.Equal(t, user, connectedUser)
	})
}

func TestChatService_SendMessage(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomStore := mocks.NewMockRoomStore(t)
	broadcaster := mocks.NewMockBroadcaster(t)
	messenger := mocks.NewMockMessenger(t)
	chatService := domain.NewChatService(roomStore, broadcaster)

	message := domain.Message{
		ID:      uuid.New(),
		Content: "hello world",
		Sender: domain.User{
			ID:        uuid.New(),
			Name:      "arthur",
			Messenger: messenger,
		},
	}

	t.Run("it should return an error if it can not get the sender user", func(t *testing.T) {
		roomStore.On("GetUser", ctx, message.Sender.ID).Return(domain.User{}, fmt.Errorf("error")).Once()

		err := chatService.SendMessage(ctx, message)
		require.Error(t, err)
	})

	t.Run("it should return an error if it can not get the connected users", func(t *testing.T) {
		roomStore.On("GetUser", ctx, message.Sender.ID).Return(domain.User{}, nil).Once()
		roomStore.On("GetConnectedUsers", ctx).Return(nil, fmt.Errorf("error")).Once()

		err := chatService.SendMessage(ctx, message)
		require.Error(t, err)
	})

	t.Run("it should return an error if it can not send the message to a user", func(t *testing.T) {
		roomStore.On("GetUser", ctx, message.Sender.ID).Return(domain.User{}, nil).Once()
		roomStore.On("GetConnectedUsers", ctx).Return([]domain.User{
			{
				ID:        uuid.New(),
				Name:      "john",
				Messenger: messenger,
			},
		}, nil).Once()
		messenger.On("SendMessage", ctx, message).Return(fmt.Errorf("error")).Once()

		err := chatService.SendMessage(ctx, message)
		require.Error(t, err)
	})

	t.Run("it should send the message to all connected users", func(t *testing.T) {
		roomStore.On("GetUser", ctx, message.Sender.ID).Return(domain.User{}, nil).Once()
		roomStore.On("GetConnectedUsers", ctx).Return([]domain.User{
			{
				ID:        uuid.New(),
				Name:      "john",
				Messenger: messenger,
			},
		}, nil).Once()
		messenger.On("SendMessage", ctx, message).Return(nil).Once()

		err := chatService.SendMessage(ctx, message)
		require.NoError(t, err)
	})
}

func TestChatService_Disconnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomStore := mocks.NewMockRoomStore(t)
	broadcaster := mocks.NewMockBroadcaster(t)
	messenger := mocks.NewMockMessenger(t)
	chatService := domain.NewChatService(roomStore, broadcaster)

	user := domain.User{
		ID:        uuid.New(),
		Name:      "arthur",
		Messenger: messenger,
	}

	t.Run("it should return an error if it can not disconnect the user", func(t *testing.T) {
		roomStore.On("Disconnect", ctx, user).Return(fmt.Errorf("error")).Once()

		err := chatService.Disconnect(ctx, user)
		require.Error(t, err)
	})

	t.Run("it should return an error if it can not get the connected users", func(t *testing.T) {
		roomStore.On("Disconnect", ctx, user).Return(nil).Once()
		roomStore.On("GetConnectedUsers", ctx).Return(nil, fmt.Errorf("error")).Once()

		err := chatService.Disconnect(ctx, user)
		require.Error(t, err)
	})

	t.Run("it should return an error if it can not send the message to a user", func(t *testing.T) {
		roomStore.On("Disconnect", ctx, user).Return(nil).Once()
		roomStore.On("GetConnectedUsers", ctx).Return([]domain.User{
			{
				ID:        uuid.New(),
				Name:      "john",
				Messenger: messenger,
			},
		}, nil).Once()
		messenger.On("SendMessage", ctx, domain.Message{
			Content: "arthur has left the room",
			Sender:  user,
		}).Return(fmt.Errorf("error")).Once()

		err := chatService.Disconnect(ctx, user)
		require.Error(t, err)
	})

	t.Run("it should send the message to all connected users", func(t *testing.T) {
		roomStore.On("Disconnect", ctx, user).Return(nil).Once()
		roomStore.On("GetConnectedUsers", ctx).Return([]domain.User{
			{
				ID:        uuid.New(),
				Name:      "john",
				Messenger: messenger,
			},
		}, nil).Once()
		messenger.On("SendMessage", ctx, domain.Message{
			Content: "arthur has left the room",
			Sender:  user,
		}).Return(nil).Once()

		err := chatService.Disconnect(ctx, user)
		require.NoError(t, err)
	})
}

func TestChatService_Close(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomStore := mocks.NewMockRoomStore(t)
	broadcaster := mocks.NewMockBroadcaster(t)
	messenger := mocks.NewMockMessenger(t)
	chatService := domain.NewChatService(roomStore, broadcaster)

	t.Run("it should return an error if it can not get the connected users", func(t *testing.T) {
		roomStore.On("GetConnectedUsers", ctx).Return(nil, fmt.Errorf("error")).Once()

		err := chatService.Close(ctx)
		require.Error(t, err)
	})

	t.Run("it should send the message to all connected users", func(t *testing.T) {
		roomStore.On("GetConnectedUsers", ctx).Return([]domain.User{
			{
				ID:        uuid.New(),
				Name:      "arthur",
				Messenger: messenger,
			},
		}, nil).Once()

		messenger.On("SendServerClosingNotification", ctx).Return(nil).Once()

		err := chatService.Close(ctx)
		require.NoError(t, err)
	})
}

func TestChatService_Broadcast(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomStore := mocks.NewMockRoomStore(t)
	broadcaster := mocks.NewMockBroadcaster(t)
	messenger := mocks.NewMockMessenger(t)
	chatService := domain.NewChatService(roomStore, broadcaster)

	user := domain.User{
		ID:        uuid.New(),
		Name:      "arthur",
		Messenger: messenger,
	}

	t.Run("it should return an error if it can not get the connected users", func(t *testing.T) {
		roomStore.On("GetConnectedUsers", ctx).Return(nil, fmt.Errorf("error")).Once()

		err := chatService.Broadcast(ctx, domain.Message{
			Content: "hello world",
			Sender:  user,
		})
		require.Error(t, err)
	})

	t.Run("it should send the message to all connected users", func(t *testing.T) {
		roomStore.On("GetConnectedUsers", ctx).Return([]domain.User{
			user,
			{
				ID:        uuid.New(),
				Name:      "john",
				Messenger: messenger,
			},
		}, nil).Once()

		messenger.On("SendMessage", ctx, domain.Message{
			Content: "hello world",
			Sender:  user,
		}).Return(nil).Twice()

		err := chatService.Broadcast(ctx, domain.Message{
			Content: "hello world",
			Sender:  user,
		})
		require.NoError(t, err)
	})
}
