package main

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
)

type Document struct{ client *firestore.Client }

func NewDocument(client *firestore.Client) Document {
	return Document{client: client}
}

func (d Document) Run(args []string) error {
	var err error
	action := args[0]

	switch action {
	case "mv":
		src := args[1]
		dst := args[2]
		err = d.move(src, dst)
	case "cp":
		src := args[1]
		dst := args[2]
		err = d.copy(src, dst)
	default:
		err = fmt.Errorf("action not found, available: mv, cp")
	}

	return err
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
