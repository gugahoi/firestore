package browse

import (
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// handleCopy copies the appropriate content based on current selection
func (m Model) handleCopy() (tea.Model, tea.Cmd) {
	item := m.getSelectedItem()
	if item == nil {
		m.statusMsg = "Nothing to copy"
		m.statusMsgTime = time.Now()
		return m, clearStatusAfterDelay()
	}

	var contentToCopy string

	if item.isData {
		// For data fields in tree view, copy the value
		contentToCopy = item.valueStr

		// Special handling for different data types
		switch item.dataType {
		case "string", "number", "bool", "null":
			// Use the string representation
			contentToCopy = item.valueStr
		case "object", "array":
			// For complex types, inform user these can't be copied directly
			m.statusMsg = "Cannot copy object/array (expand to copy fields)"
			m.statusMsgTime = time.Now()
			return m, clearStatusAfterDelay()
		}
	} else {
		// For documents/collections, copy the full Firestore path
		contentToCopy = item.path
	}

	// Attempt to write to clipboard
	err := clipboard.WriteAll(contentToCopy)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Copy failed: %v", err)
		m.statusMsgTime = time.Now()
		logDebug("Clipboard write error: %v", err)
		return m, clearStatusAfterDelay()
	}

	// Success!
	m.statusMsg = fmt.Sprintf("Copied: %s", truncateForDisplay(contentToCopy, 50))
	m.statusMsgTime = time.Now()
	logDebug("Copied to clipboard: %s", contentToCopy)

	return m, clearStatusAfterDelay()
}

// handleCopyAlternate copies the ID/key based on current selection
// For documents/collections: copies just the ID
// For data fields: copies the field name/key
func (m Model) handleCopyAlternate() (tea.Model, tea.Cmd) {
	item := m.getSelectedItem()
	if item == nil {
		m.statusMsg = "Nothing to copy"
		m.statusMsgTime = time.Now()
		return m, clearStatusAfterDelay()
	}

	var contentToCopy string
	var description string

	if item.isData {
		// For data fields in tree view, copy the key/field name
		contentToCopy = item.key
		description = "key"
	} else {
		// For documents/collections, copy just the ID
		contentToCopy = item.id
		description = "ID"
	}

	// Attempt to write to clipboard
	err := clipboard.WriteAll(contentToCopy)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Copy failed: %v", err)
		m.statusMsgTime = time.Now()
		logDebug("Clipboard write error: %v", err)
		return m, clearStatusAfterDelay()
	}

	// Success!
	m.statusMsg = fmt.Sprintf("Copied %s: %s", description, truncateForDisplay(contentToCopy, 45))
	m.statusMsgTime = time.Now()
	logDebug("Copied %s to clipboard: %s", description, contentToCopy)

	return m, clearStatusAfterDelay()
}

// handleCopyDocument copies the entire document contents as JSON
func (m Model) handleCopyDocument() (tea.Model, tea.Cmd) {
	// Verify we're viewing a document
	if m.activeColumn >= len(m.columns) {
		m.statusMsg = "No document to copy"
		m.statusMsgTime = time.Now()
		return m, clearStatusAfterDelay()
	}

	col := m.columns[m.activeColumn]

	// Check if current column is a document
	if !col.isDoc {
		m.statusMsg = "Can only copy document contents when viewing a document"
		m.statusMsgTime = time.Now()
		return m, clearStatusAfterDelay()
	}

	// Get document content
	contentToCopy := col.docContent
	if contentToCopy == "" {
		contentToCopy = "{}"
	}

	// Attempt to write to clipboard
	err := clipboard.WriteAll(contentToCopy)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Copy failed: %v", err)
		m.statusMsgTime = time.Now()
		logDebug("Clipboard write error: %v", err)
		return m, clearStatusAfterDelay()
	}

	// Success! Show size of copied content
	size := len(contentToCopy)
	m.statusMsg = fmt.Sprintf("Copied document (%d bytes)", size)
	m.statusMsgTime = time.Now()
	logDebug("Copied document to clipboard: %d bytes", size)

	return m, clearStatusAfterDelay()
}

// clearStatusAfterDelay returns a command that sends clearStatusMsg after 3 seconds
func clearStatusAfterDelay() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// truncateForDisplay truncates a string for display in status
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
