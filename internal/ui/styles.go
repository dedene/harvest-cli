package ui

import "github.com/charmbracelet/lipgloss"

// Color palette for consistent theming.
const (
	colorPrimary   = lipgloss.Color("12")  // Blue
	colorSelected  = lipgloss.Color("170") // Purple/pink
	colorSuccess   = lipgloss.Color("78")  // Green
	colorError     = lipgloss.Color("196") // Red
	colorDim       = lipgloss.Color("240") // Gray
	colorHighlight = lipgloss.Color("229") // Bright yellow
)

var (
	// TitleStyle for headers and titles.
	TitleStyle lipgloss.Style

	// SelectedStyle for currently selected items.
	SelectedStyle lipgloss.Style

	// NormalStyle for regular text.
	NormalStyle lipgloss.Style

	// DimStyle for secondary/muted text.
	DimStyle lipgloss.Style

	// ErrorStyle for error messages.
	ErrorStyle lipgloss.Style

	// SuccessStyle for success messages.
	SuccessStyle lipgloss.Style

	// SpinnerStyle for loading spinners.
	SpinnerStyle lipgloss.Style

	// HelpStyle for help text.
	HelpStyle lipgloss.Style

	// BorderStyle for bordered containers.
	BorderStyle lipgloss.Style

	// HighlightStyle for emphasized content.
	HighlightStyle lipgloss.Style

	// InputStyle for text inputs.
	InputStyle lipgloss.Style

	// PromptStyle for prompts and labels.
	PromptStyle lipgloss.Style
)

func init() {
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		MarginBottom(1)

	SelectedStyle = lipgloss.NewStyle().
		Foreground(colorSelected).
		Bold(true)

	NormalStyle = lipgloss.NewStyle()

	DimStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(colorError).
		Bold(true)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(colorSuccess).
		Bold(true)

	SpinnerStyle = lipgloss.NewStyle().
		Foreground(colorPrimary)

	HelpStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		MarginTop(1)

	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDim).
		Padding(0, 1)

	HighlightStyle = lipgloss.NewStyle().
		Foreground(colorHighlight).
		Bold(true)

	InputStyle = lipgloss.NewStyle().
		Foreground(colorPrimary)

	PromptStyle = lipgloss.NewStyle().
		Foreground(colorSelected).
		Bold(true)
}
