package styles

import "github.com/charmbracelet/lipgloss"

// Centralized Lip Gloss styles for TUI components in Rulem.
// All colors are specified using hex codes.

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ff5fd2")).
			MarginBottom(1).
			PaddingLeft(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginBottom(1).
			PaddingLeft(1)

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
			Foreground(lipgloss.Color("#a8a8a8")).
			MarginTop(1).
			Padding(0, 1)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5fd7ff"))

	// Containers for consistent layout spacing
	HeaderContainerStyle = lipgloss.NewStyle().
				MarginLeft(1).
				MarginBottom(1)

	HelpContainerStyle = lipgloss.NewStyle().
				MarginLeft(1).
				MarginTop(1)

	// Left padding for the main panes area to align with header/help
	MainContainerStyle = lipgloss.NewStyle().
				MarginLeft(1)

	// Shared pane styles for components (e.g., FilePicker).
	// Default pane with rounded border, hex colors, and sensible spacing.
	PaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5f5fff")). // default border color
			PaddingLeft(2).
			PaddingRight(1)

	// Focused pane variant that highlights the active pane.
	PaneFocusedStyle = PaneStyle.
				BorderForeground(lipgloss.Color("#ff5faf"))
)
