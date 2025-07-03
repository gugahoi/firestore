package collection

import (
	"github.com/spf13/cobra"
)

/**
* `NewCollectionCmd` generates commands that can be executed on Firestore Collections.
 */
func NewCollectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "collection",
		Aliases:      []string{"col"},
		Short:        "perform actions on firestore collections",
		SilenceUsage: true,
	}

	cmd.AddCommand(
		NewCopyCmd(),
		NewDeleteCmd(),
		NewDownloadCmd(),
		NewListCmd(),
		NewQueryCmd(),
		NewUploadCmd(),
	)

	return cmd
}
