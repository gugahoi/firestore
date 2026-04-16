package auth

import (
	"github.com/spf13/cobra"
)

func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication for Firestore CLI",
		Long: `Set up and manage Google Cloud credentials used by the Firestore CLI.

This tool uses Application Default Credentials (ADC) to authenticate with
Google Cloud. The auth command helps you configure these credentials without
needing to manually install or configure the gcloud CLI.`,
		// Override root's PersistentPreRunE — auth commands must work without
		// existing credentials since their purpose is to create them.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.AddCommand(
		NewLoginCmd(),
		NewStatusCmd(),
		NewRevokeCmd(),
	)

	return cmd
}
