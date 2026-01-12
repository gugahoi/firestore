package browse

import (
	"github.com/charmbracelet/bubbles/viewport"
)

// Helper to get total items in a tree node (including children if expanded)
func getTreeNodeCount(node ListItem) int {
	count := 1 // Count self
	if node.expanded {
		for _, child := range node.children {
			count += getTreeNodeCount(child)
		}
	}
	return count
}

// Helper to find item at absolute index in a tree
func getTreeItemAt(nodes []ListItem, targetIndex int) *ListItem {
	currentIndex := 0

	var find func(nodes []ListItem) *ListItem
	find = func(nodes []ListItem) *ListItem {
		for i := range nodes {
			if currentIndex == targetIndex {
				return &nodes[i]
			}
			currentIndex++

			if nodes[i].expanded {
				found := find(nodes[i].children)
				if found != nil {
					return found
				}
			}
		}
		return nil
	}

	return find(nodes)
}

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

		if section.title == "Data" {
			// Tree view logic
			// Calculate how many visible items are in this tree section
			sectionCount := 0
			for _, item := range section.items {
				sectionCount += getTreeNodeCount(item)
			}

			// Check if cursor is within this section
			if col.cursor < itemIndex+sectionCount {
				return getTreeItemAt(section.items, col.cursor-itemIndex)
			}

			// Not in this section, advance index
			itemIndex += sectionCount
		} else {
			// Linear list logic
			for _, item := range section.items {
				if itemIndex == col.cursor {
					return &item
				}
				itemIndex++
			}
		}
	}

	return nil
}

// getAllItems returns all visible items from a column (flattened from sections)
// Updated to handle tree expansion
func getAllItemsCount(col Column) int {
	count := 0
	for _, section := range col.sections {
		if section.hidden {
			continue
		}

		if section.title == "Data" {
			for _, item := range section.items {
				count += getTreeNodeCount(item)
			}
		} else {
			count += len(section.items)
		}
	}
	return count
}

// moveCursor moves the cursor in the active column
func (m *Model) moveCursor(delta int) {
	if m.activeColumn >= len(m.columns) {
		return
	}

	col := &m.columns[m.activeColumn]
	totalItems := getAllItemsCount(*col)
	if totalItems == 0 {
		return
	}

	oldCursor := col.cursor
	col.cursor += delta
	if col.cursor < 0 {
		col.cursor = 0
	}
	if col.cursor >= totalItems {
		col.cursor = totalItems - 1
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
	foundCursor := false

	for _, section := range col.sections {
		if section.hidden {
			continue
		}
		// Section header takes 1 line
		cursorLine++

		if section.title == "Data" {
			// Tree view logic - we need to traverse the tree to find where the cursor lands
			var traverse func(nodes []ListItem)
			traverse = func(nodes []ListItem) {
				for _, node := range nodes {
					if foundCursor {
						return
					}

					// Increment line count for this item BEFORE checking cursor
					cursorLine++

					if itemIndex == col.cursor {
						foundCursor = true
						return
					}

					itemIndex++

					if node.expanded {
						traverse(node.children)
					}
				}
			}
			traverse(section.items)
			if foundCursor {
				break
			}
		} else {
			// Linear list logic
			for range section.items {
				// Increment line count for this item BEFORE checking cursor
				cursorLine++

				if itemIndex == col.cursor {
					// Found cursor position
					foundCursor = true
					break
				}

				itemIndex++
			}
			if foundCursor {
				break
			}
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
	count := getAllItemsCount(*col)
	if count > 0 {
		col.cursor = count - 1
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
