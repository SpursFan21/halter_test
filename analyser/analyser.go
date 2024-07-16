package analyser

import (
	"net/http"
	"os"
	"strings"
	"time"

	"database/sql"
	"fmt"

	_ "embed"

	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

func connectPostgres() (db *sql.DB, err error) {
	maxAttempts := 5

	host := os.Getenv("DATABASE_HOST")
	port := os.Getenv("DATABASE_PORT")
	user := os.Getenv("DATABASE_USER")
	password := os.Getenv("DATABASE_PASSWORD")
	dbname := os.Getenv("DATABASE_DBNAME")
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	for i := 0; i < maxAttempts; i++ {
		if i != 0 {
			time.Sleep(5 * time.Second)
		}
		db, err = sql.Open("postgres", psqlInfo)
		if err != nil {
			continue
		}
		err = db.Ping()
		if err != nil {
			continue
		}
		return
	}
	return
}

func Analyser() {
	db, err := connectPostgres()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	e := echo.New()
	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.GET("/locations", func(c echo.Context) error {
		rows, err := db.Query("SELECT serial_number, latitude, longitude, timestamp FROM locations ORDER BY serial_number;")
		if err != nil {
			return err
		}
		defer rows.Close()
		result := ""
		for rows.Next() {
			var serialNumber string
			var latitude, longitude float64
			var timestamp time.Time
			if err := rows.Scan(&serialNumber, &latitude, &longitude, &timestamp); err != nil {
				return err
			}
			result = result + fmt.Sprintf("%s,%f,%f,%v\n", serialNumber, latitude, longitude, timestamp)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		return c.String(http.StatusOK, strings.TrimSpace(result))
	})
	e.Logger.Fatal(e.Start(":8080"))
}
