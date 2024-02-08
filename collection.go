package main

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Collection is used to process commands related to collection actions.
type Collection struct {
	client *firestore.Client
}

func NewCollection(client *firestore.Client) Collection {
	return Collection{client}
}

func (c Collection) Run(args []string) error {
	var err error
	action := args[0]

	switch action {
	case "cp":
		src := args[1]
		dst := args[2]
		err = c.copy(src, dst)
	case "rm":
		src := args[1]
		err = c.rm(src)
	default:
		return fmt.Errorf("action not found, available: cp")
	}

	return err
}

// CollectionCopyError is an error returned when copying a document fails during a collection copy action.
type CollectionCopyError struct {
	err             error
	sourcePath      string
	destinationPath string
	action          string
}

// copy is used to copy every document in a collection.
func (c Collection) copy(src, dst string) error {
	doc := NewDocument(c.client)
	errs := []CollectionCopyError{}

	srcRefs := c.client.Collection(src).DocumentRefs(context.Background())
	for {
		docRef, err := srcRefs.Next()
		if err == iterator.Done {
			break
		}
		// TODO: should we do something about this error case?
		// if err != nil {
		// 	readErrors = append(readErrors, docRef.Path+"(ref-error)")
		// }

		srcPath := strings.Join([]string{src, docRef.ID}, "/")
		dstPath := strings.Join([]string{dst, docRef.ID}, "/")
		fmt.Println("src:", srcPath, "dst:", dstPath)

		if err = doc.copy(srcPath, dstPath); err != nil {
			errs = append(errs, CollectionCopyError{sourcePath: docRef.Path, action: "copy", err: err, destinationPath: dstPath})
		}
	}

	if len(errs) != 0 {
		return fmt.Errorf("the following errors occurred: %v", errs)
	}
	return nil
}

// rm is used to remove all document in a collection.
func (c Collection) rm(src string) error {
	ctx := context.Background()
	col := c.client.Collection(strings.TrimPrefix(src, "/"))
	bulkwriter := c.client.BulkWriter(ctx)

	for {
		// Get a batch of documents
		iter := col.Limit(10).Documents(ctx)
		numDeleted := 0

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}

			bulkwriter.Delete(doc.Ref)
			numDeleted++
		}

		// If there are no documents to delete,
		// the process is over.
		if numDeleted == 0 {
			bulkwriter.End()
			break
		}

		bulkwriter.Flush()
	}

	return nil
}
