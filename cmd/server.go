package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc"
	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc/gen/proto"
	"github.com/arthurdotwork/chat/internal/adapters/secondary/store"
	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/spf13/cobra"
	grpcserver "google.golang.org/grpc"
)

func Server(ctx context.Context, cmd *cobra.Command) error {
	memoryRoomStore := store.NewMemoryRoomStore()
	chatService := domain.NewChatService(memoryRoomStore)
	chatServer := grpc.NewChatServer(chatService)

	srv := grpcserver.NewServer()
	proto.RegisterChatServiceServer(srv, chatServer)

	addr := fmt.Sprintf(":%d", 56001)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	sink := make(chan error, 1)

	go func() {
		if err := srv.Serve(lis); err != nil {
			slog.ErrorContext(ctx, "error serving", "error", err)
			sink <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.DebugContext(ctx, "initiating server shutdown")

		go func() {
			srv.GracefulStop()
		}()

		// Channel to signal shutdown completion
		done := make(chan struct{})

		if err := chatService.Close(context.WithoutCancel(ctx), done); err != nil {
			slog.ErrorContext(ctx, "error closing chat service", "error", err)
		}

		<-done

		// Create a timeout context for graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		<-shutdownCtx.Done()
		srv.Stop()
	case err := <-sink:
		return fmt.Errorf("srv.Serve: %w", err)
	}

	return nil
}
