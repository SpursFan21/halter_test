package main

import (
	"os"

	"github.com/SpursFan21/halter_test/analyser"
	"github.com/SpursFan21/halter_test/producer"
	"github.com/SpursFan21/halter_test/writer"
)

func main() {
	args := os.Args[1:]
	app := args[0]

	switch app {
	case "producer":
		producer.Producer()
	case "analyser":
		analyser.Analyser()
	case "writer":
		writer.Writer()
	default:
		println("Invalid application name. Use 'producer', 'analyser', or 'writer'.")
		os.Exit(1)
	}
}
