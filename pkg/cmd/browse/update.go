package browse

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		logDebug("KeyMsg received: %s", msg.String())
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
	}

	return m, nil
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	// Horizontal navigation (between columns)
	case "l", "right", "enter":
		// Get selected item and navigate forward
		item := m.getSelectedItem()
		logDebug("Forward navigation - selected item: %v", item)
		if item != nil {
			logDebug("Adding column: path='%s', isDoc=%v", item.path, item.isDoc)
			m.addColumn(item.path, item.isDoc)
			m.loading = true
			logDebug("Fetching data for column %d", len(m.columns)-1)
			return m, fetchColumnData(m.client, item.path, item.isDoc, len(m.columns)-1)
		}
		return m, nil

	case "h", "left", "backspace":
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
