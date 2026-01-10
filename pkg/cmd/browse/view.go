package browse

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the entire TUI
func (m Model) View() string {
	defer func() {
		if r := recover(); r != nil {
			logDebug("PANIC in View: %v", r)
			// Don't re-panic, return error message instead
		}
	}()

	logDebug("View called: width=%d, height=%d, columns=%d", m.width, m.height, len(m.columns))
	if m.width == 0 || m.height == 0 {
		logDebug("View returning early: dimensions not set")
		return "Initializing..."
	}

	// Header
	header := headerStyle.Render(fmt.Sprintf("Firestore Browser - Project: %s", m.projectID))

	// Footer
	footer := footerStyle.Render("j/k: Up/Down  h/l: Back/Forward  g/G: Top/Bottom  r: Refresh  q: Quit")

	// Error message
	errorMsg := ""
	if m.err != nil {
		errorMsg = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Loading indicator
	loadingMsg := ""
	if m.loading {
		loadingMsg = loadingStyle.Render("Loading...")
	}

	// Render columns
	columnsView := m.renderColumns()
	logDebug("View: columnsView length=%d", len(columnsView))

	// Combine all parts
	logDebug("View: about to JoinVertical")
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		columnsView,
		errorMsg,
		loadingMsg,
		footer,
	)
	logDebug("View: JoinVertical successful, contentLen=%d", len(content))

	return content
}

// renderColumns renders all visible columns side by side
func (m Model) renderColumns() string {
	defer func() {
		if r := recover(); r != nil {
			logDebug("PANIC in renderColumns: %v", r)
		}
	}()

	if len(m.columns) == 0 {
		return "No data"
	}

	visibleCols := getVisibleColumns(m.columns, m.activeColumn)
	colWidth := calculateColumnWidth(m.width, len(visibleCols))
	logDebug("renderColumns: visibleCols=%d, colWidth=%d", len(visibleCols), colWidth)

	var renderedCols []string
	for i, col := range visibleCols {
		// Determine if this is the active column
		isActive := (i == m.activeColumn) || (len(m.columns) > 4 && i == 3)

		rendered := m.renderColumn(col, colWidth, isActive)
		logDebug("renderColumns: column %d rendered, length=%d", i, len(rendered))
		renderedCols = append(renderedCols, rendered)
	}

	logDebug("renderColumns: about to JoinHorizontal with %d columns", len(renderedCols))
	result := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)
	logDebug("renderColumns: JoinHorizontal successful, resultLen=%d", len(result))
	return result
}

// renderColumn renders a single column
func (m Model) renderColumn(col Column, width int, isActive bool) string {
	defer func() {
		if r := recover(); r != nil {
			logPanic("renderColumn", r)
			panic(r) // re-panic after logging
		}
	}()

	height := m.height - 6 // Account for header, footer, etc.
	if height < 1 {
		height = 10 // Minimum height
	}
	logDebug("renderColumn: width=%d, height=%d, isActive=%v, sections=%d", width, height, isActive, len(col.sections))

	var content strings.Builder
	itemIndex := 0

	// Render sections
	for _, section := range col.sections {
		if section.hidden {
			continue
		}

		// Section header
		content.WriteString(sectionHeaderStyle.Render(section.title))
		content.WriteString("\n")

		// Section items
		for _, item := range section.items {
			prefix := "  "
			if itemIndex == col.cursor {
				prefix = "> "
			}

			indicator := ""
			if item.isDoc {
				indicator = docIndicatorStyle.Render("[D]")
			} else {
				indicator = colIndicatorStyle.Render("[C]")
			}

			line := fmt.Sprintf("%s%s %s", prefix, item.id, indicator)
			if itemIndex == col.cursor {
				line = selectedItemStyle.Render(line)
			}

			content.WriteString(line)
			content.WriteString("\n")
			itemIndex++
		}
		content.WriteString("\n")
	}

	// If this is a document column, render document content
	if col.isDoc && col.docContent != "" {
		content.WriteString(sectionHeaderStyle.Render("Document Content"))
		content.WriteString("\n")
		content.WriteString(col.docContent)
	}

	// Apply column style
	columnContent := content.String()
	if len(columnContent) == 0 {
		columnContent = "No items"
	}

	// Truncate to fit height
	lines := strings.Split(columnContent, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	columnContent = strings.Join(lines, "\n")

	// Apply border style
	logDebug("About to render with lipgloss: width=%d, height=%d, contentLen=%d", width, height, len(columnContent))

	var result string
	defer func() {
		if r := recover(); r != nil {
			logDebug("PANIC in lipgloss render: %v", r)
			logDebug("width=%d, height=%d, isActive=%v, contentLen=%d", width, height, isActive, len(columnContent))
			// Return a simple string instead of panicking
			result = fmt.Sprintf("Error rendering column (w=%d,h=%d)", width, height)
		}
	}()

	if isActive {
		result = activeColumnStyle.Width(width).Height(height).Render(columnContent)
	} else {
		result = inactiveColumnStyle.Width(width).Height(height).Render(columnContent)
	}

	logDebug("Lipgloss render successful")
	return result
}
