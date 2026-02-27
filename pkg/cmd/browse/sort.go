package browse

import (
	"cloud.google.com/go/firestore"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// sortDialogModel represents the state of the sort dialog
type sortDialogModel struct {
	fields           []string              // Available fields from loaded documents
	textInput        textinput.Model       // For free-text field entry
	list             list.Model            // Bubbles list for field selection
	direction        firestore.Direction   // firestore.Asc or firestore.Desc
	focusedComponent int                   // 0 for text input, 1 for list
}

// fieldItem implements list.Item interface for bubbles list
type fieldItem struct {
	name string
}

func (i fieldItem) FilterValue() string { return i.name }
func (i fieldItem) Title() string       { return i.name }
func (i fieldItem) Description() string { return "" }

// initSortDialog creates and initializes a new sort dialog
func initSortDialog(fields []string) sortDialogModel {
	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "field name"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40

	// Convert fields to list items
	items := make([]list.Item, len(fields))
	for i, field := range fields {
		items[i] = fieldItem{name: field}
	}

	// Initialize list
	l := list.New(items, list.NewDefaultDelegate(), 42, 12)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle()
	l.SetShowHelp(false)

	return sortDialogModel{
		fields:           fields,
		textInput:        ti,
		list:             l,
		direction:        firestore.Asc,
		focusedComponent: 0, // Start with text input focused
	}
}

// Update handles input for the sort dialog
func (d *sortDialogModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	if d.focusedComponent == 0 {
		// Text input is focused
		d.textInput, cmd = d.textInput.Update(msg)
	} else {
		// List is focused
		d.list, cmd = d.list.Update(msg)
	}

	return cmd
}

// View renders the sort dialog
func (d sortDialogModel) View() string {
	// Direction display
	directionText := "Ascending ▲"
	if d.direction == firestore.Desc {
		directionText = "Descending ▼"
	}
	directionLine := sortDirectionStyle.Render("Direction: " + directionText)

	// Text input section
	textInputLabel := "Field (type or select):"
	if d.focusedComponent == 0 {
		textInputLabel = "> " + textInputLabel
	} else {
		textInputLabel = "  " + textInputLabel
	}
	textInputSection := lipgloss.JoinVertical(
		lipgloss.Left,
		textInputLabel,
		"  "+d.textInput.View(),
	)

	// List section
	listLabel := "Available fields:"
	if d.focusedComponent == 1 {
		listLabel = "> " + listLabel
	} else {
		listLabel = "  " + listLabel
	}
	listSection := lipgloss.JoinVertical(
		lipgloss.Left,
		listLabel,
		d.list.View(),
	)

	// Keyboard hints
	hints := sortHintStyle.Render("Tab: Switch  Ctrl+d: Direction  ↑↓: Navigate  Enter: OK  Esc: Cancel")

	// Combine all sections
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		"Sort Collection",
		"",
		directionLine,
		"",
		textInputSection,
		"",
		listSection,
		"",
		hints,
	)

	return sortDialogStyle.Render(content)
}

// getSelectedField returns the field to sort by
// Priority: text input if non-empty, otherwise selected list item
func (d sortDialogModel) getSelectedField() string {
	// Check text input first
	if d.textInput.Value() != "" {
		return d.textInput.Value()
	}

	// Fall back to selected list item
	if item := d.list.SelectedItem(); item != nil {
		return item.(fieldItem).name
	}

	return ""
}
