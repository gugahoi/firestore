package browse

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// InputMode indicates whether the user is typing a path
type InputMode int

const (
	ModeNormal InputMode = iota
	ModePathInput
	ModeSortDialog
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

	// Path input
	mode      InputMode
	textInput textinput.Model

	// Sort dialog
	sortDialog sortDialogModel
	sortState  map[string]sortStateEntry // Path -> sort state

	// Clipboard copy state
	lastKeyPress  string    // Track last key pressed for double-tap
	lastKeyTime   time.Time // Timestamp of last key press
	statusMsg     string    // Status message to display
	statusMsgTime time.Time // When status message was set
}

// sortStateEntry stores sort configuration for a collection path
type sortStateEntry struct {
	Field     string
	Direction firestore.Direction
}

// Column represents a single column in the Miller columns layout
type Column struct {
	path            string                 // Firestore path this column represents
	isDoc           bool                   // true if this is a document, false if collection
	sections        []Section              // Documents and/or Subcollections sections
	docContent      string                 // Raw JSON content (only for document columns)
	docData         map[string]interface{} // Parsed data map
	docMetadata     map[string]string      // Document metadata (CreateTime, UpdateTime, etc.)
	availableFields []string               // Field names from first document (for collections)
	viewport        viewport.Model         // Scrollable viewport for document content
	cursor          int                    // Selected index across all items
	scrollOffset    int                    // Scroll position for lists
}

// Section represents a section within a column (Documents or Subcollections)
type Section struct {
	title  string // "Documents", "Subcollections", or "Data"
	items  []ListItem
	hidden bool // true if no items (section not displayed)
}

// ListItem represents a single item in a list (document, collection, or data node)
type ListItem struct {
	id       string            // Display ID or Key
	path     string            // Firestore path (for cols/docs) or JSON path (for data)
	isDoc    bool              // true for documents, false for collections
	isData   bool              // true if this is a data node
	metadata map[string]string // CreateTime, etc.

	// Tree view fields
	key      string
	valueStr string     // string representation of value
	dataType string     // "string", "number", "bool", "object", "array", "null"
	depth    int        // indentation level
	expanded bool       // is expanded?
	children []ListItem // children nodes (if object/array)
}

// Message types for async operations
type fetchedColumnMsg struct {
	columnIndex     int
	sections        []Section
	docContent      string
	docData         map[string]interface{}
	docMetadata     map[string]string
	availableFields []string
}

type errorMsg struct {
	err error
}

type statusMsg struct {
	message string
}

type clearStatusMsg struct{}

type documentUpdatedMsg struct {
	path string
	data map[string]interface{}
}

type sortAppliedMsg struct{}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	logDebug("Init called - columns: %d, first column path: '%s', isDoc: %v",
		len(m.columns),
		func() string {
			if len(m.columns) > 0 {
				return m.columns[0].path
			} else {
				return ""
			}
		}(),
		func() bool {
			if len(m.columns) > 0 {
				return m.columns[0].isDoc
			} else {
				return false
			}
		}())

	// Fetch initial data for first column
	if len(m.columns) > 0 {
		// Check if there's a saved sort state for this path
		sortField := ""
		sortDir := firestore.Asc
		if state, ok := m.sortState[m.columns[0].path]; ok {
			sortField = state.Field
			sortDir = state.Direction
		}

		return tea.Batch(
			textinput.Blink,
			fetchColumnData(m.client, m.columns[0].path, m.columns[0].isDoc, 0, sortField, sortDir),
		)
	}
	return textinput.Blink
}
