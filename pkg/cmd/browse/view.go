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

	// Construct path string from columns
	var pathSegments []string
	for i, col := range m.columns {
		if i > m.activeColumn {
			break
		}
		if i == 0 && col.path == "" {
			continue
		}
		parts := strings.Split(col.path, "/")
		if len(parts) > 0 {
			segment := parts[len(parts)-1]
			if segment != "" {
				pathSegments = append(pathSegments, segment)
			}
		}
	}
	fullPath := "/" + strings.Join(pathSegments, "/")

	// Header: project ID | path | mode indicator
	modeIndicator := modeNormalStyle.Render("[" + m.mode.String() + "]")
	switch m.mode {
	case ModeVisual:
		modeIndicator = modeVisualStyle.Render("[VISUAL]")
	case ModeCommand:
		modeIndicator = modeCommandStyle.Render("[COMMAND]")
	}

	headerLeft := headerStyle.Render(m.projectID)
	headerPath := pathStyle.Render(fullPath)
	headerRight := modeIndicator
	headerGap := strings.Repeat(" ", max(m.width-lipgloss.Width(headerLeft)-lipgloss.Width(headerPath)-lipgloss.Width(headerRight)-2, 1))
	header := headerLeft + " " + headerPath + headerGap + headerRight

	// Footer: context-sensitive hints, command prompt, or filter input
	var footer string
	if m.mode == ModeCommand {
		commandLine := m.commandInput.View()
		if m.commandResult != "" {
			footer = lipgloss.JoinVertical(lipgloss.Left,
				footerStyle.Render(m.commandResult),
				commandLine,
			)
		} else {
			footer = commandLine
		}
	} else if m.overlay == OverlayFilter {
		footer = m.filterInput.View()
	} else {
		footer = footerStyle.Render(m.getFooterHints())
	}

	// Status bar (placeholder between columns and footer)
	statusBar := m.renderStatusBar()

	// Error message
	errorView := ""
	if m.err != nil {
		errorView = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
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
		statusBar,
		errorView,
		loadingMsg,
		footer,
	)
	logDebug("View: JoinVertical successful, contentLen=%d", len(content))

	// Overlay dialogs
	if m.overlay == OverlayPathInput {
		dialogContent := lipgloss.JoinVertical(
			lipgloss.Center,
			"Enter Firestore Path:",
			m.textInput.View(),
		)
		dialog := inputDialogStyle.Render(dialogContent)
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(colorGray),
		)
	}

	if m.overlay == OverlaySortDialog {
		dialog := m.sortDialog.View()
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
		)
	}

	if m.overlay == OverlayDeleteConfirm {
		parts := strings.Split(m.deletePath, "/")
		docID := m.deletePath
		if len(parts) > 0 {
			docID = parts[len(parts)-1]
		}

		dialogContent := lipgloss.JoinVertical(
			lipgloss.Center,
			errorStyle.Render("Delete document?"),
			"",
			docID,
			"",
			footerStyle.Render("y/Enter: confirm  n/Esc: cancel"),
		)
		dialog := deleteDialogStyle.Render(dialogContent)
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
		)
	}

	// Overlay status notification if present (floating window)
	if m.statusMsg != "" {
		// Create notification box with border
		notificationStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Background(lipgloss.Color("235")). // Dark background
			Foreground(colorGreen).
			Padding(0, 1).
			Bold(true)

		notification := notificationStyle.Render(m.statusMsg)

		// Split content into lines
		lines := strings.Split(content, "\n")

		// Calculate position near bottom but above footer
		// Footer is typically the last line, so place notification a few lines above
		notificationLine := len(lines) - 3
		if notificationLine < 2 {
			notificationLine = 2 // Minimum position from top
		}

		// Make sure we don't go beyond the content
		if notificationLine >= len(lines) {
			notificationLine = len(lines) - 1
		}

		// Position notification at bottom right
		notificationLines := strings.Split(notification, "\n")

		// Overlay each line of the notification at the right edge
		for i, notifLine := range notificationLines {
			linePos := notificationLine + i
			if linePos < len(lines) {
				notifWidth := lipgloss.Width(notifLine)
				existingLine := lines[linePos]
				existingWidth := lipgloss.Width(existingLine)

				// Calculate padding to align to the right
				leftPad := existingWidth - notifWidth - 2 // -2 for a small margin from right edge
				if leftPad < 0 {
					leftPad = 0
				}

				// Align to right by adding left padding
				if leftPad+notifWidth <= existingWidth {
					// Right-align with padding
					lines[linePos] = strings.Repeat(" ", leftPad) + notifLine
				} else {
					// Notification is wider than existing line, just place it
					lines[linePos] = notifLine
				}
			}
		}

		// Rejoin the content
		content = strings.Join(lines, "\n")
	}

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

// Helper to render tree nodes recursively
func (m Model) renderTreeNodes(
	nodes []ListItem,
	level int,
	cursor int,
	itemIndex *int,
	content *strings.Builder,
	wrapper lipgloss.Style,
	innerWidth int,
) int {
	linesUsed := 0

	for _, node := range nodes {
		prefix := strings.Repeat("  ", level)

		cursorMarker := "  "
		if *itemIndex == cursor {
			cursorMarker = "> "
		}

		// Icon for expandable items
		icon := " "
		if node.dataType == "object" || node.dataType == "array" {
			if node.expanded {
				icon = "▼"
			} else {
				icon = "▶"
			}
		}

		// Style the key
		keyStr := treeKeyStyle.Render(node.key)

		// Style the value
		valStr := ""
		switch node.dataType {
		case "string":
			valStr = treeStringStyle.Render(fmt.Sprintf("%q", node.valueStr))
		case "number":
			valStr = treeNumberStyle.Render(node.valueStr)
		case "bool":
			valStr = treeBoolStyle.Render(node.valueStr)
		case "null":
			valStr = treeNullStyle.Render("null")
		case "object", "array":
			// Show type hint (e.g. "Object {2}" or "Array [5]")
			// If we had count, we could show it, for now just show type
			if node.dataType == "array" {
				valStr = treeTypeStyle.Render(fmt.Sprintf("[%d]", len(node.children)))
			} else {
				valStr = treeTypeStyle.Render("{}")
			}
		default:
			valStr = node.valueStr
		}

		// Build line: marker + indent + icon + key + ": " + value
		// Note: we removed the dot point for leaf nodes as requested
		lineStr := fmt.Sprintf("%s%s%s %s: %s", cursorMarker, prefix, icon, keyStr, valStr)

		// Style the line selection (inverse background or similar might be better,
		// but let's stick to simple bold/highlight for now, preserving color)
		if *itemIndex == cursor {
			// When selected, we might want to keep the colors but add an indicator
			// or change background. For now, let's just make the cursor indicator bold/visible
			// and keep the syntax highlighting.
			// Re-building line with selected indicator style only on the marker/prefix if possible
			// But since we are rendering the whole string, let's just use a selection style
			// that doesn't strip color if possible, or just bold it.

			// Simple approach: Use a different background for the whole line?
			// Or just rely on the ">" marker which is already there.
			// Let's wrap the whole line in a style that might add a background or bold
			// without overriding foreground colors if lipgloss supports it.
			// Lipgloss Foreground() overrides existing colors.
			// So let's just use the marker ">" which is already distinct.

			// However, the original code did: lineStr = selectedItemStyle.Render(lineStr)
			// which forces white color. Let's just bold it and maybe add a background.
			lineStr = lipgloss.NewStyle().Bold(true).Render(lineStr)
		}

		// Wrap and write
		wrappedLine := wrapper.Render(lineStr)
		content.WriteString(wrappedLine)
		content.WriteString("\n")
		linesUsed += strings.Count(wrappedLine, "\n") + 1

		*itemIndex++

		// Render children if expanded
		if node.expanded && len(node.children) > 0 {
			linesUsed += m.renderTreeNodes(
				node.children,
				level+1,
				cursor,
				itemIndex,
				content,
				wrapper,
				innerWidth,
			)
		}
	}

	return linesUsed
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
	// m.height includes:
	// - header (1)
	// - status bar (1)
	// - footer (1)
	// - error/loading (1)
	// - some margin (1)
	// Total overhead = 5
	// We also need to account for the border (top + bottom = 2)
	height := m.height - 5 - 2
	if height < 1 {
		height = 10 // Minimum height
	}

	// Calculate inner width for wrapping
	// Width - Border(2) - Padding(2) = Width - 4
	innerWidth := max(width-4, 1)

	// Apply filter to sections for active column
	sections := m.getEffectiveSections(col)

	logDebug(
		"renderColumn: width=%d, innerWidth=%d, height=%d, isActive=%v, sections=%d",
		width,
		innerWidth,
		height,
		isActive,
		len(sections),
	)

	var content strings.Builder
	itemIndex := 0
	linesUsed := 0

	// Helper style for wrapping document content
	wrapper := lipgloss.NewStyle().Width(innerWidth)

	// Render sections
	for _, section := range sections {
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
		if section.title == "Data" {
			// Special rendering for Data section (tree view)
			// Pass current itemIndex pointer to track global index in column
			idx := itemIndex
			used := m.renderTreeNodes(
				section.items,
				0,
				col.cursor,
				&idx,
				&content,
				wrapper,
				innerWidth,
			)
			linesUsed += used
			itemIndex = idx
		} else {
			// Standard list rendering
			for _, item := range section.items {
				prefix := "  "
				if itemIndex == col.cursor {
					prefix = "> "
				}

				// Calculate space available for ID
				// prefix(2) + space(1) = 3 chars roughly (ignoring ansi)
				extraChars := 3
				availWidth := max(innerWidth-extraChars, 5)

				displayID := item.id
				if len(displayID) > availWidth {
					displayID = displayID[:availWidth-3] + "..."
				}

				line := fmt.Sprintf("%s%s", prefix, displayID)
				if item.isMissing {
					line = fmt.Sprintf("%s%s", prefix, missingDocStyle.Render(displayID+" (no data)"))
				}
				if itemIndex == col.cursor {
					line = selectedItemStyle.Render(
						fmt.Sprintf("%s%s", prefix, displayID),
					)
					if item.isMissing {
						line = selectedItemStyle.Render(
							fmt.Sprintf("%s%s (no data)", prefix, displayID),
						)
					}
				}

				content.WriteString(line)
				content.WriteString("\n")
				itemIndex++
				linesUsed++
			}
		}
		content.WriteString("\n")
		linesUsed++
	}

	// If this is a document column, render document content (RAW JSON fallback/debug)
	// We only show this if we DON'T have a Data section (which we should always have now)
	hasDataSection := false
	for _, s := range sections {
		if s.title == "Data" {
			hasDataSection = true
			break
		}
	}

	if col.isDoc && col.docContent != "" && !hasDataSection {
		// Calculate remaining space for document content
		remainingLines := height - linesUsed - 2 // Leave 2 lines margin
		logDebug(
			"DOCUMENT CONTENT: height=%d, linesUsed=%d, remainingLines=%d",
			height,
			linesUsed,
			remainingLines,
		)

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
			logDebug(
				"DOCUMENT CONTENT: original docLines=%d, remainingLines=%d",
				len(docLines),
				remainingLines,
			)

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

	logDebug(
		"Before scrolling: linesUsed=%d, contentLines=%d, height=%d",
		linesUsed,
		strings.Count(columnContent, "\n")+1,
		height,
	)

	// Apply scrolling - show only the visible portion
	lines := strings.Split(columnContent, "\n")
	totalLines := len(lines)

	// Calculate bounds
	maxScroll := max(totalLines-height, 0)

	// Use scroll offset (bounds are enforced in scroll functions)
	scrollOffset := min(max(col.scrollOffset, 0), maxScroll)

	// Extract visible lines based on scroll position
	startLine := scrollOffset
	endLine := min(scrollOffset+height, totalLines)

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
	logDebug(
		"About to render with lipgloss: width=%d, height=%d, visibleLines=%d, finalLineCount=%d",
		width,
		height,
		len(visibleLines),
		finalLineCount,
	)

	var result string
	defer func() {
		if r := recover(); r != nil {
			logDebug("PANIC in lipgloss render: %v", r)
			logDebug(
				"width=%d, height=%d, isActive=%v, contentLen=%d",
				width,
				height,
				isActive,
				len(columnContent),
			)
			// Return a simple string instead of panicking
			result = fmt.Sprintf("Error rendering column (w=%d,h=%d)", width, height)
		}
	}()

	// Always set explicit height on the style to ensure full height columns
	if isActive {
		result = activeColumnStyle.Width(width).Height(height).Render(columnContent)
	} else {
		result = inactiveColumnStyle.Width(width).Height(height).Render(columnContent)
	}

	resultLineCount := strings.Count(result, "\n") + 1
	logDebug(
		"Lipgloss render successful - input lines: %d, output lines: %d, expected max: %d",
		finalLineCount,
		resultLineCount,
		height,
	)

	if resultLineCount > height+4 {
		logDebug("WARNING: Output exceeds expected height by %d lines!", resultLineCount-(height+4))
	}

	return result
}

// getFooterHints returns context-sensitive keybinding hints
func (m Model) getFooterHints() string {
	if m.overlay == OverlaySortDialog {
		return "Tab: Switch field  Ctrl+d: Toggle direction  Enter: Apply  Esc: Cancel"
	}
	if m.overlay == OverlayDeleteConfirm {
		return "y/Enter: Confirm  n/Esc: Cancel"
	}
	if m.overlay == OverlayPathInput {
		return "Enter: Navigate  Esc: Cancel"
	}
	if m.overlay == OverlayFilter {
		return "Enter: Confirm filter  Esc: Clear and cancel"
	}

	switch m.mode {
	case ModeVisual:
		return "Esc: Normal mode"
	case ModeCommand:
		return "Esc: Normal mode"
	default:
		return "j/k: Up/Down  h/l: Back/Forward  g/G: Top/Bottom  /: Filter  s: Sort  e: Edit  d: Delete  yy: Copy  :: Command  q: Quit"
	}
}

// renderStatusBar renders the status bar area between columns and footer
func (m Model) renderStatusBar() string {
	var parts []string

	// Show pagination info for active collection column
	if m.activeColumn < len(m.columns) {
		col := m.columns[m.activeColumn]
		if !col.isDoc && col.docCount > 0 {
			info := fmt.Sprintf("%d docs | limit: %d", col.docCount, m.pageLimit)
			parts = append(parts, info)
		}
	}

	// Show active filter
	if m.filterActive && m.filterPattern != "" {
		parts = append(parts, "Filter: "+m.filterPattern)
	}

	if len(parts) > 0 {
		return statusBarStyle.Render(strings.Join(parts, "  "))
	}
	return statusBarStyle.Render("")
}
