package document

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/urfave/cli/v2"
)

func NewCopyCmd() *cli.Command {
	return &cli.Command{
		Name:    "copy",
		Aliases: []string{"cp"},
		Usage:   "copies a document from the source to the destination",
		Action: func(c *cli.Context) error {
			client := c.App.Metadata["client"].(*firestore.Client)
			return Copy(client, c.Args().Get(0), c.Args().Get(1))
		},
	}
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
