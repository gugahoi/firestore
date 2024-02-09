package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
)

// createClient creates a firestore client.
// Close client when done with `defer client.Close()`.
func createClient(ctx context.Context) *firestore.Client {
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		log.Fatal("Missing PROJECT_ID env variable")
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return client
}
