package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "grpc-chat/proto"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewChatServiceClient(conn)
	stream, err := client.ChatStream(context.Background())
	if err != nil {
		log.Fatalf("Error creating chat stream: %v", err)
	}

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				log.Println("Server closed the stream")
				return
			}
			if err != nil {
				log.Fatalf("Error receiving response: %v", err)
			}
			fmt.Printf("Server response: %s\n", resp.Reply)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter your username:")
	scanner.Scan()
	username := scanner.Text()

	for {
		fmt.Print("Enter message: ")
		scanner.Scan()
		message := scanner.Text()

		if message == "exit" {
			log.Println("Exiting chat...")
			break
		}

		chatMessage := &pb.ChatMessage{
			User:    username,
			Message: message,
		}

		if err := stream.Send(chatMessage); err != nil {
			log.Fatalf("Error sending message: %v", err)
		}
	}

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Error closing stream: %v", err)
	}

	time.Sleep(time.Second)
}
