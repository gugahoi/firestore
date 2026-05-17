package browse

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func extractColumnTitle(col Column) string {
	if col.path == "" {
		return "Collections"
	}
	parts := strings.Split(col.path, "/")
	return parts[len(parts)-1]
}

func verticalSeparator(height int) string {
	styled := columnSeparatorStyle.Render(" │ ")
	lines := make([]string, height)
	for i := range lines {
		lines[i] = styled
	}
	return strings.Join(lines, "\n")
}

// View renders the entire TUI
func (m Model) View() string {
	defer func() {
		if r := recover(); r != nil {
			logDebug("PANIC in View: %v", r)
		}
	}()

	logDebug("View called: width=%d, height=%d, columns=%d", m.width, m.height, len(m.columns))
	if m.width == 0 || m.height == 0 {
		logDebug("View returning early: dimensions not set")
		return "Initializing..."
	}

	// Breadcrumb header: project › segment1 › segment2
	var breadcrumb strings.Builder
	breadcrumb.WriteString(headerProjectStyle.Render(m.projectID))
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
				breadcrumb.WriteString(headerSepStyle.Render("  ›  "))
				breadcrumb.WriteString(headerPathStyle.Render(segment))
			}
		}
	}

	modeIndicator := modeNormalStyle.Render("[" + m.mode.String() + "]")
	switch m.mode {
	case ModeVisual:
		label := fmt.Sprintf("[VISUAL %d]", m.selection.Count())
		modeIndicator = modeVisualStyle.Render(label)
	case ModeCommand:
		modeIndicator = modeCommandStyle.Render("[COMMAND]")
	}

	headerLeft := breadcrumb.String()
	headerRight := modeIndicator
	headerGap := strings.Repeat(" ", max(m.width-lipgloss.Width(headerLeft)-lipgloss.Width(headerRight)-4, 1))
	header := "  " + headerLeft + headerGap + headerRight + "  "

	// Columns
	columnsView := m.renderColumns()
	logDebug("View: columnsView length=%d", len(columnsView))

	// Separator line + combined status/footer
	separatorLine := "  " + footerSepStyle.Render(strings.Repeat("─", max(m.width-4, 0)))

	// Build left side: status info
	var statusParts []string
	if m.activeColumn < len(m.columns) {
		col := m.columns[m.activeColumn]
		if !col.isDoc && col.docCount > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%d docs", col.docCount))
		}
		if col.activeQuery != nil {
			statusParts = append(statusParts, "Query: "+col.activeQuery.String())
		}
	}
	if m.filterActive && m.filterPattern != "" {
		statusParts = append(statusParts, "Filter: "+m.filterPattern)
	}
	if m.previewEnabled {
		statusParts = append(statusParts, "Preview: ON")
	}

	// Error/loading inline
	if m.err != nil {
		statusParts = append(statusParts, errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}
	if m.loading {
		statusParts = append(statusParts, loadingStyle.Render("Loading..."))
	}

	statusLeft := strings.Join(statusParts, "  │  ")

	// Footer: context-sensitive hints or command input
	var footer string
	if m.mode == ModeCommand {
		footer = m.commandInput.View()
		if m.commandResult != "" {
			separatorLine = "  " + footerStyle.Render(m.commandResult)
		}
	} else if m.overlay == OverlayFilter {
		footer = m.filterInput.View()
	} else {
		hintsRight := m.getFooterHints()
		gap := strings.Repeat(" ", max(m.width-lipgloss.Width(statusLeft)-lipgloss.Width(hintsRight)-4, 1))
		footer = footerStyle.Render(statusLeft + gap + hintsRight)
	}

	// Combine all parts
	logDebug("View: about to JoinVertical")
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		columnsView,
		separatorLine,
		footer,
	)
	// Ensure content fills terminal height so overlays center correctly
	content = lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, content)
	logDebug("View: JoinVertical successful, contentLen=%d", len(content))

	// Overlay dialogs — all centered over the main layout
	if m.overlay == OverlayInfo {
		scrollHint := "Esc/q: close"
		if m.infoViewport.TotalLineCount() > m.infoViewport.VisibleLineCount() {
			pct := m.infoViewport.ScrollPercent() * 100
			scrollHint = fmt.Sprintf("j/k: scroll  (%0.f%%)  Esc/q: close", pct)
		}
		dialogContent := lipgloss.JoinVertical(
			lipgloss.Left,
			m.infoViewport.View(),
			footerSepStyle.Render(strings.Repeat("─", m.infoViewport.Width)),
			footerStyle.Render(scrollHint),
		)
		dialog := infoDialogStyle.Render(dialogContent)
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
		)
	}

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
		var dialogTitle, dialogDetail string
		if len(m.bulkDeletePaths) > 0 {
			dialogTitle = fmt.Sprintf("Delete %d documents?", len(m.bulkDeletePaths))
			dialogDetail = ""
		} else {
			dialogTitle = "Delete document?"
			parts := strings.Split(m.deletePath, "/")
			dialogDetail = m.deletePath
			if len(parts) > 0 {
				dialogDetail = parts[len(parts)-1]
			}
		}

		dialogContent := lipgloss.JoinVertical(
			lipgloss.Center,
			errorStyle.Render(dialogTitle),
			"",
			dialogDetail,
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

	// Floating status notification
	if m.statusMsg != "" {
		notificationStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Background(lipgloss.Color("235")).
			Foreground(colorGreen).
			Padding(0, 1).
			Bold(true)

		notification := notificationStyle.Render(m.statusMsg)

		lines := strings.Split(content, "\n")

		notificationLine := len(lines) - 3
		if notificationLine < 2 {
			notificationLine = 2
		}
		if notificationLine >= len(lines) {
			notificationLine = len(lines) - 1
		}

		notificationLines := strings.Split(notification, "\n")

		for i, notifLine := range notificationLines {
			linePos := notificationLine + i
			if linePos < len(lines) {
				notifWidth := lipgloss.Width(notifLine)
				existingLine := lines[linePos]
				existingWidth := lipgloss.Width(existingLine)

				leftPad := existingWidth - notifWidth - 2
				if leftPad < 0 {
					leftPad = 0
				}

				if leftPad+notifWidth <= existingWidth {
					lines[linePos] = strings.Repeat(" ", leftPad) + notifLine
				} else {
					lines[linePos] = notifLine
				}
			}
		}

		content = strings.Join(lines, "\n")
	}

	return content
}

// renderColumns renders all visible columns side by side with separators
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

	// Height for columns area: total height minus header(1), blank(1), separator(1), footer(1), margin(1)
	height := m.height - 5
	if height < 6 {
		height = 6
	}

	// Calculate column width
	sepWidth := 3 // " │ "
	margin := 4  // 2 padding on each side

	previewWidth := 0
	availWidth := m.width - margin
	if m.previewEnabled && len(m.previewNodes) > 0 {
		previewWidth = (m.width - margin) / 3
		availWidth -= previewWidth + sepWidth
	}

	colWidth := calculateColumnWidth(availWidth, len(visibleCols))
	logDebug("renderColumns: visibleCols=%d, colWidth=%d, previewWidth=%d", len(visibleCols), colWidth, previewWidth)

	var parts []string
	for i, col := range visibleCols {
		isActive := (i == m.activeColumn) || (len(m.columns) > 4 && i == 3)

		if i > 0 {
			parts = append(parts, verticalSeparator(height))
		}

		rendered := m.renderColumn(col, colWidth, height, isActive)
		logDebug("renderColumns: column %d rendered, length=%d", i, len(rendered))
		parts = append(parts, rendered)
	}

	// Add preview pane if enabled
	if m.previewEnabled && previewWidth > 0 && len(m.previewNodes) > 0 {
		parts = append(parts, verticalSeparator(height))
		previewCol := m.renderPreviewPane(previewWidth, height)
		parts = append(parts, previewCol)
	}

	logDebug("renderColumns: about to JoinHorizontal with %d parts", len(parts))
	result := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	logDebug("renderColumns: JoinHorizontal successful, resultLen=%d", len(result))

	// Add horizontal margin
	return lipgloss.NewStyle().Padding(0, 2).Render(result)
}

// renderColumn renders a single column (borderless)
func (m Model) renderColumn(col Column, width, height int, isActive bool) string {
	defer func() {
		if r := recover(); r != nil {
			logPanic("renderColumn", r)
			panic(r)
		}
	}()

	// Column header: title + underline
	title := extractColumnTitle(col)
	var headerBuf strings.Builder
	if isActive {
		headerBuf.WriteString(columnTitleStyle.Render(title))
	} else {
		headerBuf.WriteString(columnTitleInactiveStyle.Render(title))
	}
	headerBuf.WriteString("\n")
	underlineLen := min(lipgloss.Width(title), width)
	headerBuf.WriteString(columnUnderlineStyle.Render(strings.Repeat("─", underlineLen)))
	headerBuf.WriteString("\n")

	headerLines := 2
	itemsHeight := height - headerLines
	if itemsHeight < 1 {
		itemsHeight = 1
	}

	innerWidth := max(width-2, 1) // 1 char padding on each side for content

	// Apply filter to sections for active column
	sections := m.getEffectiveSections(col)

	logDebug(
		"renderColumn: width=%d, innerWidth=%d, height=%d, isActive=%v, sections=%d",
		width, innerWidth, height, isActive, len(sections),
	)

	var content strings.Builder
	itemIndex := 0
	linesUsed := 0

	wrapper := lipgloss.NewStyle().Width(innerWidth)

	for _, section := range sections {
		if section.hidden {
			continue
		}

		// Section header (subtle label)
		if section.title != "Documents" || col.isDoc {
			sTitle := section.title
			if len(sTitle) > innerWidth {
				sTitle = sTitle[:innerWidth-3] + "..."
			}
			content.WriteString(sectionHeaderStyle.Render(sTitle))
			content.WriteString("\n")
			linesUsed++
		}

		if section.title == "Data" {
			idx := itemIndex
			used := m.renderTreeNodes(
				section.items, 0, col.cursor, &idx, &content, wrapper, innerWidth, isActive,
			)
			linesUsed += used
			itemIndex = idx
		} else {
			for _, item := range section.items {
				prefix := "  "
				if itemIndex == col.cursor && isActive {
					prefix = cursorBarStyle.Render("▌") + " "
				}

				selMark := ""
				if m.mode == ModeVisual && m.selection.IsSelected(itemIndex) {
					selMark = "● "
				}

				availWidth := max(innerWidth-lipgloss.Width(prefix)-lipgloss.Width(selMark)-1, 5)
				displayID := item.id
				if len(displayID) > availWidth {
					displayID = displayID[:availWidth-3] + "..."
				}

				// Use · prefix for items in document sections
				dot := ""
				if col.isDoc && section.title != "Data" {
					dot = "· "
				}

				line := fmt.Sprintf("%s%s%s%s", prefix, selMark, dot, displayID)
				if item.isMissing {
					line = fmt.Sprintf("%s%s%s%s", prefix, selMark, dot, missingDocStyle.Render(displayID+" (no data)"))
				}

				if m.mode == ModeVisual && m.selection.IsSelected(itemIndex) {
					line = visualSelectedStyle.Render(line)
				} else if itemIndex == col.cursor && isActive {
					line = selectedItemStyle.Render(line)
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

	// Raw JSON fallback for document columns without Data section
	hasDataSection := false
	for _, s := range sections {
		if s.title == "Data" {
			hasDataSection = true
			break
		}
	}

	if col.isDoc && col.docContent != "" && !hasDataSection {
		remainingLines := itemsHeight - linesUsed - 2
		logDebug("DOCUMENT CONTENT: height=%d, linesUsed=%d, remainingLines=%d", height, linesUsed, remainingLines)

		if remainingLines > 2 {
			headerStr := sectionHeaderStyle.Render("Document Content")
			wrappedHeader := wrapper.Render(headerStr)
			content.WriteString(wrappedHeader)
			content.WriteString("\n")
			linesUsed += strings.Count(wrappedHeader, "\n") + 1
			remainingLines -= strings.Count(wrappedHeader, "\n") + 1

			wrappedDoc := wrapper.Render(col.docContent)
			docLines := strings.Split(wrappedDoc, "\n")
			logDebug("DOCUMENT CONTENT: original docLines=%d, remainingLines=%d", len(docLines), remainingLines)

			content.WriteString(strings.Join(docLines, "\n"))
			linesUsed += len(docLines)
			logDebug("DOCUMENT CONTENT: after adding doc, linesUsed=%d", linesUsed)
		}
	}

	// Apply scrolling
	columnContent := content.String()
	if len(columnContent) == 0 {
		columnContent = "No items"
	}

	lines := strings.Split(columnContent, "\n")
	totalLines := len(lines)

	maxScroll := max(totalLines-itemsHeight, 0)
	scrollOffset := min(max(col.scrollOffset, 0), maxScroll)

	startLine := scrollOffset
	endLine := min(scrollOffset+itemsHeight, totalLines)
	visibleLines := lines[startLine:endLine]

	if len(visibleLines) > itemsHeight {
		visibleLines = visibleLines[:itemsHeight]
	}

	if scrollOffset > 0 && len(visibleLines) > 0 {
		visibleLines[0] = "▲ " + visibleLines[0]
	}
	if scrollOffset < maxScroll && len(visibleLines) > 0 {
		visibleLines[len(visibleLines)-1] = "▼ " + visibleLines[len(visibleLines)-1]
	}

	scrolledContent := strings.Join(visibleLines, "\n")

	// Combine header + scrolled content
	finalContent := headerBuf.String() + scrolledContent

	// Render with fixed dimensions (no border)
	return lipgloss.NewStyle().Width(width).Height(height).Render(finalContent)
}

// renderTreeNodes renders tree nodes recursively
func (m Model) renderTreeNodes(
	nodes []ListItem,
	level int,
	cursor int,
	itemIndex *int,
	content *strings.Builder,
	wrapper lipgloss.Style,
	innerWidth int,
	isActive bool,
) int {
	linesUsed := 0

	for _, node := range nodes {
		prefix := strings.Repeat("  ", level)

		cursorMarker := "  "
		if *itemIndex == cursor && isActive {
			cursorMarker = cursorBarStyle.Render("▌") + " "
		}

		icon := " "
		if node.dataType == "object" || node.dataType == "array" {
			if node.expanded {
				icon = "▼"
			} else {
				icon = "▶"
			}
		}

		keyStr := treeKeyStyle.Render(node.key)

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
			if node.dataType == "array" {
				valStr = treeTypeStyle.Render(fmt.Sprintf("[%d]", len(node.children)))
			} else {
				valStr = treeTypeStyle.Render("{}")
			}
		default:
			valStr = node.valueStr
		}

		lineStr := fmt.Sprintf("%s%s%s %s: %s", cursorMarker, prefix, icon, keyStr, valStr)

		if *itemIndex == cursor && isActive {
			lineStr = lipgloss.NewStyle().Bold(true).Render(lineStr)
		}

		wrappedLine := wrapper.Render(lineStr)
		content.WriteString(wrappedLine)
		content.WriteString("\n")
		linesUsed += strings.Count(wrappedLine, "\n") + 1

		*itemIndex++

		if node.expanded && len(node.children) > 0 {
			linesUsed += m.renderTreeNodes(
				node.children, level+1, cursor, itemIndex, content, wrapper, innerWidth, isActive,
			)
		}
	}

	return linesUsed
}

func (m Model) renderPreviewPane(width, height int) string {
	innerWidth := max(width-2, 1)
	var headerBuf strings.Builder

	parts := strings.Split(m.previewPath, "/")
	title := m.previewPath
	if len(parts) > 0 {
		title = parts[len(parts)-1]
	}

	headerBuf.WriteString(columnTitleStyle.Render("Preview: " + title))
	headerBuf.WriteString("\n")
	underlineLen := min(lipgloss.Width("Preview: "+title), width)
	headerBuf.WriteString(columnUnderlineStyle.Render(strings.Repeat("─", underlineLen)))
	headerBuf.WriteString("\n")

	var content strings.Builder
	wrapper := lipgloss.NewStyle().Width(innerWidth)
	itemIndex := 0
	m.renderTreeNodes(m.previewNodes, 0, -1, &itemIndex, &content, wrapper, innerWidth, false)

	itemsHeight := height - 2
	columnContent := content.String()
	lines := strings.Split(columnContent, "\n")
	if len(lines) > itemsHeight {
		lines = lines[:itemsHeight]
	}

	finalContent := headerBuf.String() + strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(finalContent)
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
		return "v: Toggle select  j/k: Move  d: Delete selected  Esc: Cancel"
	case ModeCommand:
		return "Esc: Normal mode"
	default:
		return "j/k ↕  h/l ↔  g/G Top/Bottom  / Filter  s Sort  e Edit  d Del  yy Copy  : Cmd  q Quit"
	}
}

// renderStatusBar is kept for compatibility but status is now merged into footer
func (m Model) renderStatusBar() string {
	return ""
}
