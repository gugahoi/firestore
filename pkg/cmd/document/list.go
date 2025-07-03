package document

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
		Use:     "list <path>",
		Aliases: []string{"ls"},
		Short:   "lists all subcollections in a document",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			return listSubcollections(client, args[0])
		},
	}
	return cmd
}

func listSubcollections(client *firestore.Client, src string) error {
	ctx := context.Background()
	docRef := client.Doc(strings.TrimPrefix(src, "/"))

	iter := docRef.Collections(ctx)

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	count := 0
	for {
		colRef, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list subcollections: %w", err)
		}

		fmt.Fprintf(w, "%s\t%s\n", colRef.ID, colRef.Path)
		count++
	}

	w.Flush()

	if count == 0 {
		fmt.Println("No subcollections found")
	} else {
		fmt.Printf("\nFound %d subcollection(s)\n", count)
	}

	return nil
}

