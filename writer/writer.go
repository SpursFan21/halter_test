// writer.go

package writer

import (
	"fmt"
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
		// Assuming error means no row found, insert the new message
		_, err := db.Exec("INSERT INTO locations (serial_number, timestamp, latitude, longitude) VALUES ($1, $2, $3, $4)",
			message.SerialNumber, message.Timestamp, message.Latitude, message.Longitude)
		if err != nil {
			return fmt.Errorf("failed to insert message into database: %w", err)
		}
	} else {
		// Update existing message if timestamp is newer
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
