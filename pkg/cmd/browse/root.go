package browse

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
)

// NewBrowseCmd creates the browse command
func NewBrowseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse [path]",
		Short: "Interactive browser for Firestore database",
		Long: `Launch an interactive TUI to browse collections, documents, and subcollections.

Examples:
  firestore browse                  Browse from root collections
  firestore browse users            Start at 'users' collection
  firestore browse users/abc123     Start at a specific document`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			ctx := context.Background()

			// Get project ID from environment or flag
			projectID := os.Getenv("PROJECT_ID")
			if projectID == "" {
				projectID = "unknown"
			}

			// Determine starting path
			startPath := ""
			isDoc := false
			if len(args) > 0 {
				startPath = strings.TrimPrefix(args[0], "/")
				// Determine if path is a document (odd number of segments) or collection (even)
				segments := strings.Split(startPath, "/")
				isDoc = len(segments)%2 == 0
			}

			// Initialize model
			m := Model{
				client:       client,
				ctx:          ctx,
				projectID:    projectID,
				columns:      []Column{{path: startPath, isDoc: isDoc}},
				activeColumn: 0,
				width:        0,
				height:       0,
				loading:      true,
				err:          nil,
			}

			// Run TUI
			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("failed to run TUI: %w", err)
			}

			return nil
		},
	}
	return cmd
}
