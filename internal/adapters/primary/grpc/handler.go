package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc/gen/proto"
	"github.com/arthurdotwork/chat/internal/adapters/secondary/messenger"
	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type ChatService interface {
	Join(ctx context.Context, user domain.User) (domain.User, error)
	SendMessage(ctx context.Context, message domain.Message) error
	Disconnect(ctx context.Context, user domain.User) error
	Close(ctx context.Context) error
}

type ChatHandler struct {
	chatService ChatService
}

func NewChatHandler(chatService ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

func (s *ChatHandler) Chat(stream proto.ChatService_ChatServer) error {
	ctx := stream.Context()

	var (
		connectedUser *domain.User
	)

	messageManager := messenger.NewMessenger(stream)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					continue
				}

				return fmt.Errorf("error receiving message: %w", err)
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
					return fmt.Errorf("error joining chat: %w", err)
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
					return fmt.Errorf("error sending message: %w", err)
				}
			default:
				slog.ErrorContext(ctx, "unknown message type", "message", m)
				continue
			}
		}
	})

	<-ctx.Done()

	if connectedUser == nil {
		return nil
	}

	if err := s.chatService.Disconnect(ctx, *connectedUser); err != nil {
		slog.ErrorContext(ctx, "error disconnecting user", "error", err)
	}

	return nil
}
