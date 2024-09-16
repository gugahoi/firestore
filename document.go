package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Document struct{ client *firestore.Client }

func NewDocument(client *firestore.Client) Document {
	return Document{client: client}
}

func (d Document) usage() {
	log.SetOutput(os.Stderr)
	log.Fatalln(`
Description:
	Perform actions on firestore documents.
Usage: 
	firestore document [action] <...args>
Actions:
	add - adds a new document with contents from STDIN
	get - retrieves a document and prints its contents to the console
	mv  - moves a document from the source to the destination, deleting the source document
	cp  - copies a document from the source to the destination
	rm  - deletes a document
Examples: 
	firestore document add /path/to/document/here < file.json
	firestore document get /path/to/document/here
	firestore document cp /path/to/source/document /path/to/destination/document
	firestore document mv /path/to/source/document /path/to/destination/document
	firestore document rm /path/to/document/here
	`)
}

func (d Document) checkArgs(args []string, size int) {
	if len(args) < size {
		d.usage()
	}
}

func (d Document) Run(args []string) error {
	d.checkArgs(args, 1)
	var err error
	action := args[0]

	switch action {
	case "add":
		d.checkArgs(args, 2)
		path := args[1]
		err = d.add(path)
	case "get":
		d.checkArgs(args, 2)
		src := args[1]
		err = d.get(src)
	case "mv":
		d.checkArgs(args, 3)
		src := args[1]
		dst := args[2]
		err = d.move(src, dst)
	case "cp":
		d.checkArgs(args, 3)
		src := args[1]
		dst := args[2]
		err = d.copy(src, dst)
	case "rm":
		d.checkArgs(args, 2)
		src := args[1]
		err = d.delete(src)
	default:
		d.usage()
	}

	return err
}

// get retrieves the contents of the document and prints it to the console.
func (d Document) get(src string) error {
	ctx := context.Background()

	srcRef := d.client.Doc(strings.TrimPrefix(src, "/"))

	snap, err := srcRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("document not found")
		}
		return fmt.Errorf("failed to read document: %v", err)
	}

	// encoder with html escaping disabled
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	// pretty print json data
	if err := encoder.Encode(snap.Data()); err != nil {
		return fmt.Errorf("failed to parse document: %v", err)
	}

	return nil
}

// move moves a document from the source to the destination, deleting the
// source. Can be useful when wanting to rename.
func (d Document) move(src, dst string) error {
	ctx := context.Background()

	srcRef := d.client.Doc(strings.TrimPrefix(src, "/"))

	snap, err := srcRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to read source document: %v", err)
	}

	dstRef := d.client.Doc(strings.TrimPrefix(dst, "/"))
	_, err = dstRef.Set(ctx, snap.Data())
	if err != nil {
		return fmt.Errorf("failed to write destination document: %v", err)
	}

	_, err = srcRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete source document: %v", err)
	}
	return nil
}

// copy copies a document from the source to the destination.
func (d Document) copy(src, dst string) error {
	ctx := context.Background()

	srcRef := d.client.Doc(strings.TrimPrefix(src, "/"))
	snap, err := srcRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to read source document: %v", err)
	}

	dstRef := d.client.Doc(strings.TrimPrefix(dst, "/"))
	_, err = dstRef.Set(ctx, snap.Data())
	if err != nil {
		return fmt.Errorf("failed to write destination document: %v", err)
	}
	return nil
}

// delete deletes a document.
func (d Document) delete(src string) error {
	ctx := context.Background()

	srcRef := d.client.Doc(strings.TrimPrefix(src, "/"))
	_, err := srcRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete document: %v", err)
	}

	return nil
}

// Add adds a document with contents from STDIN
func (d Document) add(path string) error {
	ctx := context.Background()

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}

	var data map[string]any
	err = json.Unmarshal(input, &data)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	srcRef := d.client.Doc(strings.TrimPrefix(path, "/"))
	_, err = srcRef.Create(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to create document: %v", err)
	}

	return nil
}
