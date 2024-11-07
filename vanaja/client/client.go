package main

import (
	"context"
	"io"
	"log"
	"math/rand"
	"time"
	agriculture_service "Project/vanaja/proto"
	// agriculture_service "github.com/hjani-2003/Cloud_Computing_Project/tree/vanaja/vanaja/proto"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := agriculture_service.NewAgricultureServiceClient(conn)

	// Open bidirectional stream
	stream, err := client.AnalyzePatterns(context.Background())
	if err != nil {
		log.Fatalf("Error creating stream: %v", err)
	}

	// Goroutine to send data to server
	go func() {
		for {
			// Generate random agriculture data
			data := &agriculture_service.AgricultureData{
				Id:           int32(rand.Intn(1000)),
				CropType:     "Wheat",
				Temperature:  25.0 + rand.Float32()*5,
				Humidity:     60.0 + rand.Float32()*10,
				SoilMoisture: 30.0 + rand.Float32()*5,
				YieldAmount:  100.0 + rand.Float32()*20,
			}

			// Send data to server
			if err := stream.Send(data); err != nil {
				log.Fatalf("Failed to send data: %v", err)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	// Receive analysis results from server
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error receiving data: %v", err)
		}
		log.Printf("Received pattern analysis for crop ID %d: %s (Predicted Yield: %.2f)", res.Id, res.Message, res.PredictedYield)
	}
}
