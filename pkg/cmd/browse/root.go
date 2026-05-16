package browse

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/charmbracelet/bubbles/textinput"
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

Navigation:
  j/k or Up/Down       Move cursor up/down
  h/l or Left/Right    Navigate back/forward through columns
  Enter                Navigate forward / toggle tree node
  Space                Toggle tree node expansion
  g/G                  Jump to top/bottom of list
  Ctrl+d/Ctrl+u        Page down/up
  Ctrl+g               Go to a specific Firestore path

Sorting (collections only):
  s                    Open sort dialog
  S                    Clear sort and refresh
  Sort dialog controls:
    Tab                Switch focus between text input and field list
    Ctrl+d             Toggle direction (Ascending/Descending)
    Enter              Apply sort
    Esc                Cancel

Editing:
  e                    Edit document in $EDITOR
  d                    Delete document (with confirmation)

Clipboard:
  yy                   Copy selected value
  Y                    Copy document/collection ID
  ya                   Copy entire document as JSON

Other:
  r                    Refresh current column
  q/Esc                Quit

Examples:
  firestore browse                  Browse from root collections
  firestore browse users            Start at 'users' collection
  firestore browse users/abc123     Start at a specific document`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			ctx := context.Background()

			// Get project ID from context
			projectID, ok := cmd.Context().Value(keys.ProjectIDKey).(string)
			if !ok {
				// Fallback to "unknown" if not found (shouldn't happen with root's persistentPreRun)
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

			// Initialize text input
			ti := textinput.New()
			ti.Placeholder = "Collection/Document/Collection..."
			ti.CharLimit = 150
			ti.Width = 50

			// Initialize command input
			ci := textinput.New()
			ci.Placeholder = ""
			ci.CharLimit = 200
			ci.Width = 50
			ci.Prompt = ":"

			// Initialize filter input
			fi := textinput.New()
			fi.Placeholder = ""
			fi.CharLimit = 100
			fi.Width = 50
			fi.Prompt = "/"

			// Initialize model
			m := Model{
				client:          client,
				ctx:             ctx,
				projectID:       projectID,
				columns:         []Column{{path: startPath, isDoc: isDoc}},
				activeColumn:    0,
				width:           0,
				height:          0,
				loading:         true,
				err:             nil,
				textInput:       ti,
				sortState:       make(map[string]sortStateEntry),
				mode:            ModeNormal,
				commandRegistry: initCommandRegistry(),
				commandInput:    ci,
				filterInput:     fi,
				pageLimit:       50,
				selection:       NewSelection(),
				marks:           make(map[rune]markEntry),
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
