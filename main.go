package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/arthurdotwork/chat/cmd"
	"github.com/arthurdotwork/chat/internal/infrastructure/log"

	"github.com/spf13/cobra"
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
		RunE: func(c *cobra.Command, args []string) error {
			if err := cmd.Server(ctx, c); err != nil {
				slog.ErrorContext(ctx, "error starting server", "error", err)
				return err
			}

			return nil
		},
	}

	rootCmd.AddCommand(serverCmd)

	if err := rootCmd.Execute(); err != nil {
		slog.ErrorContext(ctx, "error executing command", "error", err)
	}
}
