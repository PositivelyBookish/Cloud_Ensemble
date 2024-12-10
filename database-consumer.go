package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

func connectToDatabase() (*sql.DB, error) {
	connStr := "user=postgres dbname=cloudproject password=postgres host=localhost sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("could not connect to the database: %v", err)
	}
	return db, nil
}

func main() {
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
		"DbLogQueue",
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
			var ipAddress string
			var requestReceivedTime string
			var predictedValueModel1, predictedValueModel2, predictedValueModel3 string
			var confidenceModel1, confidenceModel2, confidenceModel3 float64
			var status string
			var responseSentTime string
			var imageLocation string

			// Parse the message using fmt.Sscanf
			_, err := fmt.Sscanf(
				string(d.Body),
				"%s %s %s %f %s %f %s %f %s %s %s",
				&ipAddress,
				&requestReceivedTime,
				&predictedValueModel1,
				&confidenceModel1,
				&predictedValueModel2,
				&confidenceModel2,
				&predictedValueModel3,
				&confidenceModel3,
				&status,
				&responseSentTime,
				&imageLocation,
			)
			if err != nil {
				log.Fatalf("Failed to parse message: %v", err)
			}

			db, err := connectToDatabase()
			if err != nil {
				log.Fatalf("Failed to connect to the database: %v", err)
				panic(err)
			}
			defer db.Close()
			insertSQL := `
            INSERT INTO image_requests (ip_address, request_received_time, predicted_value_model_1, confidence_model_1, predicted_value_model_2, confidence_model_2, predicted_value_model_3, confidence_model_3, status, response_sent_time, image_location)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            RETURNING id`
			var requestID int
			err = db.QueryRow(insertSQL, ipAddress,
				requestReceivedTime,
				predictedValueModel1,
				confidenceModel1,
				predictedValueModel2,
				confidenceModel2,
				predictedValueModel3,
				confidenceModel3,
				status,
				responseSentTime,
				imageLocation).Scan(&requestID)
			if err != nil {
				fmt.Print("failed to insert metadata: %v", err)
			}
			fmt.Print("the id is: %d", requestID)

		}
	}()
	fmt.Println("Successfully connected to our rabbitMQ instance!")
	fmt.Println(" [*] - waiting for messages")
	<-forever
}
