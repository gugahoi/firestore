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

func NewQueryCmd() *cobra.Command {
	var (
		sort      string
		fields    string
		direction string
		filters   []string
	)

	cmd := &cobra.Command{
		Use:     "query [collection]",
		Aliases: []string{"q"},
		Short:   "perform queries on firestore collections",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			orderBy := parseSort(sort, direction)
			parsedFilters := parseFilters(filters)
			parsedFields := parseFields(fields)
			return query(client, args[0], orderBy, parsedFilters, parsedFields)
		},
	}

	cmd.Flags().StringVarP(&sort, "sort", "s", "", "field to sort by")
	cmd.Flags().StringVarP(&fields, "fields", "q", "", "fields to include in the response, comma separated. e.g.: id,name,age")
	cmd.Flags().StringVarP(&direction, "direction", "d", "", "direction to sort by (asc|desc)")
	cmd.Flags().StringSliceVarP(&filters, "filters", "f", []string{}, "filters to apply to the query, e.g.: id==2")

	return cmd
}

func parseFields(fields string) []string {
	return strings.Split(fields, ",")
}

type Filter struct {
	Field    string
	Operator string
	Value    any
}

var operators = []string{"==", "<", ">", "<=", ">="}

func parseFilters(filtersStrings []string) *[]Filter {
	var filters []Filter
	for _, filter := range filtersStrings {
		for _, operator := range operators {
			parsed := strings.Split(filter, operator)
			if len(parsed) > 1 {
				filters = append(filters, Filter{
					Field:    parsed[0],
					Operator: operator,
					Value:    parsed[1],
				})
				break
			}
		}
	}
	return &filters
}

type OrderBy struct {
	Path      string
	Direction firestore.Direction
}

func parseSort(field, direction string) *OrderBy {
	if field == "" {
		return nil
	}

	var orderBy OrderBy
	orderBy.Path = field

	switch direction {
	case "desc":
		orderBy.Direction = firestore.Desc
	default:
		orderBy.Direction = firestore.Asc
	}

	return &orderBy
}

func query(client *firestore.Client, path string, orderBy *OrderBy, filters *[]Filter, fields []string) error {
	collection := client.Collection(strings.TrimPrefix(path, "/"))
	if collection == nil {
		return fmt.Errorf("invalid path: %q", path)
	}

	query := collection.Query
	if orderBy != nil {
		query = collection.OrderBy(orderBy.Path, orderBy.Direction)
	}
	for _, filter := range *filters {
		query = query.Where(filter.Field, filter.Operator, filter.Value)
	}

	query = query.Select(fields...)

	iter := query.Documents(context.Background())
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 1, ' ', 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%+v\n", doc.Data())
	}
	w.Flush()

	return nil
}
