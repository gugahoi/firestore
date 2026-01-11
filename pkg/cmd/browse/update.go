package browse

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		logDebug("KeyMsg received: %s", msg.String())

		// Handle input mode
		if m.mode == ModePathInput {
			switch msg.String() {
			case "enter":
				// Browse to new path
				path := strings.Trim(m.textInput.Value(), "/")

				// Determine if it is a doc or collection
				// A simple heuristic: even segments = doc, odd = collection
				// Root (empty string) is handled as collection list
				isDoc := false
				if path != "" {
					segments := strings.Split(path, "/")
					isDoc = len(segments)%2 == 0
				}

				// Reset columns
				m.columns = []Column{{
					path:         path,
					isDoc:        isDoc,
					scrollOffset: 0,
				}}
				m.activeColumn = 0
				m.mode = ModeNormal
				m.textInput.Blur()
				m.textInput.Reset()
				m.loading = true
				return m, fetchColumnData(m.client, path, isDoc, 0)
			case "esc":
				m.mode = ModeNormal
				m.textInput.Blur()
				m.textInput.Reset()
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// Handle normal mode global keys
		if msg.String() == "ctrl+g" {
			m.mode = ModePathInput
			m.textInput.Focus()
			return m, textinput.Blink
		}

		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:

		logDebug("WindowSizeMsg received: width=%d, height=%d", msg.Width, msg.Height)
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport sizes for document columns
		for i := range m.columns {
			if m.columns[i].isDoc && m.columns[i].docContent != "" {
				colWidth := calculateColumnWidth(m.width, len(m.columns))
				colHeight := m.height - 6 // Account for header and footer
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
			m.columns[msg.columnIndex].scrollOffset = 0 // Reset scroll to top when new data arrives

			// Initialize viewport if this is a document column
			// Only create viewport if we have valid dimensions
			if m.columns[msg.columnIndex].isDoc && msg.docContent != "" && m.width > 0 && m.height > 0 {
				colWidth := calculateColumnWidth(m.width, len(m.columns))
				colHeight := m.height - 6
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
			return m, fetchColumnData(m.client, item.path, item.isDoc, len(m.columns)-1)
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

	case "r":
		// Refresh current column
		if m.activeColumn < len(m.columns) {
			col := m.columns[m.activeColumn]
			m.loading = true
			return m, fetchColumnData(m.client, col.path, col.isDoc, m.activeColumn)
		}
		return m, nil
	}

	return m, nil
}
