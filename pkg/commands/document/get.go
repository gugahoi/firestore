package document

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewGetCmd() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Description: "retrieves a document and prints its contents to the console",
		Usage:       "firestore document get <path>",
		Action: func(c *cli.Context) error {
			client := c.App.Metadata["client"].(*firestore.Client)
			return get(client, c.Args().First())
		},
	}
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
