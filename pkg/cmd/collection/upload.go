package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
)

func NewUploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "upload [collection] [folder]",
		Aliases: []string{"up"},
		Short:   "uploads JSON files from a folder to documents in a collection",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			return upload(client, args[0], args[1])
		},
	}

	return cmd
}

func upload(client *firestore.Client, collectionPath string, folderPath string) error {
	ctx := context.Background()
	col := client.Collection(strings.TrimPrefix(collectionPath, "/"))

	// Check if folder exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return fmt.Errorf("folder does not exist: %s", folderPath)
	}

	// Read directory contents
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(folderPath, file.Name())

		// Read file contents
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Failed to read file %s: %v\n", file.Name(), err)
			continue
		}

		// Parse JSON content
		var docData map[string]interface{}
		if err := json.Unmarshal(data, &docData); err != nil {
			fmt.Printf("Failed to parse JSON in file %s: %v\n", file.Name(), err)
			continue
		}

		// Use filename without extension as document ID
		docID := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

		// Create document
		_, err = col.Doc(docID).Set(ctx, docData)
		if err != nil {
			fmt.Printf("Failed to create document %s: %v\n", docID, err)
			continue
		}

		count++
		fmt.Printf("Uploaded: %s -> %s\n", file.Name(), docID)
	}

	fmt.Printf("Successfully uploaded %d documents to collection %s\n", count, collectionPath)
	return nil
}
