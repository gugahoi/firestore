package main

import (
	"context"
	"log"
	"os"
)

func main() {
	// remove timestamp from log lines
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	if len(os.Args) == 1 {
		usage()
	}
	cmd := os.Args[1]

	var err error

	client := createClient(context.Background())
	defer client.Close()

	switch cmd {
	case "document":
		err = NewDocument(client).Run(os.Args[2:])
	case "collection":
		err = NewCollection(client).Run(os.Args[2:])
	default:
		usage()
	}

	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("operation failed: %v", err)
	}
	os.Exit(0)
}

// usage prints the usage message for this CLI.
func usage() {
	log.SetOutput(os.Stderr)
	log.Fatalln(`
Usage:
	firestore [command] [subcommand] ...args

Example:
	firestore document get /path/to/document/here
	`)
}
