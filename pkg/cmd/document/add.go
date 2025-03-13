package document

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
)

func NewAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: "adds a new document with contents from STDIN",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			return add(client, args[0])
		},
	}
	return cmd
}

// Add adds a document with contents from STDIN
func add(client *firestore.Client, path string) error {
	ctx := context.Background()

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}

	var data map[string]any
	err = json.Unmarshal(input, &data)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	srcRef := client.Doc(strings.TrimPrefix(path, "/"))
	_, err = srcRef.Create(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to create document: %v", err)
	}

	return nil
}
