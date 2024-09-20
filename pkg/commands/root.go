package commands

import (
	"github.com/gugahoi/firestore/pkg/commands/collection"
	"github.com/gugahoi/firestore/pkg/commands/document"

	"github.com/urfave/cli/v2"
)

func NewRootCmd() *cli.Command {
	return &cli.Command{
		Name:    "firestore",
		Aliases: []string{"fs"},
		Usage:   "perform actions on firestore",
		Subcommands: []*cli.Command{
			document.NewDocumentCmd(),
			collection.NewCollectionCmd(),
		},
	}
}
