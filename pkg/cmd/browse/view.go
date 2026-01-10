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
	footer := footerStyle.Render("j/k: Up/Down  h/l: Back/Forward  g/G: Top/Bottom  Ctrl+d/u: Scroll  r: Refresh  q: Quit")

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

	// Calculate available height
	// m.height includes header (1), footer (1), and some margin (4) = 6 total
	// We also need to account for the border (top + bottom = 2)
	height := m.height - 6 - 2 // Account for header, footer, border
	if height < 1 {
		height = 10 // Minimum height
	}

	// Calculate inner width for wrapping
	// Width - Border(2) - Padding(2) = Width - 4
	innerWidth := width - 4
	if innerWidth < 1 {
		innerWidth = 1
	}

	logDebug("renderColumn: width=%d, innerWidth=%d, height=%d, isActive=%v, sections=%d", width, innerWidth, height, isActive, len(col.sections))

	var content strings.Builder
	itemIndex := 0
	linesUsed := 0

	// Helper style for wrapping document content
	wrapper := lipgloss.NewStyle().Width(innerWidth)

	// Render sections
	for _, section := range col.sections {
		if section.hidden {
			continue
		}

		// Section header
		// Truncate title if needed to ensure 1 line
		title := section.title
		if len(title) > innerWidth {
			title = title[:innerWidth-3] + "..."
		}
		headerStr := sectionHeaderStyle.Render(title)
		content.WriteString(headerStr)
		content.WriteString("\n")
		linesUsed++

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

			// Calculate space available for ID
			// prefix(2) + space(1) + indicator(3) = 6 chars roughly (ignoring ansi)
			// But indicator has ANSI, so we assume visual length 3 for [D]/[C]
			extraChars := 6
			availWidth := innerWidth - extraChars
			if availWidth < 5 {
				availWidth = 5
			}

			displayID := item.id
			if len(displayID) > availWidth {
				displayID = displayID[:availWidth-3] + "..."
			}

			line := fmt.Sprintf("%s%s %s", prefix, displayID, indicator)
			if itemIndex == col.cursor {
				line = selectedItemStyle.Render(line)
			}

			content.WriteString(line)
			content.WriteString("\n")
			itemIndex++
			linesUsed++
		}
		content.WriteString("\n")
		linesUsed++
	}

	// If this is a document column, render document content
	if col.isDoc && col.docContent != "" {
		// Calculate remaining space for document content
		remainingLines := height - linesUsed - 2 // Leave 2 lines margin
		logDebug("DOCUMENT CONTENT: height=%d, linesUsed=%d, remainingLines=%d", height, linesUsed, remainingLines)

		// Always render document content if we are in a doc column
		if remainingLines > 2 {
			headerStr := sectionHeaderStyle.Render("Document Content")
			wrappedHeader := wrapper.Render(headerStr)
			content.WriteString(wrappedHeader)
			content.WriteString("\n")
			linesUsed += strings.Count(wrappedHeader, "\n") + 1
			remainingLines -= strings.Count(wrappedHeader, "\n") + 1

			// Wrap the document content to the inner width
			wrappedDoc := wrapper.Render(col.docContent)
			docLines := strings.Split(wrappedDoc, "\n")
			logDebug("DOCUMENT CONTENT: original docLines=%d, remainingLines=%d", len(docLines), remainingLines)

			// We don't truncate anymore because scrolling will handle visibility
			content.WriteString(strings.Join(docLines, "\n"))
			linesUsed += len(docLines)
			logDebug("DOCUMENT CONTENT: after adding doc, linesUsed=%d", linesUsed)
		} else {
			logDebug("DOCUMENT CONTENT: SKIPPED - not enough space (remainingLines=%d)", remainingLines)
		}
	}

	// Apply column style
	columnContent := content.String()
	if len(columnContent) == 0 {
		columnContent = "No items"
	}

	logDebug("Before scrolling: linesUsed=%d, contentLines=%d, height=%d", linesUsed, strings.Count(columnContent, "\n")+1, height)

	// Apply scrolling - show only the visible portion
	lines := strings.Split(columnContent, "\n")
	totalLines := len(lines)

	// Calculate bounds
	maxScroll := totalLines - height
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Use scroll offset (bounds are enforced in scroll functions)
	scrollOffset := col.scrollOffset
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}

	// Extract visible lines based on scroll position
	startLine := scrollOffset
	endLine := scrollOffset + height
	if endLine > totalLines {
		endLine = totalLines
	}

	visibleLines := lines[startLine:endLine]

	// Final safety check - ensure we never exceed height
	if len(visibleLines) > height {
		visibleLines = visibleLines[:height]
	}

	// Add scroll indicators if needed
	if scrollOffset > 0 {
		// Can scroll up - add indicator at top
		if len(visibleLines) > 0 {
			visibleLines[0] = "▲ " + visibleLines[0]
		}
	}
	if scrollOffset < maxScroll {
		// Can scroll down - add indicator at bottom
		if len(visibleLines) > 0 {
			visibleLines[len(visibleLines)-1] = "▼ " + visibleLines[len(visibleLines)-1]
		}
	}

	columnContent = strings.Join(visibleLines, "\n")

	// Apply border style
	finalLineCount := strings.Count(columnContent, "\n") + 1
	logDebug("About to render with lipgloss: width=%d, height=%d, visibleLines=%d, finalLineCount=%d", width, height, len(visibleLines), finalLineCount)

	var result string
	defer func() {
		if r := recover(); r != nil {
			logDebug("PANIC in lipgloss render: %v", r)
			logDebug("width=%d, height=%d, isActive=%v, contentLen=%d", width, height, isActive, len(columnContent))
			// Return a simple string instead of panicking
			result = fmt.Sprintf("Error rendering column (w=%d,h=%d)", width, height)
		}
	}()

	// Don't set explicit height on the style - let it use the content height
	// Setting Height() would add padding to reach that height, making content longer
	if isActive {
		result = activeColumnStyle.Width(width).Render(columnContent)
	} else {
		result = inactiveColumnStyle.Width(width).Render(columnContent)
	}

	resultLineCount := strings.Count(result, "\n") + 1
	logDebug("Lipgloss render successful - input lines: %d, output lines: %d, expected max: %d", finalLineCount, resultLineCount, height)

	if resultLineCount > height+4 {
		logDebug("WARNING: Output exceeds expected height by %d lines!", resultLineCount-(height+4))
	}

	return result
}
