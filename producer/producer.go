package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	SerialNumber string    `json:"serial_number"`
	Timestamp    time.Time `json:"timestamp"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
}

func connectRabbitMQ() (conn *amqp.Connection, ch *amqp.Channel, q amqp.Queue, err error) {
	maxAttempts := 5
	for i := 0; i < maxAttempts; i++ {
		if i != 0 {
			time.Sleep(5 * time.Second)
		}
		conn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err != nil {
			continue
		}
		ch, err = conn.Channel()
		if err != nil {
			continue
		}
		q, err = ch.QueueDeclare(
			"collar_messages", // name
			false,             // durable
			false,             // delete when unused
			false,             // exclusive
			false,             // no-wait
			nil,               // arguments
		)
		if err != nil {
			continue
		}
		return
	}
	return
}

func Producer() {
	serialNumbers := []string{}
	start := rand.Intn(1000)
	for i := 0; i < 10; i++ {
		serialNumbers = append(serialNumbers, fmt.Sprintf("%010d", start+i))
	}

	conn, ch, q, err := connectRabbitMQ()
	if err != nil {
		panic(fmt.Sprintf("failed to connect to rabbitmq: %v", err))
	}
	defer conn.Close()
	defer ch.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	e := echo.New()
	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.PUT("/publish_messages", func(c echo.Context) error {
		messages := generateMessages(serialNumbers)
		for _, msg := range messages {
			body, err := json.Marshal(msg)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal json: %v", err))
			}
			err = ch.PublishWithContext(ctx,
				"",     // exchange
				q.Name, // routing key
				false,  // mandatory
				false,  // immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        []byte(body),
				})
			if err != nil {
				panic("Failed to publish message")
			}
		}
		return c.JSON(http.StatusOK, messages)
	})
	e.Logger.Fatal(e.Start(":8080"))
}

func getRandomCoord() (float64, float64) {
	minLat := -36.8095661
	maxLat := -36.9194231
	minLong := 174.7747336
	maxLong := 174.8724411
	lat := minLat + rand.Float64()*(maxLat-minLat)
	long := minLong + rand.Float64()*(maxLong-minLong)
	return lat, long
}

func generateMessages(serialNumbers []string) []Message {
	result := []Message{}
	for _, serialNumber := range serialNumbers {
		lat, long := getRandomCoord()
		result = append(result, Message{
			SerialNumber: serialNumber,
			Timestamp:    time.Now(),
			Latitude:     lat,
			Longitude:    long,
		})
	}
	return result
}
