package main

import (
	"fmt"
	"io"
	"log"
	"net"

	pb "grpc-chat/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedChatServiceServer
}

func (s *server) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	log.Println("ChatStream started")

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Println("Client closed the stream")
			return nil
		}
		if err != nil {
			log.Printf("Error receiving message from client: %v", err)
			return err
		}

		log.Printf("Received message from %s: %s", in.User, in.Message)

		reply := &pb.ChatResponse{Reply: fmt.Sprintf("Hello %s, I received your message: %s", in.User, in.Message)}
		if err := stream.Send(reply); err != nil {
			log.Printf("Error sending message to client: %v", err)
			return err
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatServiceServer(s, &server{})

	log.Println("Server is listening on port 50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
