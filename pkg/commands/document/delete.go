package document

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/urfave/cli/v2"
)

func NewDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"rm"},
		Usage:   "deletes a document",
		Action: func(c *cli.Context) error {
			client := c.App.Metadata["client"].(*firestore.Client)
			return delete(client, c.Args().First())
		},
	}
}

func delete(client *firestore.Client, src string) error {
	ctx := context.Background()

	srcRef := client.Doc(strings.TrimPrefix(src, "/"))
	_, err := srcRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete document: %v", err)
	}

	return nil
}
