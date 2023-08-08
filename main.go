package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
)

func main() {
	cmd := os.Args[1]

	var err error

	switch cmd {
	case "mv":
		src := os.Args[2]
		dst := os.Args[3]
		err = move(src, dst)
	case "cp":
		src := os.Args[2]
		dst := os.Args[3]
		err = cp(src, dst)
	default:
		log.Fatalln("unknown command, supported commands are: mv, cp")
	}

	if err != nil {
		log.Fatalf("operation failed: %v", err)
	}
	os.Exit(0)
}

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

// move moves a document from the source to the destination, deleting the
// source. Can be useful when wanting to rename.
func move(src, dst string) error {
	ctx := context.Background()
	client := createClient(ctx)
	defer client.Close()

	srcRef, err := getDocRef(client, src)
	if err != nil {
		return err
	}

	dstRef, err := getDocRef(client, dst)
	if err != nil {
		return err
	}

	snap, err := srcRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("Failed to read source document: %v", err)
	}
	_, err = dstRef.Set(ctx, snap.Data())
	if err != nil {
		return fmt.Errorf("Failed to write destination document: %v", err)
	}
	_, err = srcRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete source document: %v", err)
	}
	return nil
}

// cp copies a document from the source to the destination.
func cp(src, dst string) error {
	ctx := context.Background()
	client := createClient(ctx)
	defer client.Close()

	srcRef, err := getDocRef(client, src)
	if err != nil {
		return err
	}

	dstRef, err := getDocRef(client, dst)
	if err != nil {
		return err
	}

	snap, err := srcRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("Failed to read source document: %v", err)
	}
	_, err = dstRef.Set(ctx, snap.Data())
	if err != nil {
		return fmt.Errorf("Failed to write destination document: %v", err)
	}
	return nil
}

// getDocRef parses the path and returns the reference to the underlying document struct.
func getDocRef(client *firestore.Client, path string) (*firestore.DocumentRef, error) {
	segments := strings.Split(path, "/")

	// if the path starts with a "/" then we need to remove the first element as
	// it will be an empty string which is not a valid collection name or
	// document name in firestore
	if segments[0] == "" {
		segments = segments[1:]
	}

	// check if it's a document
	if len(segments)%2 != 0 {
		return nil, fmt.Errorf("invalid path for document: %s", path)
	}
	docRef := client.Collection(segments[0]).Doc(segments[1])
	for i := 2; i < len(segments); i = i + 2 {
		docRef = docRef.Collection(segments[i]).Doc(segments[i+1])
	}
	return docRef, nil
}
