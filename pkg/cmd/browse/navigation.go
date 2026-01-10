package browse

import (
	"github.com/charmbracelet/bubbles/viewport"
)

// getSelectedItem returns the currently selected item from the active column
func (m *Model) getSelectedItem() *ListItem {
	if m.activeColumn >= len(m.columns) {
		return nil
	}

	col := &m.columns[m.activeColumn]
	itemIndex := 0

	for _, section := range col.sections {
		if section.hidden {
			continue
		}
		for _, item := range section.items {
			if itemIndex == col.cursor {
				return &item
			}
			itemIndex++
		}
	}

	return nil
}

// getAllItems returns all visible items from a column (flattened from sections)
func getAllItems(col Column) []ListItem {
	var items []ListItem
	for _, section := range col.sections {
		if !section.hidden {
			items = append(items, section.items...)
		}
	}
	return items
}

// moveCursor moves the cursor in the active column
func (m *Model) moveCursor(delta int) {
	if m.activeColumn >= len(m.columns) {
		return
	}

	col := &m.columns[m.activeColumn]
	items := getAllItems(*col)
	if len(items) == 0 {
		return
	}

	oldCursor := col.cursor
	col.cursor += delta
	if col.cursor < 0 {
		col.cursor = 0
	}
	if col.cursor >= len(items) {
		col.cursor = len(items) - 1
	}

	// Auto-scroll to keep cursor visible
	if col.cursor != oldCursor {
		m.autoScroll()
	}
}

// autoScroll adjusts scroll offset to keep cursor visible
func (m *Model) autoScroll() {
	if m.activeColumn >= len(m.columns) {
		return
	}

	col := &m.columns[m.activeColumn]
	// Match height calculation in view.go: m.height - 7 (header/path/footer/margin) - 2 (borders)
	viewHeight := m.height - 9
	if viewHeight < 1 {
		viewHeight = 10
	}

	// Calculate cursor position in the rendered content
	// We need to account for section headers
	cursorLine := 0
	itemIndex := 0
	for _, section := range col.sections {
		if section.hidden {
			continue
		}
		// Section header takes 1 line
		cursorLine++

		for range section.items {
			if itemIndex == col.cursor {
				// Found cursor position
				break
			}
			cursorLine++ // Each item takes 1 line
			itemIndex++
		}
		if itemIndex == col.cursor {
			break
		}
		cursorLine++ // Empty line after section
	}

	// Adjust scroll to keep cursor visible
	// Keep cursor in middle third of screen when possible
	topMargin := viewHeight / 3
	bottomMargin := viewHeight - (viewHeight / 3)

	visibleTop := col.scrollOffset
	visibleBottom := col.scrollOffset + viewHeight

	if cursorLine < visibleTop+topMargin {
		// Cursor too close to top, scroll up
		col.scrollOffset = cursorLine - topMargin
		if col.scrollOffset < 0 {
			col.scrollOffset = 0
		}
	} else if cursorLine >= visibleBottom-(viewHeight-bottomMargin) {
		// Cursor too close to bottom, scroll down
		col.scrollOffset = cursorLine - bottomMargin
	}
}

// jumpToTop moves cursor to the first item
func (m *Model) jumpToTop() {
	if m.activeColumn >= len(m.columns) {
		return
	}
	m.columns[m.activeColumn].cursor = 0
	m.columns[m.activeColumn].scrollOffset = 0 // Also scroll to top
}

// jumpToBottom moves cursor to the last item
func (m *Model) jumpToBottom() {
	if m.activeColumn >= len(m.columns) {
		return
	}

	col := &m.columns[m.activeColumn]
	items := getAllItems(*col)
	if len(items) > 0 {
		col.cursor = len(items) - 1
		m.autoScroll() // Scroll to show bottom item
	}
}

// scrollDown scrolls the active column down by half a page
func (m *Model) scrollDown() {
	if m.activeColumn >= len(m.columns) {
		return
	}

	col := &m.columns[m.activeColumn]
	viewHeight := m.height - 9
	if viewHeight < 1 {
		viewHeight = 10
	}

	pageSize := viewHeight / 2 // Half page scroll
	if pageSize < 1 {
		pageSize = 1
	}

	col.scrollOffset += pageSize
	// Upper bound will be checked in renderColumn, but we can't check it here
	// because we don't know the full content length until render time.
}

// scrollUp scrolls the active column up by half a page
func (m *Model) scrollUp() {
	if m.activeColumn >= len(m.columns) {
		return
	}

	col := &m.columns[m.activeColumn]
	viewHeight := m.height - 9
	if viewHeight < 1 {
		viewHeight = 10
	}

	pageSize := viewHeight / 2 // Half page scroll
	if pageSize < 1 {
		pageSize = 1
	}

	col.scrollOffset -= pageSize
	if col.scrollOffset < 0 {
		col.scrollOffset = 0
	}
}

// addColumn adds a new column to the right
func (m *Model) addColumn(path string, isDoc bool) {
	logDebug("addColumn called: path='%s', isDoc=%v, currentColumns=%d", path, isDoc, len(m.columns))

	// Remove all columns to the right of active column
	m.columns = m.columns[:m.activeColumn+1]

	// Create new column
	newCol := Column{
		path:         path,
		isDoc:        isDoc,
		sections:     []Section{},
		docContent:   "",
		viewport:     viewport.Model{}, // Use zero value, will be initialized when dimensions are known
		cursor:       0,
		scrollOffset: 0,
	}

	m.columns = append(m.columns, newCol)
	m.activeColumn = len(m.columns) - 1
	logDebug("addColumn completed: totalColumns=%d, activeColumn=%d", len(m.columns), m.activeColumn)
}

// removeLastColumn removes the rightmost column
func (m *Model) removeLastColumn() {
	if len(m.columns) <= 1 {
		return
	}

	m.columns = m.columns[:len(m.columns)-1]
	m.activeColumn = len(m.columns) - 1
}

// calculateColumnWidth calculates the width for each column
func calculateColumnWidth(termWidth int, numColumns int) int {
	// Show max 4 columns at a time
	visibleCols := numColumns
	if visibleCols > 4 {
		visibleCols = 4
	}

	if visibleCols == 0 || termWidth == 0 {
		return termWidth
	}

	// Account for borders and padding
	width := (termWidth - (visibleCols * 4)) / visibleCols
	if width < 10 {
		width = 10 // Minimum width
	}
	return width
}

// getVisibleColumns returns the slice of columns to display
func getVisibleColumns(columns []Column, activeColumn int) []Column {
	if len(columns) <= 4 {
		return columns
	}

	// Show the last 4 columns
	start := len(columns) - 4
	if start < 0 {
		start = 0
	}

	return columns[start:]
}
