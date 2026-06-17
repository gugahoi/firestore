package document

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

func NewMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "move <source> <destination>",
		Aliases: []string{"mv"},
		Short:   "moves a document from the source to the destination, deleting the source document",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			return move(client, args[0], args[1])
		},
	}
	return cmd
}

// HasSubcollections reports whether the document has any subcollections. A
// document's subcollections are independent of its fields: they are not
// returned by Get and are not removed by Delete, so a field-only move would
// silently orphan them.
func HasSubcollections(ctx context.Context, ref *firestore.DocumentRef) (bool, error) {
	iter := ref.Collections(ctx)
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// move moves a document from the source to the destination, deleting the
// source. Can be useful when wanting to rename.
func move(client *firestore.Client, src, dst string) error {
	ctx := context.Background()

	srcRef := client.Doc(strings.TrimPrefix(src, "/"))

	hasSub, err := HasSubcollections(ctx, srcRef)
	if err != nil {
		return fmt.Errorf("failed to check source subcollections: %v", err)
	}
	if hasSub {
		return fmt.Errorf("cannot move %s: document has subcollections (not yet supported)", src)
	}

	snap, err := srcRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to read source document: %v", err)
	}

	dstRef := client.Doc(strings.TrimPrefix(dst, "/"))
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
