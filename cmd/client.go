package cmd

import (
	"context"
	"fmt"

	"github.com/arthurdotwork/chat/internal/adapters/primary/grpc/gen/proto"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Client(ctx context.Context, c *cobra.Command) error {
	conn, err := grpc.NewClient("localhost:56001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("grpc.NewClient: %w", err)
	}
	defer conn.Close()

	// Create client
	client := proto.NewChatServiceClient(conn)
	stream, err := client.Chat(ctx)
	if err != nil {
		return fmt.Errorf("client.Chat: %w", err)
	}

	prompt := promptui.Prompt{Label: "Username"}
	username, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("prompt.Run: %w", err)
	}

	if err := stream.Send(&proto.ClientMessage{
		Message: &proto.ClientMessage_Join{
			Join: &proto.JoinRoom{
				UserName: username,
			},
		},
	}); err != nil {
		return fmt.Errorf("stream.Send: %w", err)
	}

	sink := make(chan error, 1)

	go receiveMessages(ctx, stream, sink)

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sink:
			if err != nil {
				return fmt.Errorf("receiveMessages: %w", err)
			}

			return nil
		}
	}
}

func receiveMessages(
	ctx context.Context,
	stream proto.ChatService_ChatClient,
	sink chan error,
) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return
		}

		switch m := msg.Message.(type) {
		case *proto.ServerMessage_ServerClosing:
			fmt.Printf("Server is closing: %s\n", m.ServerClosing.Message)
			sink <- nil
		case *proto.ServerMessage_Chat:
			fmt.Printf("%s: %s\n", m.Chat.UserName, m.Chat.Content)
		case *proto.ServerMessage_JoinResponse:
			fmt.Printf("You joined the room\n")
		default:
			sink <- fmt.Errorf("unknown message type: %T", m)
		}
	}
}
