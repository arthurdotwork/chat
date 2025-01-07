package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc"
	ppubsub "github.com/arthurdotwork/chat/internal/adapters/primary/pubsub"
	"github.com/arthurdotwork/chat/internal/adapters/secondary/broadcaster"
	"github.com/arthurdotwork/chat/internal/adapters/secondary/store"
	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/infrastructure/log"
	ps "github.com/arthurdotwork/chat/internal/infrastructure/pubsub"
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

	pubsubClient, err := ps.NewClient(ctx, env("GCP_PROJECT_ID", "arthur-private"), env("GCP_CREDENTIALS_FILE_PATH", ".pubsub_service_account.json"))
	if err != nil {
		slog.ErrorContext(ctx, "error creating pubsub client", "error", err)
		return fmt.Errorf("ps.NewClient: %w", err)
	}

	memoryRoomStore := store.NewMemoryRoomStore()
	b := broadcaster.NewBroadcaster(pubsubClient)
	chatService := domain.NewChatService(memoryRoomStore, b)
	chatServer := grpc.NewChatServer(chatService, grpcAddress)

	chatSubscriber := ppubsub.NewSubscriber(pubsubClient, chatService)

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := chatSubscriber.Subscribe(ctx, "chat-events"); err != nil {
			slog.ErrorContext(ctx, "error subscribing to chat", "error", err)
			return fmt.Errorf("chatSubscriber.Subscribe: %w", err)
		}

		return nil
	})

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

	if err := g.Wait(); err != nil {
		slog.ErrorContext(ctx, "error running server", "error", err)
		return fmt.Errorf("errGroup.Wait: %w", err)
	}

	slog.DebugContext(ctx, "processing shutdown")
	time.Sleep(time.Second * 10)
	return nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
