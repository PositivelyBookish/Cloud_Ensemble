package main

import (
	agriculture_service "Project/vanaja/protobuf/proto" // Correct import path
	"fmt"
	// "io"
	"log"
	"os"
	"os/exec"
	"net"

	"google.golang.org/grpc" // Import grpc package
)

// Implement the ImageClassificationServiceServer
type server struct {
	agriculture_service.UnimplementedImageClassificationServiceServer
}

func classifyImageWithAllScripts(imageData *agriculture_service.ImageData) ([]*agriculture_service.ModelResult, error) {
	// Save the image to a temporary file
	imagePath := fmt.Sprintf("/tmp/temp_image_%d.jpg", imageData.Id)
	log.Printf("%s",imagePath)
	err := os.WriteFile(imagePath, imageData.Image, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to save image: %v", err)
	}
	// defer os.Remove(imagePath) // Ensure cleanup

	// Define the Python scripts to execute
	scripts := []string{
		fmt.Sprintf("../pyScripts/alexnet.py"),
		fmt.Sprintf("../pyScripts/mobilevnet.py"),
		fmt.Sprintf("../pyScripts/convnext_tiny.py"),
	}

	results := make([]*agriculture_service.ModelResult, 0)

	// Execute each script
	for _, script := range scripts {
		log.Printf("Executing script: %s for image ID: %d", script, imageData.Id)

		// Execute the script
		cmd := exec.Command("python3", script, imagePath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Script %s execution failed: %v\nOutput: %s", script, err, string(output))
			return nil, fmt.Errorf("script %s failed: %v", script, err)
		}

		// Parse the output (assuming it returns "label confidence")
		var predictedLabel string
		var confidenceScore float32
		_, err = fmt.Sscanf(string(output), "%s %f", &predictedLabel, &confidenceScore)
		if err != nil {
			return nil, fmt.Errorf("failed to parse output from script %s: %v", script, err)
		}

		// Append the result
		results = append(results, &agriculture_service.ModelResult{
			ModelName:       script,
			PredictedLabel:  predictedLabel,
			ConfidenceScore: confidenceScore,
		})
	}

	return results, nil
}


func main() {
	// Set up the server
	listenAddress := "0.0.0.0:50051" // Change this to your server's address
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("Failed to listen on %v: %v", listenAddress, err)
	}

	// Create a gRPC server
	s := grpc.NewServer()
	agriculture_service.RegisterImageClassificationServiceServer(s, &server{})

	// Start the server
	log.Printf("Server listening on %v", listenAddress)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
