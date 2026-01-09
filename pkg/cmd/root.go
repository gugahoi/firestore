package cmd

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"

	"github.com/gugahoi/firestore/pkg/cmd/collection"
	"github.com/gugahoi/firestore/pkg/cmd/document"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:     "firestore",
	Aliases: []string{"fs"},
	Short:   "perform actions on firestore",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if projectId == "" {
			envValue := os.Getenv("PROJECT_ID")
			if envValue == "" {
				return fmt.Errorf("missing PROJECT_ID: --project (-p) [PROJECT_ID]")
			}
			projectId = envValue
		}

		ctx := cmd.Context()
		client, err := firestore.NewClient(ctx, projectId)
		if err != nil {
			return fmt.Errorf("failed to create firestore client: %w", err)
		}

		cmd.SetContext(context.WithValue(ctx, keys.ClientKey, client))
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var projectId string

func init() {
	rootCmd.PersistentFlags().StringVarP(&projectId, "project", "p", "", "Google Cloud Project")
	rootCmd.AddCommand(document.NewDocumentCmd())
	rootCmd.AddCommand(collection.NewCollectionCmd())
}
