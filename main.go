// main.go

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/SpursFan21/halter_test/analyser"
	"github.com/SpursFan21/halter_test/producer"
	"github.com/SpursFan21/halter_test/writer"
	"github.com/jmoiron/sqlx"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		println("Usage: go run main.go [producer|analyser|writer]")
		os.Exit(1)
	}

	app := args[0]

	switch app {
	case "producer":
		producer.Producer()
	case "analyser":
		analyser.Analyser()
	case "writer":
		conn, ch, msgs, err := writer.ConnectRabbitMQ()
		if err != nil {
			log.Fatalf("Error connecting to RabbitMQ: %v", err)
		}
		defer conn.Close()
		defer ch.Close()

		// Get PostgreSQL connection details from environment variables
		dbHost := os.Getenv("DATABASE_HOST")
		dbPort := os.Getenv("DATABASE_PORT")
		dbUser := os.Getenv("DATABASE_USER")
		dbPassword := os.Getenv("DATABASE_PASSWORD")
		dbName := os.Getenv("DATABASE_DBNAME")

		// Connect to PostgreSQL
		db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName))
		if err != nil {
			log.Fatalf("Error connecting to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Consume messages from RabbitMQ
		for msg := range msgs {
			var message writer.Message
			err := json.Unmarshal(msg.Body, &message)
			if err != nil {
				log.Printf("Failed to decode message: %v", err)
				continue
			}

			// Handle the message: update or insert into the database
			err = writer.UpdateDatabase(db, message)
			if err != nil {
				log.Printf("Failed to update database: %v", err)
				continue
			}

			log.Printf("Updated database with message: %+v", message)
		}

	default:
		println("Invalid application name. Use 'producer', 'analyser', or 'writer'.")
		os.Exit(1)
	}
}
