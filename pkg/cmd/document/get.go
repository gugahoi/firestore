package document

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <path>",
		Short: "retrieves a document and prints its contents to the console",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			return get(client, args[0])
		},
	}
	return cmd
}

// get retrieves the contents of the document and prints it to the console.
func get(client *firestore.Client, src string) error {
	ctx := context.Background()

	srcRef := client.Doc(strings.TrimPrefix(src, "/"))

	snap, err := srcRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("document not found")
		}
		return fmt.Errorf("failed to read document: %v", err)
	}

	// encoder with html escaping disabled
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	// pretty print json data
	if err := encoder.Encode(snap.Data()); err != nil {
		return fmt.Errorf("failed to parse document: %v", err)
	}

	return nil
}
