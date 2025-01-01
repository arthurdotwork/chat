package messenger

import (
	"context"

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
