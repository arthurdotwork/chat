package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc"
	"github.com/arthurdotwork/chat/internal/adapters/secondary/broadcaster"
	"github.com/arthurdotwork/chat/internal/adapters/secondary/store"
	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/infrastructure/log"
	"github.com/arthurdotwork/chat/internal/infrastructure/redis"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Config(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		slog.DebugContext(ctx, "shutdown initiated")
		cancel()
	}()

	if err := run(ctx); err != nil {
		slog.ErrorContext(ctx, "error running server", "error", err)
	}
}

func run(ctx context.Context) error {
	grpcAddress := fmt.Sprintf(":%s", env("GRPC_PORT", "56000"))

	redisClient := redis.NewClient(env("REDIS_ADDR", "localhost:6379"))

	memoryRoomStore := store.NewMemoryRoomStore()
	redisBroadcaster := broadcaster.NewBroadcaster(redisClient)
	chatService := domain.NewChatService(memoryRoomStore, redisBroadcaster)
	chatServer := grpc.NewChatServer(chatService, grpcAddress)

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		slog.DebugContext(ctx, "starting grpc server", "address", grpcAddress)
		if err := chatServer.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "error running grpc server", "error", err)
			return fmt.Errorf("chatServer.Run: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		<-ctx.Done()

		if err := chatServer.Close(context.WithoutCancel(ctx)); err != nil {
			slog.ErrorContext(ctx, "error closing chat service", "error", err)
		}

		return nil
	})

	/*g.Go(func() error {
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
	})*/

	if err := g.Wait(); err != nil {
		slog.ErrorContext(ctx, "error running server", "error", err)
		return fmt.Errorf("errGroup.Wait: %w", err)
	}

	slog.DebugContext(ctx, "processing shutdown")
	return nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
