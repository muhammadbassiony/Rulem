package components

import (
	"rulem/internal/tui/styles"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
)

type LayoutConfig struct {
	Title    string
	Subtitle string
	HelpText string
	MarginX  int
	MarginY  int
	MaxWidth int
}

type LayoutModel struct {
	config LayoutConfig
	width  int
	height int
	err    error
}

func NewLayout(config LayoutConfig) LayoutModel {
	if config.MarginX == 0 {
		config.MarginX = 2
	}
	if config.MarginY == 0 {
		config.MarginY = 1
	}
	if config.MaxWidth == 0 {
		config.MaxWidth = 100
	}

	return LayoutModel{config: config}
}

func (m LayoutModel) Update(msg tea.Msg) (LayoutModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// Configuration methods - these allow dynamic reconfiguration
func (m LayoutModel) SetTitle(title string) LayoutModel {
	m.config.Title = title
	return m
}

func (m LayoutModel) SetSubtitle(subtitle string) LayoutModel {
	m.config.Subtitle = subtitle
	return m
}

func (m LayoutModel) SetHelpText(helpText string) LayoutModel {
	m.config.HelpText = helpText
	return m
}

func (m LayoutModel) SetError(err error) LayoutModel {
	if err != nil {
		m.err = err
	}
	return m
}

func (m LayoutModel) ClearError() LayoutModel {
	m.err = nil
	return m
}

func (m LayoutModel) GetError() error {
	return m.err
}

func (m LayoutModel) SetConfig(config LayoutConfig) LayoutModel {
	// Preserve defaults for zero values
	if config.MarginX == 0 {
		config.MarginX = m.config.MarginX
	}
	if config.MarginY == 0 {
		config.MarginY = m.config.MarginY
	}
	if config.MaxWidth == 0 {
		config.MaxWidth = m.config.MaxWidth
	}
	m.config = config
	return m
}

// Render the complete layout with content
func (m LayoutModel) Render(content string) string {
	sections := []string{}
	contentWidth := m.ContentWidth()

	// Title section
	if m.config.Title != "" {
		title := m.wrapText(m.config.Title, contentWidth)
		sections = append(sections, styles.TitleStyle.Render(title))
	}

	// Subtitle section
	if m.config.Subtitle != "" {
		subtitle := m.wrapText(m.config.Subtitle, contentWidth)
		sections = append(sections, styles.SubtitleStyle.Render(subtitle))
	}

	// Main content section
	if content != "" {
		wrappedContent := m.wrapText(content, contentWidth)
		sections = append(sections, styles.NormalTextStyle.Render(wrappedContent))
	}

	// Error section
	if m.err != nil {
		errorText := "Error: " + m.err.Error()
		wrappedError := m.wrapText(errorText, contentWidth)
		sections = append(sections, styles.ErrorStyle.Render(wrappedError))
	}

	// Help text section
	if m.config.HelpText != "" {
		help := m.wrapText(m.config.HelpText, contentWidth)
		sections = append(sections, styles.HelpStyle.Render(help))
	}

	joined := strings.Join(sections, "\n\n")
	return m.addMargins(joined)
}

// Robust text wrapping using reflow library
func (m LayoutModel) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	// Handle text that might already contain line breaks
	paragraphs := strings.Split(text, "\n\n")
	var wrappedParagraphs []string

	for _, paragraph := range paragraphs {
		// Split by single line breaks to handle manual line breaks
		lines := strings.Split(paragraph, "\n")
		var wrappedLines []string

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				wrappedLines = append(wrappedLines, "")
				continue
			}

			// Use reflow for proper word wrapping
			wrapped := wordwrap.String(line, width)
			wrappedLines = append(wrappedLines, wrapped)
		}

		wrappedParagraphs = append(wrappedParagraphs, strings.Join(wrappedLines, "\n"))
	}

	return strings.Join(wrappedParagraphs, "\n\n")
}

// Alternative wrapping method with indent support for lists
func (m LayoutModel) wrapTextWithIndent(text string, width int, indent int) string {
	if width <= 0 {
		return text
	}

	// Use reflow's wrap package for more advanced wrapping
	return wrap.String(text, width-indent)
}

func (m LayoutModel) addMargins(content string) string {
	lines := strings.Split(content, "\n")
	marginLeft := strings.Repeat(" ", m.config.MarginX)

	for i, line := range lines {
		lines[i] = marginLeft + line
	}

	marginTop := strings.Repeat("\n", m.config.MarginY)
	marginBottom := strings.Repeat("\n", m.config.MarginY)
	return marginTop + strings.Join(lines, "\n") + marginBottom
}

// Helper methods for responsive design
func (m LayoutModel) ContentWidth() int {
	available := m.width - (m.config.MarginX * 2)
	if available > m.config.MaxWidth {
		return m.config.MaxWidth
	}
	if available < 40 {
		return 40 // Minimum readable width
	}
	return available
}

func (m LayoutModel) ContentHeight() int {
	return m.height - (m.config.MarginY * 2) - 6 // Reserve space for sections
}

// Input width calculation for forms
func (m LayoutModel) InputWidth() int {
	contentWidth := m.ContentWidth()
	inputWidth := contentWidth - 8

	if inputWidth > 80 {
		return 80 // Maximum input width for readability
	}
	if inputWidth < 30 {
		return 30 // Minimum input width
	}
	return inputWidth
}

// Helper to check if we have enough space for content
func (m LayoutModel) HasSufficientSpace() bool {
	return m.ContentWidth() >= 40 && m.ContentHeight() >= 10
}

// Get current configuration (useful for debugging)
func (m LayoutModel) GetConfig() LayoutConfig {
	return m.config
}
