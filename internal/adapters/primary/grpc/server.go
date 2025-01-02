package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc/gen/proto"
	"github.com/arthurdotwork/chat/internal/adapters/secondary/messenger"
	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/google/uuid"
)

type ChatService interface {
	Join(ctx context.Context, user domain.User) (domain.User, error)
	SendMessage(ctx context.Context, message domain.Message) error
	Disconnect(ctx context.Context, user domain.User) error
}

type ChatServer struct {
	proto.UnimplementedChatServiceServer
	chatService ChatService
}

func NewChatServer(chatService ChatService) *ChatServer {
	return &ChatServer{
		chatService: chatService,
	}
}

func (s *ChatServer) Chat(stream proto.ChatService_ChatServer) error {
	ctx := stream.Context()

	var (
		sink          = make(chan error, 1)
		wg            sync.WaitGroup
		connectedUser *domain.User
	)

	slog.DebugContext(ctx, "client connected")
	messageManager := messenger.NewMessenger(stream)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}

				sink <- fmt.Errorf("error receiving message: %w", err)
				return
			}

			switch m := msg.Message.(type) {
			case *proto.ClientMessage_Join:
				if connectedUser != nil {
					continue
				}

				user := domain.User{
					ID:        uuid.New(),
					Name:      m.Join.UserName,
					Messenger: messageManager,
				}

				u, err := s.chatService.Join(ctx, user)
				if err != nil {
					sink <- fmt.Errorf("error joining chat: %w", err)
					return
				}

				connectedUser = &u

				_ = stream.Send(&proto.ServerMessage{
					Message: &proto.ServerMessage_JoinResponse{
						JoinResponse: &proto.JoinResponse{
							Success: true,
						},
					},
				})
			case *proto.ClientMessage_Chat:
				if connectedUser == nil {
					continue
				}

				message := domain.Message{
					Content: m.Chat.Content,
					Sender:  *connectedUser,
				}

				if err := s.chatService.SendMessage(ctx, message); err != nil {
					sink <- fmt.Errorf("error sending message: %w", err)
					return
				}
			default:
				slog.ErrorContext(ctx, "unknown message type", "message", m)
				sink <- fmt.Errorf("unknown message type")
			}
		}
	}()

	select {
	case <-ctx.Done():
		if connectedUser == nil {
			return nil
		}

		if err := s.chatService.Disconnect(ctx, *connectedUser); err != nil {
			slog.ErrorContext(ctx, "error disconnecting user", "error", err)
		}

		slog.DebugContext(ctx, "client disconnected")
		return nil
	case err := <-sink:
		slog.ErrorContext(ctx, "error handling message", "error", err)
		return fmt.Errorf("error handling message: %w", err)
	}
}
