package messenger

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc/gen/proto"
	"github.com/arthurdotwork/chat/internal/domain"
)

type Messenger struct {
	Stream proto.ChatService_ChatServer
}

func NewMessenger(stream proto.ChatService_ChatServer) *Messenger {
	return &Messenger{Stream: stream}
}

func (m *Messenger) SendMessage(ctx context.Context, msg domain.Message) error {
	protoMessage := &proto.ServerMessage_Chat{
		Chat: &proto.ChatMessage{
			UserName: msg.Sender.Name,
			Content:  msg.Content,
		},
	}

	return m.Stream.Send(&proto.ServerMessage{Message: protoMessage})
}

func (m *Messenger) SendServerClosingNotification(ctx context.Context) error {
	msg := &proto.ServerMessage{
		Message: &proto.ServerMessage_ServerClosing{
			ServerClosing: &proto.ServerClosing{
				Message: "server is closing",
			},
		},
	}

	slog.DebugContext(ctx, "sending server closing notification",
		"message", fmt.Sprintf("%+v", msg),
		"message_type", fmt.Sprintf("%T", msg.Message))

	if err := m.Stream.Send(msg); err != nil {
		slog.ErrorContext(ctx, "failed to send server closing", "error", err)
		return fmt.Errorf("stream.Send: %w", err)
	}

	slog.DebugContext(ctx, "server closing notification sent")
	return nil
}
