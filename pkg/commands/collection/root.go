package collection

import "github.com/urfave/cli/v2"

func NewCollectionCmd() *cli.Command {
	return &cli.Command{
		Name:    "collection",
		Aliases: []string{"col"},
		Usage:   "perform actions on firestore collections",
		Subcommands: []*cli.Command{
			NewDeleteCmd(),
			NewListCmd(),
			NewCopyCmd(),
		},
	}
}
