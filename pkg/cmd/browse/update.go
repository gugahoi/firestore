package browse

import (
	"context"
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
			if msg.String() == "ctrl+o" {
				entry, ok := m.jumplist.Back()
				if ok {
					m.columns = []Column{{path: entry.Path, isDoc: entry.IsDoc, scrollOffset: 0}}
					m.activeColumn = 0
					m.loading = true
					sortField, sortDir := m.getSortParams(entry.Path)
					return m, fetchColumnData(m.client, entry.Path, entry.IsDoc, 0, sortField, sortDir, withLimit(m.pageLimit))
				}
				return m, nil
			}
			if msg.String() == "ctrl+i" {
				entry, ok := m.jumplist.Forward()
				if ok {
					m.columns = []Column{{path: entry.Path, isDoc: entry.IsDoc, scrollOffset: 0}}
					m.activeColumn = 0
					m.loading = true
					sortField, sortDir := m.getSortParams(entry.Path)
					return m, fetchColumnData(m.client, entry.Path, entry.IsDoc, 0, sortField, sortDir, withLimit(m.pageLimit))
				}
				return m, nil
			}
			// Enter filter mode
			if msg.String() == "/" {
				m.overlay = OverlayFilter
				m.filterInput.Reset()
				m.filterInput.Focus()
				return m, textinput.Blink
			}
			// Enter command mode
			if msg.String() == ":" {
				m.mode = ModeCommand
				m.commandInput.Reset()
				m.commandInput.Focus()
				m.commandResult = ""
				return m, textinput.Blink
			}
			return m.handleKeyPress(msg)

		case ModeVisual:
			return m.handleVisualMode(msg)

		case ModeCommand:
			return m.handleCommandMode(msg)
		}

	case tea.WindowSizeMsg:

		logDebug("WindowSizeMsg received: width=%d, height=%d", msg.Width, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport sizes for document columns
		for i := range m.columns {
			if m.columns[i].isDoc && m.columns[i].docContent != "" {
				colWidth := calculateColumnWidth(m.width-4, len(m.columns))
				colHeight := m.itemViewHeight()
				vpWidth := colWidth - 2
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
		logDebug("fetchedColumnMsg received: columnIndex=%d, sections=%d, docContentLen=%d, append=%v",
			msg.columnIndex, len(msg.sections), len(msg.docContent), msg.appendItems)
		if msg.columnIndex < len(m.columns) {
			if msg.appendItems {
				// Append new items to existing sections
				col := &m.columns[msg.columnIndex]
				for i, newSection := range msg.sections {
					if i < len(col.sections) && col.sections[i].title == newSection.title {
						// Remove old "Load more..." sentinel
						existingItems := col.sections[i].items
						if len(existingItems) > 0 && existingItems[len(existingItems)-1].path == "__load_more__" {
							existingItems = existingItems[:len(existingItems)-1]
						}
						// Append new items
						col.sections[i].items = append(existingItems, newSection.items...)
						col.sections[i].hidden = len(col.sections[i].items) == 0
					}
				}
				col.hasMore = msg.hasMore
				col.docCount += msg.docCount
			} else {
				m.columns[msg.columnIndex].sections = msg.sections
				m.columns[msg.columnIndex].docContent = msg.docContent
				m.columns[msg.columnIndex].docData = msg.docData
				m.columns[msg.columnIndex].docMetadata = msg.docMetadata
				if len(msg.availableFields) > 0 {
					m.columns[msg.columnIndex].availableFields = msg.availableFields
				}
				m.columns[msg.columnIndex].scrollOffset = 0
				m.columns[msg.columnIndex].hasMore = msg.hasMore
				m.columns[msg.columnIndex].docCount = msg.docCount
			}

			// Initialize viewport if this is a document column
			// Only create viewport if we have valid dimensions
			if m.columns[msg.columnIndex].isDoc && msg.docContent != "" && m.width > 0 && m.height > 0 {
				colWidth := calculateColumnWidth(m.width-4, len(m.columns))
				colHeight := m.itemViewHeight()
				vpWidth := colWidth - 2
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
				fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn, sortField, sortDir, withLimit(m.pageLimit)),
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

	case previewDebounceMsg:
		if !m.previewEnabled || msg.path != m.previewPending {
			return m, nil
		}
		if msg.path == m.previewPath {
			return m, nil
		}
		return m, fetchPreviewData(m.client, msg.path)

	case previewFetchedMsg:
		if !m.previewEnabled {
			return m, nil
		}
		m.previewPath = msg.path
		m.previewData = msg.data
		if msg.data != nil {
			m.previewNodes = buildTreeNodes(msg.data)
		} else {
			m.previewNodes = nil
		}
		return m, nil

	case documentCreatedMsg:
		m.loading = false
		m.statusMsg = fmt.Sprintf("Document created: %s", msg.docID)
		m.statusMsgTime = time.Now()

		if msg.colIndex < len(m.columns) {
			col := m.columns[msg.colIndex]
			sortField, sortDir := m.getSortParams(col.path)
			m.loading = true
			return m, tea.Batch(
				fetchColumnData(m.client, col.path, col.isDoc, msg.colIndex, sortField, sortDir, withLimit(m.pageLimit)),
				clearStatusAfterDelay(),
			)
		}
		return m, clearStatusAfterDelay()

	case bulkDeletedMsg:
		m.loading = false
		m.statusMsg = fmt.Sprintf("%d documents deleted", msg.count)
		m.statusMsgTime = time.Now()

		if msg.colIndex < len(m.columns) {
			col := m.columns[msg.colIndex]
			sortField, sortDir := m.getSortParams(col.path)
			m.loading = true
			return m, tea.Batch(
				fetchColumnData(m.client, col.path, col.isDoc, msg.colIndex, sortField, sortDir, withLimit(m.pageLimit)),
				clearStatusAfterDelay(),
			)
		}
		return m, clearStatusAfterDelay()

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

	// Handle pending z-key for fold controls
	if m.pendingZKey {
		m.pendingZKey = false
		switch currentKey {
		case "M":
			m.foldAll(false)
		case "R":
			m.foldAll(true)
		case "1":
			m.foldToDepth(1)
		case "2":
			m.foldToDepth(2)
		case "3":
			m.foldToDepth(3)
		}
		return m, nil
	}

	// Handle pending mark operations
	if m.pendingMark == "m" {
		m.pendingMark = ""
		if len(currentKey) == 1 && currentKey[0] >= 'a' && currentKey[0] <= 'z' {
			letter := rune(currentKey[0])
			if m.activeColumn < len(m.columns) {
				col := m.columns[m.activeColumn]
				m.marks[letter] = markEntry{path: col.path, isDoc: col.isDoc}
				m.statusMsg = fmt.Sprintf("Mark '%c' set: %s", letter, col.path)
				m.statusMsgTime = time.Now()
				return m, clearStatusAfterDelay()
			}
		}
		return m, nil
	}
	if m.pendingMark == "'" {
		m.pendingMark = ""
		if len(currentKey) == 1 && currentKey[0] >= 'a' && currentKey[0] <= 'z' {
			letter := rune(currentKey[0])
			mark, ok := m.marks[letter]
			if !ok {
				m.statusMsg = fmt.Sprintf("Mark '%c' not set", letter)
				m.statusMsgTime = time.Now()
				return m, clearStatusAfterDelay()
			}
			// Push current location to jumplist
			if m.activeColumn < len(m.columns) {
				cur := m.columns[m.activeColumn]
				m.jumplist.Push(cur.path, cur.isDoc)
			}
			m.columns = []Column{{path: mark.path, isDoc: mark.isDoc, scrollOffset: 0}}
			m.activeColumn = 0
			m.loading = true
			sortField, sortDir := m.getSortParams(mark.path)
			return m, fetchColumnData(m.client, mark.path, mark.isDoc, 0, sortField, sortDir, withLimit(m.pageLimit))
		}
		return m, nil
	}

	// Any other key resets the double-tap tracking
	if currentKey != "shift" && currentKey != "ctrl" && currentKey != "alt" {
		m.lastKeyPress = ""
	}

	switch msg.String() {
	// Quit
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.statusMsg = "Press q to quit"
		m.statusMsgTime = time.Now()
		return m, clearStatusAfterDelay()

	// Vertical navigation (within column)
	case "j", "down":
		m.moveCursor(1)
		return m, m.schedulePreviewFetch()

	case "k", "up":
		m.moveCursor(-1)
		return m, m.schedulePreviewFetch()

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
			// Handle "Load more..." sentinel
			if item.path == "__load_more__" {
				if m.activeColumn < len(m.columns) {
					col := m.columns[m.activeColumn]
					m.loading = true
					sortField, sortDir := m.getSortParams(col.path)
					return m, fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn, sortField, sortDir,
						withLimit(m.pageLimit), withOffset(col.docCount), withAppend())
				}
				return m, nil
			}
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
			// Push current path to jumplist before navigating forward
			if m.activeColumn < len(m.columns) {
				cur := m.columns[m.activeColumn]
				m.jumplist.Push(cur.path, cur.isDoc)
			}
			m.addColumn(item.path, item.isDoc)
			m.loading = true
			logDebug("Fetching data for column %d", len(m.columns)-1)
			sortField, sortDir := m.getSortParams(item.path)
			return m, fetchColumnData(m.client, item.path, item.isDoc, len(m.columns)-1, sortField, sortDir, withLimit(m.pageLimit))
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
			fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn, "", 0, withLimit(m.pageLimit)),
			clearStatusAfterDelay(),
		)

	case "z":
		m.pendingZKey = true
		return m, nil

	case "p":
		m.previewEnabled = !m.previewEnabled
		if !m.previewEnabled {
			m.previewPath = ""
			m.previewData = nil
			m.previewNodes = nil
			m.previewPending = ""
		} else {
			return m, m.schedulePreviewFetch()
		}
		return m, nil

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
			return m, fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn, sortField, sortDir, withLimit(m.pageLimit))
		}
		return m, nil

	case "m":
		m.pendingMark = "m"
		return m, nil

	case "'":
		m.pendingMark = "'"
		return m, nil

	case "v":
		// Toggle selection and enter visual mode
		if m.activeColumn < len(m.columns) && !m.columns[m.activeColumn].isDoc {
			col := m.columns[m.activeColumn]
			cursor := col.cursor
			m.selection.Toggle(cursor)
			m.mode = ModeVisual
		}
		return m, nil

	case "V":
		// Range-select mode
		if m.activeColumn < len(m.columns) && !m.columns[m.activeColumn].isDoc {
			col := m.columns[m.activeColumn]
			m.selection.SetAnchor(col.cursor)
			m.mode = ModeVisual
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
			return m, fetchColumnData(m.client, path, isDoc, 0, sortField, sortDir, withLimit(m.pageLimit))
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
			m.overlay = OverlayNone

			// Bulk delete
			if len(m.bulkDeletePaths) > 0 {
				paths := m.bulkDeletePaths
				m.bulkDeletePaths = nil
				m.selection.Clear()
				m.mode = ModeNormal
				m.loading = true
				return m, bulkDeleteDocuments(m.client, paths, m.activeColumn)
			}

			// Single delete
			path := m.deletePath
			fromDocView := m.deleteFromDocView
			m.deletePath = ""
			m.deleteFromDocView = false
			m.loading = true
			return m, deleteDocument(m.client, path, fromDocView)
		case "n", "esc", "q":
			m.deletePath = ""
			m.deleteFromDocView = false
			m.bulkDeletePaths = nil
			m.overlay = OverlayNone
			return m, nil
		}
		return m, nil

	case OverlayInfo:
		switch msg.String() {
		case "esc", "q", "enter":
			m.overlay = OverlayNone
			m.infoContent = ""
			return m, nil
		}
		return m, nil

	case OverlayFilter:
		switch msg.String() {
		case "esc":
			m.overlay = OverlayNone
			m.filterInput.Blur()
			m.filterInput.Reset()
			m.filterActive = false
			m.filterPattern = ""
			if m.activeColumn < len(m.columns) {
				m.columns[m.activeColumn].cursor = 0
				m.columns[m.activeColumn].scrollOffset = 0
			}
			return m, nil
		case "enter":
			m.overlay = OverlayNone
			m.filterInput.Blur()
			pattern := m.filterInput.Value()
			if pattern != "" {
				m.filterActive = true
				m.filterPattern = pattern
			} else {
				m.filterActive = false
				m.filterPattern = ""
			}
			return m, nil
		}
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.filterPattern = m.filterInput.Value()
		if m.activeColumn < len(m.columns) {
			m.columns[m.activeColumn].cursor = 0
			m.columns[m.activeColumn].scrollOffset = 0
		}
		return m, cmd
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
		fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn, field, m.sortDialog.direction, withLimit(m.pageLimit)),
		clearStatusAfterDelay(),
	)
}

// handleCommandMode processes input in command mode
func (m Model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeNormal
		m.commandInput.Blur()
		m.commandInput.Reset()
		m.commandResult = ""
		return m, nil

	case "enter":
		input := m.commandInput.Value()
		m.mode = ModeNormal
		m.commandInput.Blur()
		m.commandInput.Reset()

		name, args := ParseCommand(input)
		if name == "" {
			return m, nil
		}

		cmd, ok := m.commandRegistry.Get(name)
		if !ok {
			m.statusMsg = fmt.Sprintf("Unknown command: %s", name)
			m.statusMsgTime = time.Now()
			return m, clearStatusAfterDelay()
		}

		statusText, err := cmd.Handler(&m, args)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error: %s", err)
			m.statusMsgTime = time.Now()
			return m, clearStatusAfterDelay()
		}

		if statusText != "" {
			if strings.Contains(statusText, "\n") {
				m.overlay = OverlayInfo
				m.infoContent = statusText
			} else {
				m.statusMsg = statusText
				m.statusMsgTime = time.Now()
			}
		}

		// Check if the handler set a pending editor launch
		if m.pendingEditor != nil {
			cmd := m.pendingEditor
			m.pendingEditor = nil
			return m, cmd
		}

		// Check if the handler set a pending fetch
		if m.pendingFetch != nil {
			pf := m.pendingFetch
			m.pendingFetch = nil
			return m, tea.Batch(
				fetchColumnData(m.client, pf.path, pf.isDoc, pf.colIndex, pf.sortField, pf.sortDir, pf.opts...),
				clearStatusAfterDelay(),
			)
		}

		return m, clearStatusAfterDelay()

	case "tab":
		// Tab completion
		input := m.commandInput.Value()
		name, _ := ParseCommand(input)
		matches := m.commandRegistry.Complete(name)
		if len(matches) == 1 {
			m.commandInput.SetValue(matches[0] + " ")
			m.commandInput.CursorEnd()
		} else if len(matches) > 1 {
			// Find common prefix
			prefix := longestCommonPrefix(matches)
			if len(prefix) > len(name) {
				m.commandInput.SetValue(prefix)
				m.commandInput.CursorEnd()
			}
			m.commandResult = strings.Join(matches, "  ")
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.commandInput, cmd = m.commandInput.Update(msg)
	m.commandResult = ""
	return m, cmd
}

// handleVisualMode processes keyboard input in Visual mode
func (m Model) handleVisualMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeNormal
		m.selection.Clear()
		return m, nil

	case "v":
		// Toggle selection on current item
		if m.activeColumn < len(m.columns) {
			m.selection.Toggle(m.columns[m.activeColumn].cursor)
			if m.selection.Count() == 0 {
				m.mode = ModeNormal
			}
		}
		return m, nil

	case "j", "down":
		m.moveCursor(1)
		if m.selection.HasAnchor() && m.activeColumn < len(m.columns) {
			m.selection.ExtendTo(m.columns[m.activeColumn].cursor)
		}
		return m, nil

	case "k", "up":
		m.moveCursor(-1)
		if m.selection.HasAnchor() && m.activeColumn < len(m.columns) {
			m.selection.ExtendTo(m.columns[m.activeColumn].cursor)
		}
		return m, nil

	case "g":
		m.jumpToTop()
		if m.selection.HasAnchor() && m.activeColumn < len(m.columns) {
			m.selection.ExtendTo(m.columns[m.activeColumn].cursor)
		}
		return m, nil

	case "G":
		m.jumpToBottom()
		if m.selection.HasAnchor() && m.activeColumn < len(m.columns) {
			m.selection.ExtendTo(m.columns[m.activeColumn].cursor)
		}
		return m, nil

	case "d":
		// Bulk delete selected items
		if m.selection.Count() == 0 {
			return m, nil
		}

		if m.activeColumn >= len(m.columns) {
			return m, nil
		}

		// Collect paths of selected items
		col := m.columns[m.activeColumn]
		sections := m.getEffectiveSections(col)
		var paths []string
		itemIndex := 0
		for _, section := range sections {
			if section.hidden {
				continue
			}
			for _, item := range section.items {
				if m.selection.IsSelected(itemIndex) && item.isDoc && item.path != "__load_more__" {
					paths = append(paths, item.path)
				}
				itemIndex++
			}
		}

		if len(paths) == 0 {
			m.statusMsg = "No documents selected"
			m.statusMsgTime = time.Now()
			return m, clearStatusAfterDelay()
		}

		m.bulkDeletePaths = paths
		m.overlay = OverlayDeleteConfirm
		return m, nil
	}

	return m, nil
}

func (m *Model) schedulePreviewFetch() tea.Cmd {
	if !m.previewEnabled {
		return nil
	}
	item := m.getSelectedItem()
	if item == nil || !item.isDoc || item.path == "__load_more__" {
		m.previewPending = ""
		return nil
	}
	m.previewPending = item.path
	path := item.path
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
		return previewDebounceMsg{path: path}
	})
}

func fetchPreviewData(client *firestore.Client, path string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		docRef := client.Doc(strings.TrimPrefix(path, "/"))
		snap, err := docRef.Get(ctx)
		if err != nil {
			return previewFetchedMsg{path: path, data: nil}
		}
		if !snap.Exists() {
			return previewFetchedMsg{path: path, data: nil}
		}
		return previewFetchedMsg{path: path, data: snap.Data()}
	}
}

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for len(prefix) > 0 && !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}
