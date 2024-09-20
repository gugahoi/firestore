package collection

import "github.com/urfave/cli/v2"

func NewCollectionCmd() *cli.Command {
	return &cli.Command{
		Name:        "collection",
		Aliases:     []string{"col"},
		Description: "perform actions on firestore collections",
		Subcommands: []*cli.Command{
			NewCopyCmd(),
			NewDeleteCmd(),
			NewListCmd(),
			NewQueryCmd(),
		},
	}
}
