package main

import (
	"os"

	"github.com/kurtmc/cloud-engineering-intern-test/analyser"
	"github.com/kurtmc/cloud-engineering-intern-test/producer"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		println("Usage: go run main.go [producer|analyser]")
		os.Exit(1)
	}

	app := args[0]

	switch app {
	case "producer":
		producer.Producer()
	case "analyser":
		analyser.Analyser()
	default:
		println("Invalid application name. Use 'producer' or 'analyser'.")
		os.Exit(1)
	}
}
