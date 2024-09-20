package document

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/urfave/cli/v2"
)

func NewMoveCmd() *cli.Command {
	return &cli.Command{
		Name:        "move",
		Aliases:     []string{"mv"},
		Description: "moves a document from the source to the destination, deleting the source document",
		Usage:       "firestore document move <source> <destination>",
		Action: func(c *cli.Context) error {
			client := c.App.Metadata["client"].(*firestore.Client)
			return move(client, c.Args().Get(0), c.Args().Get(1))
		},
	}
}

// move moves a document from the source to the destination, deleting the
// source. Can be useful when wanting to rename.
func move(client *firestore.Client, src, dst string) error {
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

	_, err = srcRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete source document: %v", err)
	}
	return nil
}
