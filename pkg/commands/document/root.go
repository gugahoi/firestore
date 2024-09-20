package document

import "github.com/urfave/cli/v2"

func NewDocumentCmd() *cli.Command {
	return &cli.Command{
		Name:    "document",
		Aliases: []string{"doc"},
		Usage:   "perform actions on firestore documents",
		Subcommands: []*cli.Command{
			NewAddCmd(),
			NewCopyCmd(),
			NewDeleteCmd(),
			NewGetCmd(),
			NewMoveCmd(),
		},
	}
}
