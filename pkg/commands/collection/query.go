package collection

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"cloud.google.com/go/firestore"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/iterator"
)

func NewQueryCmd() *cli.Command {
	return &cli.Command{
		Name:    "query",
		Aliases: []string{"q"},
		Usage:   "perform queries on firestore collections",
		Args:    true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "sort",
				Aliases: []string{"s"},
				Usage:   "field to sort by",
			},
			&cli.StringFlag{
				Name:    "fields",
				Aliases: []string{"q"},
				Usage:   "fields to include in the response, comma separated. e.g.: id,name,age",
			},
			&cli.StringFlag{
				Name:    "direction",
				Aliases: []string{"d"},
				Usage:   "direction to sort by (asc|desc)",
			},
			&cli.StringSliceFlag{
				Name:    "filters",
				Aliases: []string{"f"},
				Usage:   "filters to apply to the query, e.g.: id==2",
			},
		},
		Action: func(c *cli.Context) error {
			client := c.App.Metadata["client"].(*firestore.Client)
			orderBy := parseSort(c)
			filters := parseFilters(c)
			fields := parseFields(c)
			return query(client, c.Args().First(), orderBy, filters, fields)
		},
	}
}

func parseFields(c *cli.Context) []string {
	fields := c.String("fields")
	return strings.Split(fields, ",")
}

type Filter struct {
	Field    string
	Operator string
	Value    interface{}
}

var operators = []string{"==", "<", ">", "<=", ">="}

func parseFilters(c *cli.Context) *[]Filter {
	var filters []Filter
	filtersStrings := c.StringSlice("filters")
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

func parseSort(c *cli.Context) *OrderBy {
	field := c.String("sort")

	if field == "" {
		return nil
	}

	var orderBy OrderBy
	orderBy.Path = field

	direction := c.String("direction")
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
