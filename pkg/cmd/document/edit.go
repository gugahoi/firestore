package document

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/cmd/keys"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewEditCmd() *cobra.Command {
	var set []string

	cmd := &cobra.Command{
		Use:     "edit <path>",
		Aliases: []string{"e"},
		Short:   "edits a document, interactively in $EDITOR or via --set field=value",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := cmd.Context().Value(keys.ClientKey).(*firestore.Client)
			if len(set) > 0 {
				return editFields(client, args[0], set)
			}
			return editInteractive(client, args[0])
		},
	}

	cmd.Flags().StringSliceVar(&set, "set", nil, "field=value pair to update (repeatable, e.g. --set age=42 --set address.city=NYC)")
	return cmd
}

// editFields partially updates the named fields, leaving all other fields untouched.
func editFields(client *firestore.Client, path string, sets []string) error {
	ctx := context.Background()

	updates := make([]firestore.Update, 0, len(sets))
	for _, s := range sets {
		u, err := parseSetArg(s)
		if err != nil {
			return err
		}
		updates = append(updates, u)
	}

	docRef := client.Doc(strings.TrimPrefix(path, "/"))
	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("document not found")
		}
		return fmt.Errorf("failed to update document: %v", err)
	}

	return nil
}

// editInteractive opens the current document contents in $EDITOR and writes the
// edited JSON back, replacing the whole document.
func editInteractive(client *firestore.Client, path string) error {
	ctx := context.Background()

	docRef := client.Doc(strings.TrimPrefix(path, "/"))
	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("document not found")
		}
		return fmt.Errorf("failed to read document: %v", err)
	}

	editor, err := detectEditor()
	if err != nil {
		return err
	}

	content, err := json.MarshalIndent(snap.Data(), "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format document: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "firestore-doc-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// strings.Fields handles editors with args, e.g. EDITOR="code --wait"
	parts := strings.Fields(editor)
	c := exec.Command(parts[0], append(parts[1:], tmpFile.Name())...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %v", err)
	}

	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to read edited file: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(edited, &data); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}

	if _, err := docRef.Set(ctx, data); err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}

	return nil
}

// detectEditor finds the user's preferred editor.
// ponytail: duplicated from browse/editor.go (~12 lines) — browse pulls in Bubble Tea, so
// importing it here would drag the whole TUI into this lightweight command. Extract to a shared
// pkg/cmd/editor package if a third caller ever needs it.
func detectEditor() (string, error) {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor, nil
	}
	for _, editor := range []string{"vim", "vi", "nano"} {
		if _, err := exec.LookPath(editor); err == nil {
			return editor, nil
		}
	}
	return "", fmt.Errorf("no editor found (set $EDITOR)")
}

// parseSetArg splits a "field=value" pair into a firestore.Update. The field may be a dotted
// path (e.g. address.city); firestore.Update.Path interprets dots as field-path separators.
func parseSetArg(s string) (firestore.Update, error) {
	field, value, found := strings.Cut(s, "=")
	if !found || field == "" {
		return firestore.Update{}, fmt.Errorf("invalid --set %q, expected field=value", s)
	}
	return firestore.Update{Path: field, Value: parseValue(value)}, nil
}

// parseValue auto-detects the type of a value string: quoted->string, true/false->bool,
// [..]->JSON array, numeric->number, anything else->bare string.
func parseValue(s string) any {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		var v string
		if err := json.Unmarshal([]byte(s), &v); err == nil {
			return v
		}
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if len(s) >= 2 && s[0] == '[' {
		var arr []any
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return arr
		}
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n
	}
	return s
}
