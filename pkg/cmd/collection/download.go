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
	"google.golang.org/api/iterator"
)

func NewDownloadCmd() *cobra.Command {
	var (
		outputDir string
		filters   []string
		limit     int
		sort      string
		direction string
	)

	cmd := &cobra.Command{
		Use:     "download [collection]",
		Aliases: []string{"dl"},
		Short:   "downloads all documents in a collection to JSON files",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			parsedFilters := parseFilters(filters)
			orderBy := parseSort(sort, direction)
			return download(client, args[0], outputDir, limit, parsedFilters, orderBy)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "output directory for downloaded files")
	cmd.Flags().
		StringSliceVarP(&filters, "filters", "f", []string{}, "filters to apply to the query, e.g.: id==2")
	cmd.Flags().
		IntVarP(&limit, "limit", "l", 0, "maximum number of documents to return, 0 for no limit")
	cmd.Flags().StringVarP(&sort, "sort", "s", "", "field to sort by")
	cmd.Flags().StringVarP(&direction, "direction", "d", "", "direction to sort by (asc|desc)")

	return cmd
}

func download(client *firestore.Client, src string, outputDir string, limit int, filters *[]Filter, orderBy *OrderBy) error {
	ctx := context.Background()
	col := client.Collection(strings.TrimPrefix(src, "/"))

	query := col.Query
	if orderBy != nil {
		query = col.OrderBy(orderBy.Path, orderBy.Direction)
	}
	if filters != nil {
		for _, filter := range *filters {
			query = query.Where(filter.Field, filter.Operator, filter.Value)
		}
	}
	if limit != 0 {
		query = query.Limit(limit)
	}

	iter := query.Documents(ctx)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate documents: %w", err)
		}

		// Create filename from document ID
		filename := fmt.Sprintf("%s.json", doc.Ref.ID)
		filepath := filepath.Join(outputDir, filename)

		// Create file
		file, err := os.Create(filepath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filepath, err)
		}

		// Write document data as JSON
		encoder := json.NewEncoder(file)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(doc.Data()); err != nil {
			file.Close()
			return fmt.Errorf("failed to encode document %s: %w", doc.Ref.ID, err)
		}

		file.Close()
		count++
		fmt.Printf("Downloaded: %s\n", filename)
	}

	fmt.Printf("Successfully downloaded %d documents to %s\n", count, outputDir)
	return nil
}
