package document

import (
	"github.com/spf13/cobra"
)

func NewDocumentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "document",
		Aliases: []string{"doc"},
		Short:   "perform actions on firestore documents",
	}

	cmd.AddCommand(
		NewAddCmd(),
		NewCopyCmd(),
		NewDeleteCmd(),
		NewEditCmd(),
		NewGetCmd(),
		NewListCmd(),
		NewMoveCmd(),
	)
	return cmd
}
