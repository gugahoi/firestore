package main

import (
	"context"
	"encoding/json"
	"fmt"
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
Example: 
	firestore document get <path>
	firestore document cp <source> <destination>
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
		err = fmt.Errorf("action not found, available: mv, cp")
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

	// pretty print json data
	contents, _ := json.MarshalIndent(snap.Data(), "", "    ")
	log.Printf(string(contents))

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
