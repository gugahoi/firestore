package collection

import (
	"context"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/iterator"
)

func NewDeleteCmd() *cli.Command {
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"rm"},
		Usage:   "deletes all documents in a collection",
		Action: func(c *cli.Context) error {
			client := c.App.Metadata["client"].(*firestore.Client)
			return delete(client, c.Args().First())
		},
	}
}

// rm is used to remove all document in a collection.
func delete(client *firestore.Client, src string) error {
	ctx := context.Background()
	col := client.Collection(strings.TrimPrefix(src, "/"))
	bulkwriter := client.BulkWriter(ctx)

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
