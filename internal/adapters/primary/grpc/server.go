package grpc

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc/gen/proto"
)

type ChatServer struct {
	proto.UnimplementedChatServiceServer
}

func NewChatServer() *ChatServer {
	return &ChatServer{}
}

func (s *ChatServer) Chat(stream proto.ChatService_ChatServer) error {
	ctx := stream.Context()

	var (
		joinInfo *proto.JoinRoom
		sink     = make(chan error, 1)
		wg       sync.WaitGroup
	)

	slog.DebugContext(ctx, "client connected")

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}

				slog.Debug("error receiving message here", "error", err)
				sink <- err
				return
			}

			slog.Debug("received message", "message", msg)

			switch m := msg.Message.(type) {
			case *proto.ClientMessage_Join:
				if joinInfo != nil {
					slog.DebugContext(ctx, "client already joined", "client", joinInfo.UserName)
					continue
				}

				joinInfo = m.Join

				slog.DebugContext(ctx, "client joined", "client", joinInfo.UserName)

				_ = stream.Send(&proto.ServerMessage{
					Message: &proto.ServerMessage_JoinResponse{
						JoinResponse: &proto.JoinResponse{
							Success: true,
						},
					},
				})
			case *proto.ClientMessage_Chat:
				if joinInfo == nil {
					slog.ErrorContext(ctx, "client not joined")
					continue
				}

				slog.DebugContext(ctx, "client sent message", "client", joinInfo.UserName, "message", m.Chat.Content)

				msg := &proto.ServerMessage_Chat{
					Chat: &proto.ChatMessage{
						UserName: joinInfo.UserName,
						Content:  m.Chat.Content,
					},
				}

				_ = stream.Send(&proto.ServerMessage{
					Message: msg,
				})
			default:
				slog.ErrorContext(ctx, "unknown message type", "message", m)
				sink <- fmt.Errorf("unknown message type")
			}
		}
	}()

	select {
	case <-ctx.Done():
		slog.DebugContext(ctx, "client disconnected")
		return nil
	case err := <-sink:
		slog.ErrorContext(ctx, "error receiving message", "error", err)
		return err
	}
}
