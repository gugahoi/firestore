package browse

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Muted/Pastel color palette
	colorAccent    = lipgloss.Color("75")  // Slate blue — primary accent
	colorDim       = lipgloss.Color("243") // Gray — inactive/secondary text
	colorSeparator = lipgloss.Color("238") // Dim gray — structural lines
	colorHeader    = lipgloss.Color("252") // Near white — column headers
	colorError     = lipgloss.Color("167") // Soft red
	colorSuccess   = lipgloss.Color("108") // Sage green
	colorWarning   = lipgloss.Color("222") // Soft yellow

	// Keep color aliases used by notification overlay in view.go
	colorGreen = lipgloss.Color("108")
	colorGray  = lipgloss.Color("243")

	// Breadcrumb header
	headerProjectStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorHeader)

	headerSepStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	headerPathStyle = lipgloss.NewStyle().
		Foreground(colorAccent)

	// Column title
	columnTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorHeader)

	columnTitleInactiveStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	columnUnderlineStyle = lipgloss.NewStyle().
		Foreground(colorSeparator)

	// Separator between columns (vertical pipe)
	columnSeparatorStyle = lipgloss.NewStyle().
		Foreground(colorSeparator)

	// Cursor bar for active column
	cursorBarStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	// Footer
	footerSepStyle = lipgloss.NewStyle().
		Foreground(colorSeparator)

	footerStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Padding(0, 2)

	// Section header (subtle label within columns)
	sectionHeaderStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Bold(true)

	// Selected item
	selectedItemStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	// Document indicator style
	docIndicatorStyle = lipgloss.NewStyle().
		Foreground(colorSuccess)

	// Missing document
	missingDocStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Faint(true)

	// Collection indicator style
	colIndicatorStyle = lipgloss.NewStyle().
		Foreground(colorAccent)

	// Error message
	errorStyle = lipgloss.NewStyle().
		Foreground(colorError).
		Bold(true)

	// Loading indicator
	loadingStyle = lipgloss.NewStyle().
		Foreground(colorWarning).
		Italic(true)

	// Status message style
	statusStyle = lipgloss.NewStyle().
		Foreground(colorSuccess).
		Italic(true).
		Padding(0, 2)

	// Tree View Data Type Styles (muted/pastel)
	treeKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("139")) // Lavender

	treeStringStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("108")) // Sage green

	treeNumberStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("180")) // Warm sand

	treeBoolStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("110")) // Soft blue

	treeNullStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("242")). // Dim gray
		Italic(true)

	treeTypeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")). // Light gray
		Italic(true)

	// Dialogs (keep borders — they float over content)
	infoDialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 3)

	inputDialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2)

	sortDialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2).
		Width(50)

	sortDirectionStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorWarning).
		Align(lipgloss.Center)

	sortHintStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Italic(true)

	deleteDialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorError).
		Padding(1, 2).
		Width(50)

	// Mode indicator styles
	modeNormalStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorSuccess).
		Padding(0, 1)

	modeVisualStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorWarning).
		Padding(0, 1)

	modeCommandStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorAccent).
		Padding(0, 1)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Padding(0, 2)

	// Visual mode selected item
	visualSelectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorWarning)

	// Preview pane (borderless, matches other columns)
	previewStyle = lipgloss.NewStyle()

	// Path style (used by header breadcrumb)
	pathStyle = headerPathStyle

	// Legacy alias — headerStyle is used in overlay dialog
	headerStyle = headerProjectStyle
)
