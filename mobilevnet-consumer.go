package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/streadway/amqp"
)

func classifyImageWithModel2(imageLocationId string, ch *amqp.Channel) error {
	// Variables to hold parsed values
	var id int
	var imageLocation string

	// Parse the message using fmt.Sscanf
	_, err := fmt.Sscanf(imageLocationId, "%s %d", &imageLocation, &id)
	if err != nil {
		log.Fatalf("Failed to parse message: %v", err)
	}
	cmd := exec.Command("bash", "-c", fmt.Sprintf("source ./.venv/bin/activate && python3 ./pyScripts/%s.py %s", "mobilevnet", imageLocation))
	output, err := cmd.CombinedOutput()
	log.Printf("Command output: %s", output)

	if err != nil {
		return fmt.Errorf("failed to classify image: %v", err)
	}
	log.Printf("%s.py executed", "mobilevnet")

	var predictedLabel string
	var confidenceScore float32

	_, err = fmt.Sscanf(string(output), "Predicted Class: %s\nConfidence: %f%%", &predictedLabel, &confidenceScore)
	if err != nil {
		return fmt.Errorf("failed to parse classification result: %v", err)
	}

	confidenceScore = confidenceScore / 100.0

	// Create the concatenated string message
	messageBody := fmt.Sprintf("%d %s %.2f", id, predictedLabel, confidenceScore)

	q, err := ch.QueueDeclare(
		"MobilevnetResultQueue",
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

	err = ch.Publish(
		"",
		"MobilevnetResultQueue",
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

	fmt.Println("successfully published message to MobilevnetResultQueue")

	return nil
}

func main() {
	fmt.Println("Go consumer rabbitmq ")
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

	msgs, err := ch.Consume(
		"MobilevnetQueue",
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
			classifyImageWithModel2(string(d.Body), ch)
		}
	}()

	fmt.Println("Successfully connected to our rabbitMQ instance!")
	fmt.Println(" [*] - waiting for messages")
	<-forever
}
