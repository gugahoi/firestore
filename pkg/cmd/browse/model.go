package browse

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the entire TUI state
type Model struct {
	client       *firestore.Client
	ctx          context.Context
	projectID    string
	columns      []Column // Miller columns (left to right)
	activeColumn int      // Index of currently focused column
	width        int      // Terminal width
	height       int      // Terminal height
	loading      bool
	err          error
}

// Column represents a single column in the Miller columns layout
type Column struct {
	path         string             // Firestore path this column represents
	isDoc        bool               // true if this is a document, false if collection
	sections     []Section          // Documents and/or Subcollections sections
	docContent   string             // JSON content (only for document columns)
	viewport     viewport.Model     // Scrollable viewport for document content
	cursor       int                // Selected index across all items
	scrollOffset int                // Scroll position for lists
}

// Section represents a section within a column (Documents or Subcollections)
type Section struct {
	title  string     // "Documents" or "Subcollections"
	items  []ListItem
	hidden bool       // true if no items (section not displayed)
}

// ListItem represents a single item in a list (document or collection)
type ListItem struct {
	id       string
	path     string
	isDoc    bool              // true for documents, false for collections
	metadata map[string]string // CreateTime, etc.
}

// Message types for async operations
type fetchedColumnMsg struct {
	columnIndex int
	sections    []Section
	docContent  string
}

type errorMsg struct {
	err error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	logDebug("Init called - columns: %d, first column path: '%s', isDoc: %v",
		len(m.columns),
		func() string { if len(m.columns) > 0 { return m.columns[0].path } else { return "" } }(),
		func() bool { if len(m.columns) > 0 { return m.columns[0].isDoc } else { return false } }())

	// Fetch initial data for first column
	if len(m.columns) > 0 {
		return fetchColumnData(m.client, m.columns[0].path, m.columns[0].isDoc, 0)
	}
	return nil
}
