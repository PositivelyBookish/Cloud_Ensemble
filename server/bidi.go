package main

import (
	agriculture_service "Project/testing-rabbit/protobuf/proto" // Correct import path
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
	"google.golang.org/grpc" // Import grpc package
	"google.golang.org/grpc/peer"
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

func main() {

	port := flag.String("port", "50051", "The server port")
	flag.Parse()

	// Set up the server
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

	// Start the server
	log.Printf("Server listening on %v", listenAddress)
	if err := s.Serve(customLis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
