package main

import (
	agriculture_service "Project/testing-rabbit/protobuf/proto" // Correct import path
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"context"
	"time"

	"github.com/streadway/amqp"

	_ "github.com/lib/pq"

	"google.golang.org/grpc/peer"

	"google.golang.org/grpc" // Import grpc package
)

// CustomListener is a wrapper around net.Listener to log client connections
type CustomListener struct {
	net.Listener
}

type mockImageStream struct {
    grpc.ServerStream
    reqChan  chan *agriculture_service.ImageData
    respChan chan *agriculture_service.ClassificationResults
}


func (m *mockImageStream) Context() context.Context {
    // Create a mock peer with an address
    ctx := context.Background()
    p := &peer.Peer{
        Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 50051}, // mock IP and port
    }
    return peer.NewContext(ctx, p)
}


func (m *mockImageStream) Send(res *agriculture_service.ClassificationResults) error {
    m.respChan <- res
    return nil
}

func (m *mockImageStream) Recv() (*agriculture_service.ImageData, error) {
    req, ok := <-m.reqChan
    if !ok {
        return nil, io.EOF
    }
    return req, nil
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

// Implement the bidirectional streaming RPC method
func (s *server) ClassifyImage(stream agriculture_service.ImageClassificationService_ClassifyImageServer) error {

	var ipAddress string
	var requestReceivedTime string
	var predictedValueModel1, predictedValueModel2, predictedValueModel3 string
	var time_taken_model_1, time_taken_model_2, time_taken_model_3 float32
	var confidenceModel1, confidenceModel2, confidenceModel3 float32
	var status int
	var responseSentTime string
	var imageLocation string

	status = 0
	// Retrieve the client's IP address from the context using the peer package
	p, ok := peer.FromContext(stream.Context())
	if !ok {
		return fmt.Errorf("failed to get peer from context")
	}

	ipAddress = p.Addr.String() // clientIP will have the format "IP:PORT"
	log.Printf("Received request from client with IP: %s", ipAddress)

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
		updatedImagePath := fmt.Sprintf("./Images/temp_image_%d.jpg", imageData.Id)
		imageLocation = updatedImagePath
		err = os.WriteFile(updatedImagePath, imageData.Image, 0644)
		if err != nil {
			return fmt.Errorf("failed to save image: %v", err)
		}
		log.Printf("Image saved at %s", updatedImagePath)

		// Generate unique request ID and record request received time
		requestReceivedTime = time.Now().Format(time.RFC3339)

		// ##########################################################################

		modelResults := make([]*agriculture_service.ModelResult, 0)

		fmt.Println("Go rabbitmq")
		conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		defer conn.Close()
		fmt.Println("Successfully connected to rabbitMQ instance")
		ch, err := conn.Channel()
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		defer ch.Close()

		q0, err := ch.QueueDeclare(
			"DbLogQueue",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		fmt.Println(q0)

		q, err := ch.QueueDeclare(
			"AlexnetQueue",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		fmt.Println(q)

		q1, err := ch.QueueDeclare(
			"Convnext_tinyQueue",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		fmt.Println(q1)

		q2, err := ch.QueueDeclare(
			"MobilevnetQueue",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		fmt.Println(q2)

		imageID := time.Now().Unix()
		// Create the concatenated string message
		messageBody := fmt.Sprintf("%s%d%s %d", "./Images/temp_image_", imageData.Id, ".jpg", imageID)
		fmt.Println("this")
		fmt.Println(messageBody)
		fmt.Println("this")
		err = ch.Publish(
			"",
			"AlexnetQueue",
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(messageBody),
			},
		)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		err = ch.Publish(
			"",
			"MobilevnetQueue",
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(messageBody),
			},
		)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		err = ch.Publish(
			"",
			"Convnext_tinyQueue",
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(messageBody),
			},
		)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		fmt.Println("successfully published message to queues")

		msgs, err := ch.Consume(
			"AlexnetResultQueue",
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		forever := make(chan bool)
		go func() {
			for d := range msgs {

				fmt.Printf("Received message: %s\n", d.Body)

				// Variables to hold parsed values
				var id int
				var label string
				var confidence float32
				var executedTime float32

				// Parse the message using fmt.Sscanf
				_, err := fmt.Sscanf(string(d.Body), "%d %s %f %f", &id, &label, &confidence, &executedTime)
				if err != nil {
					log.Fatalf("Failed to parse message: %v", err)
				}
				// Store the result
				modelResults = append(modelResults, &agriculture_service.ModelResult{
					ModelName:       "alexnet",
					PredictedLabel:  label,
					ConfidenceScore: confidence,
				})
				predictedValueModel1 = label
				confidenceModel1 = confidence
				time_taken_model_1 = executedTime
				if id == int(imageID) {
					fmt.Println("Exit message received. Closing application...")
					forever <- true // Send a signal to unblock the main thread
					break           // Exit the loop
				}
			}
		}()

		fmt.Println("Successfully connected to our rabbitMQ instance!")
		fmt.Println(" [*] - waiting for messages")
		<-forever

		// ---------------------------------------

		msgs2, err := ch.Consume(
			"MobilevnetResultQueue",
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		forever2 := make(chan bool)
		go func() {
			for d := range msgs2 {

				fmt.Printf("Received message: %s\n", d.Body)

				// Variables to hold parsed values
				var id int
				var label string
				var confidence float32
				var executedTime float32

				// Parse the message using fmt.Sscanf
				_, err := fmt.Sscanf(string(d.Body), "%d %s %f %f", &id, &label, &confidence, &executedTime)
				if err != nil {
					log.Fatalf("Failed to parse message: %v", err)
				}

				// Store the result
				modelResults = append(modelResults, &agriculture_service.ModelResult{
					ModelName:       "mobilevnet",
					PredictedLabel:  label,
					ConfidenceScore: confidence,
				})
				predictedValueModel3 = label
				confidenceModel3 = confidence
				time_taken_model_2 = executedTime

				if id == int(imageID) {
					fmt.Println("Exit message received. Closing application...")
					forever2 <- true // Send a signal to unblock the main thread
					break            // Exit the loop
				}
			}
		}()

		fmt.Println("Successfully connected to our rabbitMQ instance!")
		fmt.Println(" [*] - waiting for messages")
		<-forever2

		// ----------------------------------------------------------------------

		msgs3, err := ch.Consume(
			"Convnext_tinyResultQueue",
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		forever3 := make(chan bool)
		go func() {
			for d := range msgs3 {

				fmt.Printf("Received message: %s\n", d.Body)

				// Variables to hold parsed values
				var id int
				var label string
				var confidence float32
				var executedTime float32

				// Parse the message using fmt.Sscanf
				_, err := fmt.Sscanf(string(d.Body), "%d %s %f %f", &id, &label, &confidence, &executedTime)
				if err != nil {
					log.Fatalf("Failed to parse message: %v", err)
				}

				// Store the result
				modelResults = append(modelResults, &agriculture_service.ModelResult{
					ModelName:       "convnext_tiny",
					PredictedLabel:  label,
					ConfidenceScore: confidence,
				})

				predictedValueModel2 = label
				confidenceModel2 = confidence
				time_taken_model_3 = executedTime

				if id == int(imageID) {
					fmt.Println("Exit message received. Closing application...")
					forever3 <- true // Send a signal to unblock the main thread
					break            // Exit the loop
				}
			}
		}()

		fmt.Println("Successfully connected to our rabbitMQ instance!")
		fmt.Println(" [*] - waiting for messages")
		<-forever3

		//###############################################################################################################################

		// Prepare the response to send back to the client
		response := &agriculture_service.ClassificationResults{
			Id:             imageData.Id,
			Results:        modelResults,
			OverallMessage: "Image classification completed successfully.",
		}

		responseSentTime = time.Now().Format(time.RFC3339)
		status = 1
		// Create the concatenated string
		message := fmt.Sprintf("%s %s %s %.2f %.2f %s %.2f %.2f %s %.2f %.2f %d %s %s",
			ipAddress,
			requestReceivedTime,
			predictedValueModel1,
			confidenceModel1,
			time_taken_model_1,
			predictedValueModel2,
			confidenceModel2,
			time_taken_model_2,
			predictedValueModel3,
			confidenceModel3,
			time_taken_model_3,
			status,
			responseSentTime,
			imageLocation,
		)

		err = ch.Publish(
			"",
			"DbLogQueue",
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(message),
			},
		)
		if err != nil {
			fmt.Println(err)
			panic(err)
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

	// Create mock stream for the bidirectional RPC
	reqChan := make(chan *agriculture_service.ImageData, 1)
	respChan := make(chan *agriculture_service.ClassificationResults, 1)
	errChan := make(chan error, 1)

	stream := &mockImageStream{
		reqChan:  reqChan,
		respChan: respChan,
	}

	// Send the image data to the stream
	reqChan <- imageData
	close(reqChan) // Indicate no more data will be sent

	// Call the ClassifyImage function in a goroutine
	go func() {
		// Ensure you call with a pointer receiver
		err := (&server{}).ClassifyImage(stream)
		if err != nil {
			log.Printf("Error during classification: %v", err)
			errChan <- err
		}
		close(respChan) // Close the response channel
	}()

	// Wait for the response or error
	select {
	case response := <-respChan:
		if response == nil {
			http.Error(w, "Error classifying image", http.StatusInternalServerError)
			return
		}

		// Prepare the response in the format expected by the frontend
		responseData := map[string]interface{}{
			"Results":        response.Results,
			"OverallMessage": response.OverallMessage,
		}

		// Send the JSON response
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseData)
		if err != nil {
			http.Error(w, "Failed to send response", http.StatusInternalServerError)
			log.Printf("Error encoding response: %v", err)
		}
	case err := <-errChan:
		http.Error(w, fmt.Sprintf("Error during classification: %v", err), http.StatusInternalServerError)
	}
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