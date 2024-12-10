package main

import (
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

func main() {

	var ipAddress string
	var requestReceivedTime string
	var predictedValueModel1, predictedValueModel2, predictedValueModel3 string
	var confidenceModel1, confidenceModel2, confidenceModel3 float64
	var status int
	var responseSentTime string
	var imageLocation string

	// Variables to hold parsed values
	ipAddress = "192.168.0.101"
	requestReceivedTime = "2024-12-10T18:30:45Z"
	predictedValueModel1 = "cat"
	predictedValueModel2 = "dog"
	predictedValueModel3 = "bird"
	confidenceModel1 = 0.95
	confidenceModel2 = 0.87
	confidenceModel3 = 0.78
	status = 1
	responseSentTime = "2024-12-10T18:30:50Z"
	imageLocation = "./Images/cat_image_12345.jpg"

	// Create the concatenated string
	message := fmt.Sprintf("%s %s %s %.2f %s %.2f %s %.2f %d %s %s",
		ipAddress,
		requestReceivedTime,
		predictedValueModel1,
		confidenceModel1,
		predictedValueModel2,
		confidenceModel2,
		predictedValueModel3,
		confidenceModel3,
		status,
		responseSentTime,
		imageLocation,
	)

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
	messageBody := fmt.Sprintf("%s %d", "./Images/temp_image_1733764561.jpg", imageID)
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
			var confidence float64

			// Parse the message using fmt.Sscanf
			_, err := fmt.Sscanf(string(d.Body), "%d %s %f", &id, &label, &confidence)
			if err != nil {
				log.Fatalf("Failed to parse message: %v", err)
			}

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
			var confidence float64

			// Parse the message using fmt.Sscanf
			_, err := fmt.Sscanf(string(d.Body), "%d %s %f", &id, &label, &confidence)
			if err != nil {
				log.Fatalf("Failed to parse message: %v", err)
			}

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
			var confidence float64

			// Parse the message using fmt.Sscanf
			_, err := fmt.Sscanf(string(d.Body), "%d %s %f", &id, &label, &confidence)
			if err != nil {
				log.Fatalf("Failed to parse message: %v", err)
			}

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
}
