package main

import (
	"context"
	pb "github.com/hjani-2003/Cloud_Computing_Project/tree/harman/part2/proto"
)

func (s *helloServer) SayHello(ctx context.Context, req *pb.NoParam) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{
		Message: "Main kheta hello",
	}, nil
}