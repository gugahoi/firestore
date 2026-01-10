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

	col.cursor += delta
	if col.cursor < 0 {
		col.cursor = 0
	}
	if col.cursor >= len(items) {
		col.cursor = len(items) - 1
	}
}

// jumpToTop moves cursor to the first item
func (m *Model) jumpToTop() {
	if m.activeColumn >= len(m.columns) {
		return
	}
	m.columns[m.activeColumn].cursor = 0
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
