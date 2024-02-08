package main

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
)

// createClient creates a firestore client.
func createClient(ctx context.Context) *firestore.Client {
	projectID := "demo-flux"

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	// Close client when done with
	// defer client.Close()
	return client
}
