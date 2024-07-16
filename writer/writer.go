// writer.go

package writer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	SerialNumber string    `json:"serial_number"`
	Timestamp    time.Time `json:"timestamp"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
}

func ConnectPostgres() (*sqlx.DB, error) {
	maxAttempts := 5

	dbHost := os.Getenv("DATABASE_HOST")
	dbPort := os.Getenv("DATABASE_PORT")
	dbUser := os.Getenv("DATABASE_USER")
	dbPassword := os.Getenv("DATABASE_PASSWORD")
	dbName := os.Getenv("DATABASE_DBNAME")
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var db *sqlx.DB
	var err error

	for i := 0; i < maxAttempts; i++ {
		if i != 0 {
			time.Sleep(5 * time.Second)
		}
		db, err = sqlx.Connect("postgres", psqlInfo)
		if err != nil {
			continue
		}
		err = db.Ping()
		if err != nil {
			continue
		}
		return db, nil
	}

	return nil, fmt.Errorf("failed to connect to PostgreSQL after %d attempts", maxAttempts)
}

func ConnectRabbitMQ() (*amqp.Connection, *amqp.Channel, <-chan amqp.Delivery, error) {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	q, err := ch.QueueDeclare(
		"collar_messages", // name
		false,             // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to register a consumer: %w", err)
	}

	return conn, ch, msgs, nil
}

func UpdateDatabase(db *sqlx.DB, message Message) error {
	var existingMessage Message
	err := db.Get(&existingMessage, "SELECT * FROM locations WHERE serial_number=$1", message.SerialNumber)
	if err != nil {
		_, err := db.Exec("INSERT INTO locations (serial_number, timestamp, latitude, longitude) VALUES ($1, $2, $3, $4)",
			message.SerialNumber, message.Timestamp, message.Latitude, message.Longitude)
		if err != nil {
			return fmt.Errorf("failed to insert message into database: %w", err)
		}
	} else {
		if message.Timestamp.After(existingMessage.Timestamp) {
			_, err := db.Exec("UPDATE locations SET timestamp=$1, latitude=$2, longitude=$3 WHERE serial_number=$4",
				message.Timestamp, message.Latitude, message.Longitude, message.SerialNumber)
			if err != nil {
				return fmt.Errorf("failed to update message in database: %w", err)
			}
		}
	}

	return nil
}

func Writer() {
	conn, ch, msgs, err := ConnectRabbitMQ()
	if err != nil {
		log.Fatalf("Error connecting to RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	db, err := ConnectPostgres()
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}
	defer db.Close()

	for msg := range msgs {
		var message Message
		err := json.Unmarshal(msg.Body, &message)
		if err != nil {
			log.Printf("Failed to decode message: %v", err)
			continue
		}

		err = UpdateDatabase(db, message)
		if err != nil {
			log.Printf("Failed to update database: %v", err)
			continue
		}

		log.Printf("Updated database with message: %+v", message)
	}
}
