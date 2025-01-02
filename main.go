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
	"github.com/arthurdotwork/chat/internal/adapters/secondary/store"
	"github.com/arthurdotwork/chat/internal/domain"
	"github.com/arthurdotwork/chat/internal/infrastructure/log"

	"github.com/spf13/cobra"
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

	rootCmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat is a simple chat application",
	}
	rootCmd.PersistentFlags().BoolP("help", "", false, "help for this command")

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start the chat server",
		Run: func(cmd *cobra.Command, args []string) {
			memoryRoomStore := store.NewMemoryRoomStore()
			chatService := domain.NewChatService(memoryRoomStore)

			chatServer := grpc.NewChatServer(chatService)

			srv := grpcserver.NewServer()
			proto.RegisterChatServiceServer(srv, chatServer)

			addr := fmt.Sprintf(":%d", 56001)
			lis, err := net.Listen("tcp", addr)
			if err != nil {
				slog.ErrorContext(ctx, "error listening", "error", err)
				return
			}

			sink := make(chan error, 1)

			slog.DebugContext(ctx, "starting server", "address", addr)

			go func() {
				slog.DebugContext(ctx, "server started", "address", addr)
				if err := srv.Serve(lis); err != nil {
					slog.ErrorContext(ctx, "error serving", "error", err)
					sink <- err
				}
			}()

			go func() {
				tickEachSecond := time.NewTicker(1 * time.Second)
				defer tickEachSecond.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-tickEachSecond.C:
						conn, err := grpcserver.DialContext(ctx, addr, grpcserver.WithInsecure())
						if err != nil {
							slog.ErrorContext(ctx, "error dialing", "error", err)
							return
						}
						conn.Close()

						connectedUsers, err := memoryRoomStore.GetConnectedUsers(ctx)
						if err != nil {
							slog.ErrorContext(ctx, "error getting connected users", "error", err)
							continue
						}

						slog.DebugContext(ctx, "connected users", "count", len(connectedUsers))
					}
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
				slog.ErrorContext(ctx, "error serving", "error", err)
			}
		},
	}

	rootCmd.AddCommand(serverCmd)

	if err := rootCmd.Execute(); err != nil {
		slog.ErrorContext(ctx, "error executing command", "error", err)
	}
}
