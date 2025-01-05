package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc/gen/proto"
	"google.golang.org/grpc"
)

type ChatServer struct {
	proto.UnimplementedChatServiceServer
	chatService ChatService
	chatHandler *ChatHandler

	srv  *grpc.Server
	addr string
}

func NewChatServer(chatService ChatService, addr string) *ChatServer {
	srv := grpc.NewServer()
	chatHandler := NewChatHandler(chatService)

	chatServer := &ChatServer{
		chatService: chatService,
		chatHandler: chatHandler,
		srv:         srv,
		addr:        addr,
	}

	proto.RegisterChatServiceServer(srv, chatServer)
	return chatServer
}

func (s *ChatServer) Chat(stream proto.ChatService_ChatServer) error {
	return s.chatHandler.Chat(stream)
}

func (s *ChatServer) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		slog.ErrorContext(ctx, "error listening", "error", err)
		return fmt.Errorf("net.Listen: %w", err)
	}

	if err := s.srv.Serve(lis); err != nil {
		slog.ErrorContext(ctx, "error serving", "error", err)
		return fmt.Errorf("srv.Serve: %w", err)
	}

	return nil
}

func (s *ChatServer) Close(ctx context.Context) error {
	gracefulShutdownCompleted := make(chan bool, 1)
	go func() {
		if err := s.chatService.Close(ctx); err != nil {
			slog.ErrorContext(ctx, "error closing chat service", "error", err)
		}

		s.srv.GracefulStop()

		gracefulShutdownCompleted <- true
	}()

	for {
		select {
		case <-gracefulShutdownCompleted:
			return nil
		case <-time.After(15 * time.Second):
			s.srv.Stop()
			return nil
		}
	}
}
