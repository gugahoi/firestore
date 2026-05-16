package browse

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	colorCyan   = lipgloss.Color("86")
	colorGray   = lipgloss.Color("240")
	colorGreen  = lipgloss.Color("42")
	colorBlue   = lipgloss.Color("39")
	colorRed    = lipgloss.Color("196")
	colorYellow = lipgloss.Color("220")

	// Header style
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			Padding(0, 1)

	// Footer style
	footerStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Padding(0, 1)

	// Path style (for bottom of screen)
	pathStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBlue).
			Padding(0, 1)

	// Active column border
	activeColumnStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorCyan).
				Padding(0, 1)

	// Inactive column border
	inactiveColumnStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorGray).
				Padding(0, 1)

	// Section header style
	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Underline(true).
				Foreground(colorYellow)

	// Selected item style
	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15"))

	// Document indicator style
	docIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	// Missing document style (documents with no data, only subcollections)
	missingDocStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Faint(true)

	// Collection indicator style
	colIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorBlue)

	// Error message style
	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	// Loading indicator style
	loadingStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Italic(true)

	// Status message style
	statusStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Italic(true).
			Padding(0, 1)

	// Tree View Data Type Styles
	treeKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")) // Hot Pink for keys

	treeStringStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("113")) // Green for strings

	treeNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("215")) // Orange/Gold for numbers

	treeBoolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")) // Blue for booleans

	treeNullStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // Dark gray for null
			Italic(true)

	treeTypeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")). // Light gray for object/array type indicators
			Italic(true)

	// Input dialog style
	inputDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorCyan).
				Padding(1, 2)

	// Sort dialog styles
	sortDialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan).
			Padding(1, 2).
			Width(50)

	sortDirectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorYellow).
				Align(lipgloss.Center)

	sortHintStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Italic(true)

	deleteDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorRed).
				Padding(1, 2).
				Width(50)

	// Mode indicator styles
	modeNormalStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(colorGreen).
			Padding(0, 1)

	modeVisualStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(colorYellow).
			Padding(0, 1)

	modeCommandStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(colorBlue).
				Padding(0, 1)

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Padding(0, 1)
)
