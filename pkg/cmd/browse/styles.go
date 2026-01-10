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
)
