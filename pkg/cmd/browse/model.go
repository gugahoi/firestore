package browse

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Mode represents the primary vim-style mode
type Mode int

const (
	ModeNormal  Mode = iota
	ModeVisual
	ModeCommand
)

// Overlay represents a dialog/overlay on top of the current mode
type Overlay int

const (
	OverlayNone Overlay = iota
	OverlayPathInput
	OverlaySortDialog
	OverlayDeleteConfirm
	OverlayFilter
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeVisual:
		return "VISUAL"
	case ModeCommand:
		return "COMMAND"
	default:
		return "UNKNOWN"
	}
}

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

	// Mode system
	mode    Mode
	overlay Overlay
	textInput textinput.Model

	// Command mode
	commandRegistry *CommandRegistry
	commandInput    textinput.Model
	commandResult   string
	pendingFetch    *pendingFetchCmd

	// Sort dialog
	sortDialog sortDialogModel
	sortState  map[string]sortStateEntry // Path -> sort state

	// Clipboard copy state
	lastKeyPress  string    // Track last key pressed for double-tap
	lastKeyTime   time.Time // Timestamp of last key press
	statusMsg     string    // Status message to display
	statusMsgTime time.Time // When status message was set

	// Pagination
	pageLimit int

	// Delete confirmation
	deletePath        string // Path of document pending deletion
	deleteFromDocView bool   // Whether delete was initiated from a document column

	// Filter
	filterInput   textinput.Model
	filterActive  bool
	filterPattern string
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
	hasMore         bool                   // true if more pages available
	docCount        int                    // total document count loaded so far
}

// Section represents a section within a column (Documents or Subcollections)
type Section struct {
	title  string // "Documents", "Subcollections", or "Data"
	items  []ListItem
	hidden bool // true if no items (section not displayed)
}

// ListItem represents a single item in a list (document, collection, or data node)
type ListItem struct {
	id        string            // Display ID or Key
	path      string            // Firestore path (for cols/docs) or JSON path (for data)
	isDoc     bool              // true for documents, false for collections
	isData    bool              // true if this is a data node
	isMissing bool              // true if document has no data (only subcollections)
	metadata  map[string]string // CreateTime, etc.

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
	hasMore         bool
	docCount        int
	appendItems     bool // true when loading more pages (append to existing)
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

type documentDeletedMsg struct {
	path        string
	fromDocView bool // true if delete was initiated from a document column
}

// getSortParams returns the saved sort field and direction for a given path
func (m Model) getSortParams(path string) (string, firestore.Direction) {
	if state, ok := m.sortState[path]; ok {
		return state.Field, state.Direction
	}
	return "", firestore.Asc
}

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
		sortField, sortDir := m.getSortParams(m.columns[0].path)
		return tea.Batch(
			textinput.Blink,
			fetchColumnData(m.client, m.columns[0].path, m.columns[0].isDoc, 0, sortField, sortDir, withLimit(m.pageLimit)),
		)
	}
	return textinput.Blink
}
