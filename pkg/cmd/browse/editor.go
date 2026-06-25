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
	isCreate bool   // true when creating a new document
	colPath  string // collection path (for create mode)
	colIndex int    // column index to refresh after create
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

		// Hovered-document edit: data not loaded yet, fetch it.
		if docData == nil {
			ctx := context.Background()
			snap, err := client.Doc(strings.TrimPrefix(docPath, "/")).Get(ctx)
			if err != nil {
				return errorMsg{err: fmt.Errorf("failed to load document: %w", err)}
			}
			docData = snap.Data()
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

// generateTemplate creates a JSON template from the first document's field structure
func generateTemplate(sections []Section) string {
	for _, s := range sections {
		if s.title == "Documents" && len(s.items) > 0 {
			// We can't derive the template from ListItems directly since they don't have full data.
			// The template will be based on available fields from the column.
			break
		}
	}
	return "{}"
}

// generateTemplateFromFields creates a JSON template with null values for each field
func generateTemplateFromFields(fields []string) string {
	if len(fields) == 0 {
		return "{}"
	}
	data := make(map[string]interface{})
	for _, f := range fields {
		data[f] = nil
	}
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

// startAddCmd initializes a create session and launches the editor
func startAddCmd(client *firestore.Client, colPath string, docID string, availableFields []string, colIndex int) tea.Cmd {
	return func() tea.Msg {
		editor, err := detectEditor()
		if err != nil {
			return errorMsg{err: err}
		}

		template := generateTemplateFromFields(availableFields)

		tempFile, err := createTempFile(template)
		if err != nil {
			return errorMsg{err: err}
		}

		session := editSession{
			client:   client,
			docPath:  colPath + "/" + docID,
			tempFile: tempFile,
			editor:   editor,
			isCreate: true,
			colPath:  colPath,
			colIndex: colIndex,
		}

		if docID == "" {
			session.docPath = ""
		}

		return launchEditorMsg{session: session}
	}
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

		ctx := context.Background()

		if session.isCreate {
			// Create new document
			var docRef *firestore.DocumentRef
			colRef := session.client.Collection(strings.TrimPrefix(session.colPath, "/"))

			if session.docPath == "" {
				// Auto-generated ID
				docRef = colRef.NewDoc()
			} else {
				// Custom ID
				parts := strings.Split(session.docPath, "/")
				docID := parts[len(parts)-1]
				docRef = colRef.Doc(docID)
			}

			_, err = docRef.Set(ctx, newData)

			os.Remove(session.tempFile)

			if err != nil {
				return errorMsg{err: fmt.Errorf("failed to create document: %w", err)}
			}

			return documentCreatedMsg{
				colPath:  session.colPath,
				docID:    docRef.ID,
				colIndex: session.colIndex,
			}
		}

		// Update existing document
		docRef := session.client.Doc(strings.TrimPrefix(session.docPath, "/"))
		_, err = docRef.Set(ctx, newData)

		os.Remove(session.tempFile)

		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to save document: %w", err)}
		}

		return documentUpdatedMsg{
			path: session.docPath,
			data: newData,
		}
	}
}
