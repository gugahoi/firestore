package browse

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cloud.google.com/go/firestore"
	tea "github.com/charmbracelet/bubbletea"
)

// editSession holds state for an ongoing edit session
type editSession struct {
	client   *firestore.Client
	docPath  string
	tempFile string
	editor   string
}

// detectEditor finds the user's preferred editor
func detectEditor() (string, error) {
	// Check environment variables
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor, nil
	}

	// Try fallbacks
	fallbacks := []string{"vim", "vi", "nano"}
	for _, editor := range fallbacks {
		if _, err := exec.LookPath(editor); err == nil {
			return editor, nil
		}
	}

	return "", fmt.Errorf("no editor found (set $EDITOR)")
}

// formatJSON pretty-prints JSON with 2-space indentation
func formatJSON(data map[string]interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// createTempFile creates a temporary file with JSON content
func createTempFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "firestore-doc-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Write the content
	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	// Sync to ensure content is flushed to disk before editor opens
	if err := tmpFile.Sync(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to sync temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// readAndValidateJSON reads and parses JSON from file
func readAndValidateJSON(filepath string) (map[string]interface{}, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return data, nil
}

// startEditCmd initializes an edit session and launches the editor
func startEditCmd(client *firestore.Client, docPath string, docData map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		// 1. Detect editor
		editor, err := detectEditor()
		if err != nil {
			return errorMsg{err: err}
		}

		logDebug("Starting edit for document: %s", docPath)
		logDebug("Document data: %+v", docData)

		// 2. Format current document as JSON
		jsonContent, err := formatJSON(docData)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to format document: %w", err)}
		}

		logDebug("Formatted JSON length: %d bytes", len(jsonContent))

		// 3. Create temp file
		tempFile, err := createTempFile(jsonContent)
		if err != nil {
			return errorMsg{err: err}
		}

		logDebug("Created temp file: %s", tempFile)

		// 4. Create edit session and launch editor
		session := editSession{
			client:   client,
			docPath:  docPath,
			tempFile: tempFile,
			editor:   editor,
		}

		return launchEditorMsg{session: session}
	}
}

// launchEditorMsg triggers the editor to open
type launchEditorMsg struct {
	session editSession
}

// editorFinishedMsg is sent when the editor closes
type editorFinishedMsg struct {
	session editSession
	err     error
}

// openEditorCmd returns a command that opens the editor using tea.ExecProcess
func openEditorCmd(session editSession) tea.Cmd {
	c := exec.Command(session.editor, session.tempFile)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{
			session: session,
			err:     err,
		}
	})
}

// handleEditorFinished processes the result after editor closes
func handleEditorFinished(session editSession) tea.Cmd {
	return func() tea.Msg {
		// Read and validate JSON
		newData, err := readAndValidateJSON(session.tempFile)
		if err != nil {
			// Invalid JSON - write error to file and reopen editor
			logDebug("JSON validation error: %v", err)

			content, _ := os.ReadFile(session.tempFile)
			errorComment := fmt.Sprintf("// ERROR: %s\n// Please fix the JSON and save again\n\n", err.Error())
			os.WriteFile(session.tempFile, append([]byte(errorComment), content...), 0644)

			// Return message to reopen editor
			return launchEditorMsg{session: session}
		}

		// Valid JSON - save to Firestore
		ctx := context.Background()
		docRef := session.client.Doc(strings.TrimPrefix(session.docPath, "/"))
		_, err = docRef.Set(ctx, newData)

		// Clean up temp file
		os.Remove(session.tempFile)

		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to save document: %w", err)}
		}

		// Return success message
		return documentUpdatedMsg{
			path: session.docPath,
			data: newData,
		}
	}
}
