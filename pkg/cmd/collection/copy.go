package collection

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/cmd/document"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewCopyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "copy [source] [destination]",
		Aliases: []string{"cp"},
		Short:   "copies all documents in a collection",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			return copy(client, args[0], args[1])
		},
	}
	return cmd
}

// CollectionCopyError is an error returned when copying a document fails during a collection copy action.
type CollectionCopyError struct {
	err             error
	sourcePath      string
	destinationPath string
	action          string
}

// copy is used to copy every document in a collection.
func copy(client *firestore.Client, src, dst string) error {
	ctx := context.Background()
	errs := []CollectionCopyError{}

	srcRefs := client.Collection(strings.TrimPrefix(src, "/")).DocumentRefs(ctx)
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
		iter := client.Doc(strings.TrimPrefix(srcPath, "/")).Collections(ctx)
		for {
			collection, err := iter.Next()
			if err == iterator.Done {
				break
			}
			srcSubCollection := strings.Join([]string{srcPath, collection.ID}, "/")
			dstSubCollection := strings.Join([]string{dstPath, collection.ID}, "/")
			if err = copy(client, srcSubCollection, dstSubCollection); err != nil {
				errs = append(errs, CollectionCopyError{sourcePath: srcSubCollection, action: "subcollection copy", err: err, destinationPath: dstSubCollection})
			}
		}

		if err = document.Copy(client, srcPath, dstPath); err != nil {
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
