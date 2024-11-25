package main

import (
	agriculture_service "Project/vanaja/protobuf/proto" // Correct import path
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
)

// Function to read an image file and return it as a byte slice
func readImageFile(imagePath string) ([]byte, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	// Read the entire file into a byte slice
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	fileSize := fileInfo.Size()
	buffer := make([]byte, fileSize)

	_, err = file.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	return buffer, nil
}

func main() {
	// Set up the connection to the server
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close() // Ensure the connection is closed when done
	log.Println("Successfully connected to the server at 0.0.0.0:50051")

	client := agriculture_service.NewImageClassificationServiceClient(conn)

	// Open a bidirectional stream
	stream, err := client.ClassifyImage(context.Background())
	if err != nil {
		log.Fatalf("Error creating stream: %v", err)
	}

	// Read and send the image
	imagePath := "/home/cloud-ensemble1/Desktop/bidi/Cloud_Computing_Project/vanaja/test_images/Tomato___Tomato_mosaic_virus.jpg"
	imageData, err := readImageFile(imagePath)
	if err != nil {
		log.Fatalf("Failed to read image: %v", err)
	}

	// Generate a unique ID (e.g., using timestamp or a counter)
	imageID := time.Now().Unix()

	// Create the ImageData object with the unique ID
	data := &agriculture_service.ImageData{
		Id:    int32(imageID),
		Image: imageData,
	}

	// Send image data to the server
	if err := stream.Send(data); err != nil {
		log.Fatalf("Failed to send image data: %v", err)
	}
	log.Printf("Sent image data with ID %d", data.Id)

	// Wait for the result from the server
	res, err := stream.Recv()
	if err == io.EOF {
		// Server finished sending results
		log.Println("Server closed the connection.")
	} else if err != nil {
		log.Fatalf("Failed to receive response: %v", err)
	} else {
		// Process and log the classification results
		log.Printf("Received results for image ID %d:", res.Id)
		for _, result := range res.Results {
			log.Printf("Model: %s, Predicted Label: %s, Confidence: %.2f",
				result.ModelName, result.PredictedLabel, result.ConfidenceScore)
		}
		log.Printf("Overall message: %s", res.OverallMessage)
	}

	// Close the stream after processing the request
	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Failed to close stream: %v", err)
	}

	// Optionally, exit the process after closing the connection
	log.Println("Client finished processing the image. Exiting.")
}
