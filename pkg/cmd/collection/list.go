package collection

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"cloud.google.com/go/firestore"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"

	"github.com/gugahoi/firestore/pkg/cmd/keys"
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

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\t%s\n", doc.Ref.ID, doc.CreateTime.Local().Format("2006-01-02 15:04:05"))
	}
	w.Flush()

	return nil
}
