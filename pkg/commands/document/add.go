package document

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/urfave/cli/v2"
)

func NewAddCmd() *cli.Command {
	return &cli.Command{
		Name:        "add",
		Description: "adds a new document with contents from STDIN",
		Usage:       "firestore document add <path>",
		Action: func(c *cli.Context) error {
			client := c.App.Metadata["client"].(*firestore.Client)
			return add(client, c.Args().First())
		},
	}
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
