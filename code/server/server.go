package main

import (
	agriculture_service "Project/code/protobuf/proto" // Correct import path
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	_ "github.com/lib/pq"

	"google.golang.org/grpc/peer"

	"google.golang.org/grpc" // Import grpc package
)

// Connect to the PostgreSQL database
func connectToDatabase() (*sql.DB, error) {
	connStr := "user=postgres dbname=cloudproject password=postgres host=localhost sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("could not connect to the database: %v", err)
	}
	return db, nil
}

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

	// Modify the imagePath to store it in the "Images" folder on your desktop
	desktopPath := "../Images" // Adjust the username if necessary
	imagePath := fmt.Sprintf("%s/temp_image_%d.jpg", desktopPath, imageData.Id)
	// imagePath := fmt.Sprintf("/tmp/temp_image_%d.jpg", imageData.Id)
	log.Printf("IMAGE LOCATION: %s", imagePath)
	log.Printf("Image Path: %s", imagePath)
	err := os.WriteFile(imagePath, imageData.Image, 0644)
	if err != nil {
		return "", 0, fmt.Errorf("failed to save image: %v", err)
	}
	log.Printf("Image saved")
	// defer os.Remove(imagePath) // Clean up the image file after classification

	// Use Python to classify the image with the model
	// cmd := exec.Command("bash", "-c", fmt.Sprintf("ls ../../../../cloud_project"))
	cmd := exec.Command("bash", "-c", fmt.Sprintf("source ../.venv/bin/activate && python3 ../pyScripts/%s.py %s", modelName, imagePath))
	output, err := cmd.CombinedOutput()
	log.Printf("Command output: %s", output)

	// log.Println("Error:    :()",err)
	if err != nil {
		return "", 0, fmt.Errorf("failed to classify image: %v", err)
	}
	log.Printf("%s.py executed", modelName)

	// Parse the output (assuming it outputs "Predicted Class: <class> Confidence: <confidence>%")
	var predictedLabel string
	var confidenceScore float32

	// Use fmt.Sscanf to capture predicted label and confidence (without the '%')
	_, err = fmt.Sscanf(string(output), "Predicted Class: %s\nConfidence: %f%%", &predictedLabel, &confidenceScore)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse classification result: %v", err)
	}

	// Convert confidence from percentage to a decimal value (if needed)
	confidenceScore = confidenceScore / 100.0

	return predictedLabel, confidenceScore, nil

	// Parse the output (assuming it outputs "label confidence")
	// var predictedLabel string
	// var confidenceScore float32
	// _, err = fmt.Sscanf(string(output), "%s %f", &predictedLabel, &confidenceScore)
	// if err != nil {
	// 	return "", 0, fmt.Errorf("failed to parse classification result: %v", err)
	// }

	// return predictedLabel, confidenceScore, nil
	// return modelName, 0.0, nil // You can modify this as needed when you parse the result

}

// Implement the bidirectional streaming RPC method
func (s *server) ClassifyImage(stream agriculture_service.ImageClassificationService_ClassifyImageServer) error {
	db, err := connectToDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
		return err
	}
	defer db.Close()

	// Retrieve the client's IP address from the context using the peer package
	p, ok := peer.FromContext(stream.Context())
	if !ok {
		return fmt.Errorf("failed to get peer from context")
	}

	clientIP := p.Addr.String() // clientIP will have the format "IP:PORT"
	log.Printf("Received request from client with IP: %s", clientIP)

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

		// Save the image to a specific location
		updatedImagePath := fmt.Sprintf("../Images/temp_image_%d.jpg", imageData.Id)
		err = os.WriteFile(updatedImagePath, imageData.Image, 0644)
		if err != nil {
			return fmt.Errorf("failed to save image: %v", err)
		}
		log.Printf("Image saved at %s", updatedImagePath)

		// Generate unique request ID and record request received time
		requestTime := time.Now()
		// Insert metadata into the database (status: 0 for processing initially)
		insertSQL := `
            INSERT INTO image_requests (ip_address, request_received_time, status)
            VALUES ($1, $2, 0)
            RETURNING id`

		var requestID int
		err = db.QueryRow(insertSQL, clientIP, requestTime).Scan(&requestID)
		if err != nil {
			return fmt.Errorf("failed to insert metadata: %v", err)
		}

		// Update the image location in the database
		query := `
            UPDATE image_requests
            SET image_location = $1, status = 1
            WHERE id = $2;
        `
		_, err = db.Exec(query, updatedImagePath, requestID)
		if err != nil {
			return fmt.Errorf("failed to update image path in database: %v", err)
		}
		log.Printf("Database updated with new image path for ID %d", imageData.Id)

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
				ModelName:       modelName,
				PredictedLabel:  label,
				ConfidenceScore: confidence,
			})
		}

		// Prepare the response to send back to the client
		response := &agriculture_service.ClassificationResults{
			Id:             imageData.Id,
			Results:        modelResults,
			OverallMessage: "Image classification completed successfully.",
		}

		// Update the database with model results and status (status: 1 for processed)
		updateSQL := `
            UPDATE image_requests 
            SET predicted_value_model_1 = $1, confidence_model_1 = $2,
                predicted_value_model_2 = $3, confidence_model_2 = $4,
                predicted_value_model_3 = $5, confidence_model_3 = $6,
                status = 1, response_sent_time = $7
            WHERE id = $8`

		_, err = db.Exec(updateSQL, modelResults[0].PredictedLabel, modelResults[0].ConfidenceScore,
			modelResults[1].PredictedLabel, modelResults[1].ConfidenceScore,
			modelResults[2].PredictedLabel, modelResults[2].ConfidenceScore,
			time.Now(), requestID)
		if err != nil {
			return fmt.Errorf("failed to update metadata: %v", err)
		}

		// Send the results back to the client
		if err := stream.Send(response); err != nil {
			return fmt.Errorf("failed to send response to client: %v", err)
		}

		// Close the connection after processing the request
		log.Println("Processed one request, closing connection.")
		return nil
	}
}

// Handle image classification via HTTP POST request
// Handle image classification via HTTP POST request
func classifyImageHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure it's a POST request
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Check that the Content-Type is multipart/form-data
	if !isMultipartForm(r) {
		http.Error(w, "Invalid Content-Type, expected multipart/form-data", http.StatusBadRequest)
		return
	}

	// Parse the image file from the form data
	err := r.ParseMultipartForm(10 << 20) // Limit to 10 MB
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		log.Printf("Error parsing form: %v", err)
		return
	}

	// Get the file from the form
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Unable to get image from form", http.StatusBadRequest)
		log.Printf("Error retrieving image from form: %v", err)
		return
	}
	defer file.Close()

	// Read the image file into a byte slice
	imageBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Unable to read image", http.StatusInternalServerError)
		log.Printf("Error reading image: %v", err)
		return
	}

	// Generate a unique image ID (for simplicity, using a timestamp)
	imageID := time.Now().Unix()

	// Create an ImageData object
	imageData := &agriculture_service.ImageData{
		Id:    int32(imageID),
		Image: imageBytes,
	}

	// Call the classifyImageWithModel function for each model
	modelNames := []string{"alexnet", "convnext_tiny", "mobilevnet"}
	var modelResults []*agriculture_service.ModelResult

	for _, modelName := range modelNames {
		// Call the classification function
		label, confidence, err := classifyImageWithModel(imageData, modelName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error classifying image with model %s: %v", modelName, err), http.StatusInternalServerError)
			return
		}

		// Append the result for each model
		modelResults = append(modelResults, &agriculture_service.ModelResult{
			ModelName:       modelName,
			PredictedLabel:  label,
			ConfidenceScore: confidence,
		})
	}

	// Prepare the response
	response := &agriculture_service.ClassificationResults{
		Id:             imageData.Id,
		Results:        modelResults,
		OverallMessage: "Image classification completed successfully.",
	}

	// Convert the response to JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Check if the Content-Type is multipart/form-data
func isMultipartForm(r *http.Request) bool {
	return r.Header.Get("Content-Type")[:19] == "multipart/form-data"
}

func main() {
	// Setup gRPC server
	port := flag.String("port", "50051", "The server port")
	flag.Parse()

	// Set up the gRPC server
	listenAddress := fmt.Sprintf("0.0.0.0:%s", *port)
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("Failed to listen on %v: %v", listenAddress, err)
	}

	// Wrap the listener with the custom implementation
	customLis := &CustomListener{Listener: lis}

	// Create a gRPC server
	s := grpc.NewServer()
	agriculture_service.RegisterImageClassificationServiceServer(s, &server{})

	// Start the gRPC server in a goroutine
	go func() {
		log.Printf("gRPC Server listening on %v", listenAddress)
		if err := s.Serve(customLis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// HTTP handler for image classification with CORS headers
	http.HandleFunc("/classify", classifyImageHandler)

	// Start the HTTP server for image classification (listening on port 8082)
	log.Printf("HTTP Server listening on port 8082")
	err = http.ListenAndServe(":8082", nil)
	if err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
