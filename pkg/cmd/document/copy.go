package document

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
)

func NewCopyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "copy <source> <destination>",
		Aliases: []string{"cp"},
		Short:   "copies a document from the source to the destination",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			return Copy(client, args[0], args[1])
		},
	}
	return cmd
}

// copy copies a document from the source to the destination.
// This method is exported for convenience so that we can leverage the same
// logic in the Collection.copy command.
func Copy(client *firestore.Client, src, dst string) error {
	ctx := context.Background()

	srcRef := client.Doc(strings.TrimPrefix(src, "/"))
	snap, err := srcRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to read source document: %v", err)
	}

	dstRef := client.Doc(strings.TrimPrefix(dst, "/"))
	_, err = dstRef.Set(ctx, snap.Data())
	if err != nil {
		return fmt.Errorf("failed to write destination document: %v", err)
	}
	return nil
}
