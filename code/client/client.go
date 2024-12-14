package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	agriculture_service "Project/code/protobuf/proto" // Correct import path

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"google.golang.org/grpc"
)

// Function to read an image file and return it as a byte slice
func readImageFile(imagePath string) ([]byte, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

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
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	log.Println("Successfully connected to the server at 0.0.0.0:50051")

	client := agriculture_service.NewImageClassificationServiceClient(conn)

	// Create a Fyne app
	myApp := app.New()
	myWindow := myApp.NewWindow("Crop Image Classification")

	// Create UI elements
	label := widget.NewLabel("Select an image to classify")
	selectImageButton := widget.NewButton("Select Image", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, myWindow)
				return
			}
			if reader == nil {
				return // User cancelled the dialog
			}

			imagePath := reader.URI().Path()
			log.Printf("Selected image: %s", imagePath)

			// Read and send the image to the server
			imageData, err := readImageFile(imagePath)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to read image: %v", err), myWindow)
				return
			}

			// Open a bidirectional stream
			stream, err := client.ClassifyImage(context.Background())
			if err != nil {
				dialog.ShowError(fmt.Errorf("Error creating stream: %v", err), myWindow)
				return
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
				dialog.ShowError(fmt.Errorf("Failed to send image data: %v", err), myWindow)
				return
			}
			log.Printf("Sent image data with ID %d", data.Id)

			// Wait for the result from the server
			res, err := stream.Recv()
			if err == io.EOF {
				// Server finished sending results
				log.Println("Server closed the connection.")
			} else if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to receive response: %v", err), myWindow)
				return
			} else {
				// Process and display the classification results
				results := fmt.Sprintf("Results for image ID %d:\n", res.Id)
				for _, result := range res.Results {
					results += fmt.Sprintf("Model: %s, Predicted Label: %s, Confidence: %.2f\n",
						result.ModelName, result.PredictedLabel, result.ConfidenceScore)
				}
				results += fmt.Sprintf("Overall message: %s", res.OverallMessage)
				dialog.ShowInformation("Classification Results", results, myWindow)
			}

			// Close the stream after processing the request
			if err := stream.CloseSend(); err != nil {
				dialog.ShowError(fmt.Errorf("Failed to close stream: %v", err), myWindow)
				return
			}
		}, myWindow)
	})

	// Arrange UI elements in a vertical layout
	content := container.NewVBox(
		label,
		selectImageButton,
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(400, 300))
	myWindow.ShowAndRun()
}
