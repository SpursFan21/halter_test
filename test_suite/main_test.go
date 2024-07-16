package test_suite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"

	amqp "github.com/rabbitmq/amqp091-go"
)

type CloudEngineeringInternTestSuite struct {
	suite.Suite
	StartingNumber int
}

// updated
func checkRabbitMQ() error {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func checkPostgres() error {
	host := "location_storage"
	port := 5432
	user := "postgres"
	password := "example"
	dbname := "postgres"
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return err
	}
	return nil
}

func checkProducer() error {
	resp, err := http.Get("http://location_producer:8080/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return err
	}
	return nil
}

func checkAnalyser() error {
	resp, err := http.Get("http://location_analyser:8080/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return err
	}
	return nil
}

// added
func checkWriter() error {
	resp, err := http.Get("http://location_writer:8080/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("writer service returned non-200 status: %v", resp.StatusCode)
	}
	return nil
}

func (suite *CloudEngineeringInternTestSuite) SetupSuite() {
	// check that everything is up, fail test suite if after a few retries things are still not up
	maxAttempts := 5
	var failureMessage string

	var err error
	for i := 0; i < maxAttempts; i++ {
		if i != 0 {
			time.Sleep(10 * time.Second)
		}

		err = checkRabbitMQ()
		if err != nil {
			failureMessage = fmt.Sprintf("rabbitmq is not running correctly: %v", err)
			continue
		}

		err = checkPostgres()
		if err != nil {
			failureMessage = fmt.Sprintf("postgres is not running correctly: %v", err)
			continue
		}
		err = checkProducer()
		if err != nil {
			failureMessage = fmt.Sprintf("producer is not running correctly: %v", err)
			continue
		}
		err = checkAnalyser()
		if err != nil {
			failureMessage = fmt.Sprintf("analyser is not running correctly: %v", err)
			continue
		}
		err = checkWriter() // added
		if err != nil {
			failureMessage = fmt.Sprintf("writer is not running correctly: %v", err)
			continue
		}

		return

	}

	suite.FailNow(failureMessage)
}

func (suite *CloudEngineeringInternTestSuite) TearDownSuite() {
}

func (suite *CloudEngineeringInternTestSuite) SetupTest() {
}

func (suite *CloudEngineeringInternTestSuite) TearDownTest() {
}

type Message struct {
	SerialNumber string    `json:"serial_number"`
	Timestamp    time.Time `json:"timestamp"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
}

func (suite *CloudEngineeringInternTestSuite) TestBasicMessageFlow() {
	client := &http.Client{}
	for i := 0; i < 10; i++ {
		url := "http://location_producer:8080/publish_messages"
		req, err := http.NewRequest(http.MethodPut, url, nil)
		if err != nil {
			suite.Fail("failed to create request: %v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			suite.Fail("failed to send request: %v", err)
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			suite.Fail("failed to read response body: %v", err)
		}
		result := []Message{}
		err = json.Unmarshal(b, &result)
		if err != nil {
			suite.Fail("failed to unmarshal response body: %v", err)
		}

		// give application time to consume and write value to database
		time.Sleep(1 * time.Second)

		url = "http://location_analyser:8080/locations"
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			suite.Fail("failed to create request: %v", err)
		}
		resp, err = client.Do(req)
		if err != nil {
			suite.Fail("failed to send request: %v", err)
		}
		defer resp.Body.Close()
		b, err = io.ReadAll(resp.Body)
		if err != nil {
			suite.Fail("failed to read response body: %v", err)
		}

		lines := strings.Split(string(b), "\n")

		messagesSent := len(result)
		messagesInDatabase := len(lines)
		suite.Assert().Equal(messagesSent, messagesInDatabase, fmt.Sprintf("expected there to be %d messages in the database but got %d", messagesSent, messagesInDatabase))

	}
}

func TestCloudEngineeringInternTestSuite(t *testing.T) {
	suite.Run(t, new(CloudEngineeringInternTestSuite))
}
