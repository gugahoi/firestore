package collection

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [collection]",
		Aliases: []string{"ls"},
		Short:   "lists the documents in a collection",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			return list(client, args[0])
		},
	}
	return cmd
}

func list(client *firestore.Client, src string) error {
	ctx := context.Background()
	col := client.Collection(strings.TrimPrefix(src, "/"))
	iter := col.Documents(ctx)

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, 1, 1)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(w, doc.Ref.ID, doc.CreateTime.Local(), doc.Ref.Parent.Path)
	}
	w.Flush()

	return nil
}
