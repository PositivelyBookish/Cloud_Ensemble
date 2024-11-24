package main

import (
	agriculture_service "Project/vanaja/protobuf/proto" // Correct import path
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"

	"google.golang.org/grpc" // Import grpc package
)

// CustomListener is a wrapper around net.Listener to log client connections
type CustomListener struct {
	net.Listener
}

func (cl *CustomListener) Accept() (net.Conn, error) {
	conn, err := cl.Listener.Accept()
	if err != nil {
		return nil, err
	}
	// Log client connection
	log.Printf("Client connected: %v", conn.RemoteAddr())
	return conn, nil
}

// Implement the ImageClassificationServiceServer
type server struct {
	agriculture_service.UnimplementedImageClassificationServiceServer
}

// Function to classify the image using a Python script for each model
func classifyImageWithModel(imageData *agriculture_service.ImageData, modelName string) (string, float32, error) {
	// Save the image to a temporary file
	log.Printf("Received image data of size: %d bytes for image ID: %d", len(imageData.Image), imageData.Id)

	imagePath := fmt.Sprintf("/tmp/temp_image_%d.jpg", imageData.Id)
	log.Printf("Image Path: %s", imagePath)
	err := os.WriteFile(imagePath, imageData.Image, 0644)
	if err != nil {
		return "", 0, fmt.Errorf("failed to save image: %v", err)
	}
	log.Printf("Image saved")
	// defer os.Remove(imagePath) // Clean up the image file after classification

	// Use Python to classify the image with the model
	cmd := exec.Command("bash","-c", fmt.Sprintf("source ../.venv/bin/activate && python3 ../pyScripts/%s.py", modelName), imagePath)
	output, err := cmd.CombinedOutput()
	log.Printf("Command output: %s", output)

	// log.Println("Error:    :()",err)
	if err != nil {
		return "", 0, fmt.Errorf("failed to classify image: %v", err)
	}
	log.Printf("%s.py executed",modelName)

	// Parse the output (assuming it outputs "label confidence")
	// var predictedLabel string
	// var confidenceScore float32
	// _, err = fmt.Sscanf(string(output), "%s %f", &predictedLabel, &confidenceScore)
	// if err != nil {
	// 	return "", 0, fmt.Errorf("failed to parse classification result: %v", err)
	// }

	// return predictedLabel, confidenceScore, nil
	return "PlaceholderLabel", 0.0, nil // You can modify this as needed when you parse the result


}

// Implement the bidirectional streaming RPC method
func (s *server) ClassifyImage(stream agriculture_service.ImageClassificationService_ClassifyImageServer) error {
	// Process images sent by the client
	for {
		// Receive the image data from the client
		imageData, err := stream.Recv()
		if err == io.EOF {
			// The client closed the stream
			log.Println("Client closed the connection.")
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to receive image data: %v", err)
		}

		// Process the image with each model
		modelResults := make([]*agriculture_service.ModelResult, 0)
		modelNames := []string{"alexnet", "convnext_tiny", "mobilevnet"} // Replace with actual model names

		for _, modelName := range modelNames {
			// Classify the image with the current model
			label, confidence, err := classifyImageWithModel(imageData, modelName)
			if err != nil {
				return fmt.Errorf("failed to classify image with model %s: %v", modelName, err)
			}

			// Store the result
			modelResults = append(modelResults, &agriculture_service.ModelResult{
				ModelName:      modelName,
				PredictedLabel: label,
				ConfidenceScore: confidence,
			})
		}

		// Prepare the response to send back to the client
		response := &agriculture_service.ClassificationResults{
			Id:             imageData.Id,
			Results:        modelResults,
			OverallMessage: "Image classification completed successfully.",
		}

		// Send the results back to the client
		if err := stream.Send(response); err != nil {
			return fmt.Errorf("failed to send response to client: %v", err)
		}
	}
}

func main() {
	// Set up the server
	listenAddress := "0.0.0.0:50051" // Change this to your server's address
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("Failed to listen on %v: %v", listenAddress, err)
	}

	// Wrap the listener with the custom implementation
	customLis := &CustomListener{Listener: lis}

	// Create a gRPC server
	s := grpc.NewServer()
	agriculture_service.RegisterImageClassificationServiceServer(s, &server{})

	// Start the server
	log.Printf("Server listening on %v", listenAddress)
	if err := s.Serve(customLis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
