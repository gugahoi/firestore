package collection

import (
	"context"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/iterator"
)

func NewListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "lists the documents in a collection",
		Action: func(c *cli.Context) error {
			client := c.App.Metadata["client"].(*firestore.Client)
			return list(client, c.Args().First())
		},
	}
}

func list(client *firestore.Client, src string) error {
	ctx := context.Background()
	col := client.Collection(strings.TrimPrefix(src, "/"))
	iter := col.Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		log.Println(doc.Ref.ID)
	}

	return nil
}
