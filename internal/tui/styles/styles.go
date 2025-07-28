package styles

import "github.com/charmbracelet/lipgloss"

// Centralized Lip Gloss styles for TUI components in Rulem.
// All colors are specified using hex codes.

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ff5fd2")).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginBottom(1)

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5fd7ff")).
			Padding(0, 1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff005f")).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff5f")).
			Bold(true)

	NormalTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			MarginBottom(1)

	HelpStyle = lipgloss.NewStyle().
			Faint(true).
			Foreground(lipgloss.Color("#a8a8a8"))

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5fd7ff"))
)
