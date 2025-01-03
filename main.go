package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc"
	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc/gen/proto"
	subscriber "github.com/arthurdotwork/chat/internal/adapters/primary/redis"
	"github.com/arthurdotwork/chat/internal/adapters/secondary/broadcaster"
	"github.com/arthurdotwork/chat/internal/adapters/secondary/store"
	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/infrastructure/log"
	"github.com/arthurdotwork/chat/internal/infrastructure/redis"
	grpcserver "google.golang.org/grpc"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Config(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		cancel()
	}()

	if err := run(ctx); err != nil {
		slog.ErrorContext(ctx, "error running server", "error", err)
	}
}

func run(ctx context.Context) error {
	redisClient := redis.NewClient(env("REDIS_ADDR", "localhost:6379"))

	memoryRoomStore := store.NewMemoryRoomStore()
	redisBroadcaster := broadcaster.NewBroadcaster(redisClient)
	chatService := domain.NewChatService(memoryRoomStore, redisBroadcaster)
	chatServer := grpc.NewChatServer(chatService)

	srv := grpcserver.NewServer()
	proto.RegisterChatServiceServer(srv, chatServer)

	sink := make(chan error, 1)

	go func() {
		addr := fmt.Sprintf(":%s", env("GRPC_PORT", "56000"))

		lis, err := net.Listen("tcp", addr)
		if err != nil {
			slog.ErrorContext(ctx, "error listening", "error", err)
			sink <- err
		}

		if err := srv.Serve(lis); err != nil {
			slog.ErrorContext(ctx, "error serving", "error", err)
			sink <- err
		}
	}()

	sub := subscriber.NewSubscriber(redisClient, chatService)
	go func() {
		if err := sub.Subscribe(ctx, "chat"); err != nil {
			slog.ErrorContext(ctx, "error subscribing", "error", err)
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

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
