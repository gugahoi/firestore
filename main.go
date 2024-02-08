package main

import (
	"context"
	"log"
	"os"
)

func main() {
	cmd := os.Args[1]

	var err error

	client := createClient(context.Background())

	switch cmd {
	case "document":
		err = NewDocument(client).Run(os.Args[2:])
	// case "collection":
	// 	err = NewCollection(client).Run(os.Args[2:])
	default:
		log.Fatalln("unknown command, supported commands are: document, collection")
	}

	if err != nil {
		log.Fatalf("operation failed: %v", err)
	}
	os.Exit(0)
}
