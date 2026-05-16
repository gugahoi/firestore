package browse

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		logDebug("KeyMsg received: %s (mode=%s, overlay=%d)", msg.String(), m.mode, m.overlay)

		// Handle overlays first (dialogs on top of any mode)
		if m.overlay != OverlayNone {
			return m.handleOverlay(msg)
		}

		// Route through mode system
		switch m.mode {
		case ModeNormal:
			// Global keys available in normal mode
			if msg.String() == "ctrl+g" {
				m.overlay = OverlayPathInput
				m.textInput.Focus()
				return m, textinput.Blink
			}
			return m.handleKeyPress(msg)

		case ModeVisual:
			if msg.String() == "esc" {
				m.mode = ModeNormal
				return m, nil
			}
			return m, nil

		case ModeCommand:
			if msg.String() == "esc" {
				m.mode = ModeNormal
				return m, nil
			}
			return m, nil
		}

	case tea.WindowSizeMsg:

		logDebug("WindowSizeMsg received: width=%d, height=%d", msg.Width, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport sizes for document columns
		for i := range m.columns {
			if m.columns[i].isDoc && m.columns[i].docContent != "" {
				colWidth := calculateColumnWidth(m.width, len(m.columns))
				colHeight := m.height - 7 // Account for header and footer
				vpWidth := colWidth - 4
				vpHeight := colHeight - 10
				// Ensure minimum viewport dimensions (at least 20x5)
				if vpWidth >= 20 && vpHeight >= 5 {
					// Initialize viewport if it doesn't exist yet
					if m.columns[i].viewport.Width == 0 {
						m.columns[i].viewport = viewport.New(vpWidth, vpHeight)
						m.columns[i].viewport.SetContent(m.columns[i].docContent)
					} else {
						// Just update dimensions
						m.columns[i].viewport.Width = vpWidth
						m.columns[i].viewport.Height = vpHeight
					}
				}
			}
		}
		return m, nil

	case fetchedColumnMsg:
		logDebug("fetchedColumnMsg received: columnIndex=%d, sections=%d, docContentLen=%d",
			msg.columnIndex, len(msg.sections), len(msg.docContent))
		// Update the specified column with fetched data
		if msg.columnIndex < len(m.columns) {
			m.columns[msg.columnIndex].sections = msg.sections
			m.columns[msg.columnIndex].docContent = msg.docContent
			m.columns[msg.columnIndex].docData = msg.docData
			m.columns[msg.columnIndex].docMetadata = msg.docMetadata
			if len(msg.availableFields) > 0 {
				m.columns[msg.columnIndex].availableFields = msg.availableFields
			}
			m.columns[msg.columnIndex].scrollOffset = 0 // Reset scroll to top when new data arrives

			// Initialize viewport if this is a document column
			// Only create viewport if we have valid dimensions
			if m.columns[msg.columnIndex].isDoc && msg.docContent != "" && m.width > 0 && m.height > 0 {
				colWidth := calculateColumnWidth(m.width, len(m.columns))
				colHeight := m.height - 7
				// Ensure minimum viewport dimensions (at least 20x5)
				vpWidth := colWidth - 4
				vpHeight := colHeight - 10
				if vpWidth >= 20 && vpHeight >= 5 {
					m.columns[msg.columnIndex].viewport = viewport.New(vpWidth, vpHeight)
					m.columns[msg.columnIndex].viewport.SetContent(msg.docContent)
				}
			}
		}
		m.loading = false
		return m, nil

	case errorMsg:
		logDebug("errorMsg received: %v", msg.err)
		m.err = msg.err
		m.loading = false
		return m, nil

	case clearStatusMsg:
		// Clear status message if it's old enough
		if time.Since(m.statusMsgTime) >= 3*time.Second {
			m.statusMsg = ""
		}
		return m, nil

	case documentUpdatedMsg:
		// Update the column data with new content
		if m.activeColumn < len(m.columns) {
			col := &m.columns[m.activeColumn]
			if col.path == msg.path {
				col.docData = msg.data

				// Re-format JSON for display
				jsonBytes, _ := json.MarshalIndent(msg.data, "", "  ")
				col.docContent = string(jsonBytes)

				// Regenerate tree view sections
				sections := []Section{
					{
						title: "Data",
						items: buildTreeNodes(msg.data),
					},
				}

				// Add existing Metadata section if present
				if len(col.docMetadata) > 0 {
					var metadataItems []ListItem
					keys := []string{"Created", "Updated", "Read"}
					for _, k := range keys {
						if v, ok := col.docMetadata[k]; ok {
							metadataItems = append(metadataItems, ListItem{
								id:     fmt.Sprintf("%s: %s", k, v),
								isData: false,
								isDoc:  false,
							})
						}
					}
					sections = append(sections, Section{
						title:  "Metadata",
						items:  metadataItems,
						hidden: false,
					})
				}

				col.sections = sections
			}
		}

		m.statusMsg = "Document updated successfully"
		m.statusMsgTime = time.Now()
		m.loading = false

		return m, clearStatusAfterDelay()

	case documentDeletedMsg:
		m.loading = false

		if msg.fromDocView {
			// Delete was from a document column — go back to parent collection
			m.removeLastColumn()
		}

		// Refresh the now-active column (the parent collection)
		if m.activeColumn < len(m.columns) {
			col := m.columns[m.activeColumn]
			sortField, sortDir := m.getSortParams(col.path)

			m.statusMsg = "Document deleted"
			m.statusMsgTime = time.Now()
			m.loading = true

			return m, tea.Batch(
				fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn, sortField, sortDir),
				clearStatusAfterDelay(),
			)
		}

		m.statusMsg = "Document deleted"
		m.statusMsgTime = time.Now()
		return m, clearStatusAfterDelay()

	case launchEditorMsg:
		// Launch the editor using tea.Exec
		return m, openEditorCmd(msg.session)

	case editorFinishedMsg:
		// Editor closed - check for error, validate, and save
		if msg.err != nil {
			// Clean up temp file on error
			os.Remove(msg.session.tempFile)
			return m, func() tea.Msg {
				return errorMsg{err: fmt.Errorf("editor error: %w", msg.err)}
			}
		}
		return m, handleEditorFinished(msg.session)

	}

	return m, nil
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Track key press for double-tap detection
	currentKey := msg.String()
	currentTime := time.Now()

	// Check for double 'y' tap (yy - vim yank)
	if currentKey == "y" {
		if m.lastKeyPress == "y" && currentTime.Sub(m.lastKeyTime) < 500*time.Millisecond {
			// Double tap detected! Reset state and perform copy
			m.lastKeyPress = ""
			m.lastKeyTime = time.Time{}
			return m.handleCopy()
		}
		// First 'y' press - track it
		m.lastKeyPress = currentKey
		m.lastKeyTime = currentTime
		return m, nil
	}

	// Check for 'ya' sequence (yank all - copy entire document)
	if currentKey == "a" {
		if m.lastKeyPress == "y" && currentTime.Sub(m.lastKeyTime) < 500*time.Millisecond {
			// 'ya' sequence detected! Reset state and copy document
			m.lastKeyPress = ""
			m.lastKeyTime = time.Time{}
			return m.handleCopyDocument()
		}
	}

	// Check for shift+y (Y - copy ID/key)
	if currentKey == "Y" {
		m.lastKeyPress = ""
		return m.handleCopyAlternate()
	}

	// Any other key resets the double-tap tracking
	if currentKey != "shift" && currentKey != "ctrl" && currentKey != "alt" {
		m.lastKeyPress = ""
	}

	switch msg.String() {
	// Quit
	case "q", "esc", "ctrl+c":
		return m, tea.Quit

	// Vertical navigation (within column)
	case "j", "down":
		m.moveCursor(1)
		return m, nil

	case "k", "up":
		m.moveCursor(-1)
		return m, nil

	case "g":
		m.jumpToTop()
		return m, nil

	case "G":
		m.jumpToBottom()
		return m, nil

	// Scrolling within column
	case "ctrl+d", "pgdown":
		m.scrollDown()
		return m, nil

	case "ctrl+u", "pgup":
		m.scrollUp()
		return m, nil

	// Horizontal navigation (between columns)
	case "space":
		// Toggle expansion for data nodes
		item := m.getSelectedItem()
		if item != nil && item.isData {
			item.expanded = !item.expanded
			return m, nil
		}
		return m, nil

	case "l", "right", "enter":
		// Get selected item and navigate forward
		item := m.getSelectedItem()
		logDebug("Forward navigation - selected item: %v", item)
		if item != nil {
			if item.isData {
				// Handle tree expansion/toggle
				if msg.String() == "enter" {
					item.expanded = !item.expanded
				} else if !item.expanded {
					item.expanded = true
				}
				return m, nil
			}

			logDebug("Adding column: path='%s', isDoc=%v", item.path, item.isDoc)
			m.addColumn(item.path, item.isDoc)
			m.loading = true
			logDebug("Fetching data for column %d", len(m.columns)-1)
			sortField, sortDir := m.getSortParams(item.path)
			return m, fetchColumnData(m.client, item.path, item.isDoc, len(m.columns)-1, sortField, sortDir)
		}
		return m, nil

	case "h", "left", "backspace":
		// Check if we can collapse a data node
		item := m.getSelectedItem()
		if item != nil && item.isData && item.expanded {
			item.expanded = false
			return m, nil
		}

		// Navigate back
		m.removeLastColumn()
		return m, nil

	case "s":
		// Open sort dialog for collections
		if m.activeColumn >= len(m.columns) {
			return m, nil
		}

		col := m.columns[m.activeColumn]
		if col.isDoc {
			m.statusMsg = "Can only sort collections (not documents)"
			m.statusMsgTime = time.Now()
			return m, clearStatusAfterDelay()
		}

		// Initialize sort dialog with available fields
		m.sortDialog = initSortDialog(col.availableFields)
		m.overlay = OverlaySortDialog
		return m, nil

	case "S":
		// Clear sort for current collection
		if m.activeColumn >= len(m.columns) {
			return m, nil
		}

		col := m.columns[m.activeColumn]
		if col.isDoc {
			return m, nil
		}

		delete(m.sortState, col.path)

		m.statusMsg = "Sort cleared"
		m.statusMsgTime = time.Now()
		m.loading = true

		return m, tea.Batch(
			fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn, "", 0),
			clearStatusAfterDelay(),
		)

	case "e":
		// Edit document
		if m.activeColumn >= len(m.columns) {
			return m, nil
		}

		col := m.columns[m.activeColumn]
		if !col.isDoc {
			m.statusMsg = "Can only edit documents (not collections)"
			m.statusMsgTime = time.Now()
			return m, clearStatusAfterDelay()
		}

		return m, startEditCmd(m.client, col.path, col.docData)

	case "r":
		// Refresh current column
		if m.activeColumn < len(m.columns) {
			col := m.columns[m.activeColumn]
			m.loading = true
			sortField, sortDir := m.getSortParams(col.path)
			return m, fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn, sortField, sortDir)
		}
		return m, nil

	case "d":
		if m.activeColumn >= len(m.columns) {
			return m, nil
		}

		col := m.columns[m.activeColumn]
		var docPath string

		if col.isDoc {
			docPath = col.path
			m.deleteFromDocView = true
		} else {
			item := m.getSelectedItem()
			if item == nil || !item.isDoc {
				m.statusMsg = "Select a document to delete"
				m.statusMsgTime = time.Now()
				return m, clearStatusAfterDelay()
			}
			docPath = item.path
			m.deleteFromDocView = false
		}

		m.deletePath = docPath
		m.overlay = OverlayDeleteConfirm
		return m, nil
	}

	return m, nil
}

// handleOverlay processes keyboard input when an overlay/dialog is active
func (m Model) handleOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.overlay {
	case OverlayPathInput:
		switch msg.String() {
		case "enter":
			path := strings.Trim(m.textInput.Value(), "/")
			isDoc := false
			if path != "" {
				segments := strings.Split(path, "/")
				isDoc = len(segments)%2 == 0
			}
			m.columns = []Column{{
				path:         path,
				isDoc:        isDoc,
				scrollOffset: 0,
			}}
			m.activeColumn = 0
			m.overlay = OverlayNone
			m.textInput.Blur()
			m.textInput.Reset()
			m.loading = true
			sortField, sortDir := m.getSortParams(path)
			return m, fetchColumnData(m.client, path, isDoc, 0, sortField, sortDir)
		case "esc":
			m.overlay = OverlayNone
			m.textInput.Blur()
			m.textInput.Reset()
			return m, nil
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	case OverlaySortDialog:
		switch msg.String() {
		case "tab":
			if m.sortDialog.focusedComponent == 0 {
				m.sortDialog.focusedComponent = 1
				m.sortDialog.textInput.Blur()
			} else {
				m.sortDialog.focusedComponent = 0
				m.sortDialog.textInput.Focus()
			}
			return m, nil
		case "ctrl+d":
			if m.sortDialog.direction == firestore.Asc {
				m.sortDialog.direction = firestore.Desc
			} else {
				m.sortDialog.direction = firestore.Asc
			}
			return m, nil
		case "enter":
			return m.applySortAndClose()
		case "esc":
			m.overlay = OverlayNone
			return m, nil
		}
		if m.sortDialog.focusedComponent == 0 {
			m.sortDialog.textInput, cmd = m.sortDialog.textInput.Update(msg)
		} else {
			cmd = m.sortDialog.Update(msg)
		}
		return m, cmd

	case OverlayDeleteConfirm:
		switch msg.String() {
		case "y", "enter":
			path := m.deletePath
			fromDocView := m.deleteFromDocView
			m.deletePath = ""
			m.deleteFromDocView = false
			m.overlay = OverlayNone
			m.loading = true
			return m, deleteDocument(m.client, path, fromDocView)
		case "n", "esc", "q":
			m.deletePath = ""
			m.deleteFromDocView = false
			m.overlay = OverlayNone
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// applySortAndClose applies the sort from the dialog and closes it
func (m Model) applySortAndClose() (tea.Model, tea.Cmd) {
	// Get selected field (text input takes priority)
	field := m.sortDialog.getSelectedField()
	
	if field == "" {
		m.statusMsg = "No field selected"
		m.statusMsgTime = time.Now()
		return m, clearStatusAfterDelay()
	}

	// Get current column
	if m.activeColumn >= len(m.columns) {
		return m, nil
	}
	col := m.columns[m.activeColumn]

	// Save sort state for this collection path
	m.sortState[col.path] = sortStateEntry{
		Field:     field,
		Direction: m.sortDialog.direction,
	}

	// Close dialog and refresh with sort
	m.overlay = OverlayNone
	m.loading = true

	// Show status message
	dirStr := "Ascending"
	if m.sortDialog.direction == firestore.Desc {
		dirStr = "Descending"
	}
	m.statusMsg = fmt.Sprintf("Sorted by %s (%s)", field, dirStr)
	m.statusMsgTime = time.Now()

	return m, tea.Batch(
		fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn, field, m.sortDialog.direction),
		clearStatusAfterDelay(),
	)
}
