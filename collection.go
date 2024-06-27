package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Collection is used to process commands related to collection actions.
type Collection struct {
	client *firestore.Client
}

func NewCollection(client *firestore.Client) Collection {
	return Collection{client}
}

func (c Collection) usage() {
	log.SetOutput(os.Stderr)
	log.Fatalln(`
Description:
	Perform actions on firestore collections.
Usage: 
	firestore collection [action] <...args>
Actions:
	cp - copies a collection from the source to the destination, recursively
	rm - deletes a collection
Examples: 
	firestore collection cp /source/collection/path /destination/collection/path
	firestore collection rm /path/to/collection
	`)

}
func (c Collection) checkArgs(args []string, size int) {
	if len(args) < size {
		c.usage()
	}
}

func (c Collection) Run(args []string) error {
	c.checkArgs(args, 1)
	var err error
	action := args[0]

	switch action {
	case "cp":
		c.checkArgs(args, 3)
		src := args[1]
		dst := args[2]
		err = c.copy(src, dst)
	case "rm":
		c.checkArgs(args, 2)
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
	ctx := context.Background()
	doc := NewDocument(c.client)
	errs := []CollectionCopyError{}

	srcRefs := c.client.Collection(strings.TrimPrefix(src, "/")).DocumentRefs(ctx)
	for {
		docRef, err := srcRefs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			errs = append(errs, CollectionCopyError{sourcePath: docRef.Path, action: "reading", err: err})
			continue
		}

		srcPath := strings.Join([]string{src, docRef.ID}, "/")
		dstPath := strings.Join([]string{dst, docRef.ID}, "/")

		// check for subcollections recursively
		iter := c.client.Doc(strings.TrimPrefix(srcPath, "/")).Collections(ctx)
		for {
			collection, err := iter.Next()
			if err == iterator.Done {
				break
			}
			srcSubCollection := strings.Join([]string{srcPath, collection.ID}, "/")
			dstSubCollection := strings.Join([]string{dstPath, collection.ID}, "/")
			if err = c.copy(srcSubCollection, dstSubCollection); err != nil {
				errs = append(errs, CollectionCopyError{sourcePath: srcSubCollection, action: "subcollection copy", err: err, destinationPath: dstSubCollection})
			}
		}

		if err = doc.copy(srcPath, dstPath); err != nil {
			// if the document is NotFound, we don't need to register the error since it might still have subcollections.
			if status.Code(err) != codes.NotFound {
				errs = append(errs, CollectionCopyError{sourcePath: srcPath, action: "copy", err: err, destinationPath: dstPath})
			}
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
