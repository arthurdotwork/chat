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
	"github.com/arthurdotwork/chat/internal/infrastructure/runner"
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
		slog.DebugContext(ctx, "received signal, initiating shutdown")
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

	r := runner.New(ctx)
	r.Go(func() error {
		errCh := make(chan error, 1)

		go func() {
			proto.RegisterChatServiceServer(srv, chatServer)

			addr := fmt.Sprintf(":%s", env("GRPC_PORT", "56000"))

			slog.DebugContext(ctx, "starting server", "address", addr)

			lis, err := net.Listen("tcp", addr)
			if err != nil {
				slog.ErrorContext(ctx, "error listening", "error", err)
				errCh <- fmt.Errorf("net.Listen: %w", err)
				return
			}

			if err := srv.Serve(lis); err != nil {
				slog.ErrorContext(ctx, "error serving", "error", err)
				errCh <- fmt.Errorf("srv.Serve: %w", err)
				return
			}

			slog.DebugContext(ctx, "server stopped", "address", addr)
			errCh <- nil
		}()

		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "context done, stopping server")
			return ctx.Err()
		case err := <-errCh:
			return err
		}
	})

	r.Go(func() error {
		sub := subscriber.NewSubscriber(redisClient, chatService)
		errCh := make(chan error, 1)

		go func() {
			errCh <- sub.Subscribe(ctx, "chat")
		}()

		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "context done, stopping subscriber")
			return ctx.Err()
		case err := <-errCh:
			if err != nil {
				slog.ErrorContext(ctx, "error subscribing", "error", err)
				return fmt.Errorf("sub.Subscribe: %w", err)
			}
		}

		slog.DebugContext(ctx, "subscriber stopped")
		return nil
	})

	if err := r.Wait(); err != nil {
		slog.ErrorContext(ctx, "error running server", "error", err)
		return fmt.Errorf("errGroup.Wait: %w", err)
	}

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

	return nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
