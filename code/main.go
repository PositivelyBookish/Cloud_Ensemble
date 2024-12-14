package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	agriculture_service "Project/code/protobuf/proto"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/storage"
	"google.golang.org/grpc"
)

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

func createStyledLabel(text string, size float32, weight fyne.TextStyle) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = weight
	label.Alignment = fyne.TextAlignCenter
	return label
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("CLOUD-ENSEMBLE")

	// Create styled headers and labels
	headerLabel := createStyledLabel("CROP DISEASE PREDICTION SYSTEM", 104, fyne.TextStyle{Bold: true})
	subHeaderLabel := createStyledLabel("Upload a leaf image for disease detection", 18, fyne.TextStyle{Italic: true})

	// Create a card for results
	resultLabel := widget.NewTextGrid()
	resultLabel.SetText("Results will be displayed here...")

	// Style the result container
	resultCard := widget.NewCard(
		"",
		"Disease detection details",

		resultLabel,
	)

	displayedImage := canvas.NewImageFromResource(nil)
	displayedImage.SetMinSize(fyne.NewSize(400, 400))
	displayedImage.FillMode = canvas.ImageFillContain
	displayedImage.Hide()

	imageBorder := container.NewBorder(nil, nil, nil, nil, displayedImage)
	
	selectImageButton := widget.NewButtonWithIcon("Select Image", theme.FolderOpenIcon(), nil)
	selectImageButton.Importance = widget.HighImportance
	
	buttonContainer := container.NewHBox(
		layout.NewSpacer(),
		selectImageButton,
		layout.NewSpacer(),
	)

	selectImageButton.OnTapped = func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, myWindow)
				return
			}
			if reader == nil {
				return
			}

			imagePath := reader.URI().Path()
			log.Printf("Selected image: %s", imagePath)

			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				log.Printf("Image file does not exist: %s", imagePath)
				dialog.ShowError(fmt.Errorf("The selected image file does not exist"), myWindow)
				return
			}
			displayedImage.File = imagePath
			displayedImage.Refresh()
			displayedImage.Show()
			
			resultLabel.SetText("Processing image...\nPlease wait...")
			myWindow.Content().Refresh()

			progress := widget.NewProgressBarInfinite()
			resultCard.SetContent(container.NewVBox(
				widget.NewLabel("Analyzing image..."),
				progress,
			))

			go func() {
				conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to connect to server: %v", err), myWindow)
					resultCard.SetContent(widget.NewTextGridFromString("Error: Unable to connect to the server"))
					return
				}
				defer conn.Close()

				client := agriculture_service.NewImageClassificationServiceClient(conn)

				imageData, err := readImageFile(imagePath)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to read image: %v", err), myWindow)
					resultCard.SetContent(widget.NewTextGridFromString("Error: Unable to read the image"))
					return
				}

				stream, err := client.ClassifyImage(context.Background())
				if err != nil {
					dialog.ShowError(fmt.Errorf("Error creating stream: %v", err), myWindow)
					resultCard.SetContent(widget.NewTextGridFromString("Error: Unable to create stream"))
					return
				}

				imageID := time.Now().Unix()
				data := &agriculture_service.ImageData{
					Id:    int32(imageID),
					Image: imageData,
				}

				if err := stream.Send(data); err != nil {
					dialog.ShowError(fmt.Errorf("Failed to send image data: %v", err), myWindow)
					resultCard.SetContent(widget.NewTextGridFromString("Error: Unable to send image data"))
					return
				}

				res, err := stream.Recv()
				if err == io.EOF {
					log.Println("Server closed the connection.")
					resultCard.SetContent(widget.NewTextGridFromString("No response from server."))
				} else if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to receive response: %v", err), myWindow)
					resultCard.SetContent(widget.NewTextGridFromString("Error: Failed to receive response"))
					return
				} else {
					results := fmt.Sprintf("Analysis Complete!\n")
					for _, result := range res.Results {
						results += fmt.Sprintf("🔍 Model: %s\n📋 Prediction: %s\n💯 Confidence: %.2f%%\n\n",
							result.ModelName, result.PredictedLabel, result.ConfidenceScore)
					}
					results += fmt.Sprintf("\n📝 %s", res.OverallMessage)
					
					resultCard.SetContent(widget.NewTextGridFromString(results))
				}

				if err := stream.CloseSend(); err != nil {
					dialog.ShowError(fmt.Errorf("Failed to close stream: %v", err), myWindow)
					resultCard.SetContent(widget.NewTextGridFromString("Error: Failed to close stream"))
					return
				}
			}()
		}, myWindow)

		fileDialog.Resize(fyne.NewSize(1000, 800))
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png"}))
		fileDialog.Show()
	}

	headerContainer := container.NewVBox(
		headerLabel,
		widget.NewSeparator(),
		subHeaderLabel,
		layout.NewSpacer(),
	)

	imageAndResults := container.NewHSplit(
		container.NewPadded(imageBorder),
		container.NewPadded(resultCard),
	)
	imageAndResults.SetOffset(0.6)

	content := container.NewVBox(
		headerContainer,
		buttonContainer,
		widget.NewSeparator(),
		imageAndResults,
	)

	paddedContent := container.NewPadded(content)

	myWindow.SetContent(paddedContent)
	myWindow.Resize(fyne.NewSize(1200, 800))
	myWindow.ShowAndRun()
}