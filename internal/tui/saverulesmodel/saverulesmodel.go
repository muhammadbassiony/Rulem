package saverulesmodel

import (
	"fmt"
	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/tui/components"
	"rulem/internal/tui/styles"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type SaveFileModelState int

const (
	StateFileSelection SaveFileModelState = iota
	StateConfirmation
	StateSaving
	StateLoading
	StateError
	// StateSuccess
	// StateCancel
	// StateQuitting
)

// Custom messages for async operations
type (
	FileScanCompleteMsg struct {
		Files []string
	}

	FileScanErrorMsg struct {
		Err error
	}

	StartSaveFileMsg struct {
		FilePath string
	}

	SaveFileCompleteMsg struct{}
)

type SaveRulesModel struct {
	config *config.Config
	logger *logging.AppLogger
	state  SaveFileModelState

	// Layout for consistent UI
	layout  components.LayoutModel
	spinner spinner.Model

	// Data
	markdownFiles []string
	err           error
}

func NewSaveRulesModel(cfg *config.Config, logger *logging.AppLogger) SaveRulesModel {
	// Create layout
	layout := components.NewLayout(components.LayoutConfig{
		MarginX:  2,
		MarginY:  1,
		MaxWidth: 100,
	})

	// Initialize spinner
	s := spinner.New()
	s.Style = styles.SpinnerStyle
	s.Spinner = spinner.Pulse

	return SaveRulesModel{
		config:        cfg,
		logger:        logger,
		state:         StateLoading,
		layout:        layout,
		spinner:       s,
		markdownFiles: []string{},
		err:           nil,
	}
}

func (m SaveRulesModel) Init() tea.Cmd {
	return tea.Batch(
		m.scanForFilesCmd(),
		m.spinner.Tick, // Start spinner
	)
}

func (m SaveRulesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Log all messages for debugging
	m.logger.LogMessage(msg)

	// Update layout first for size changes
	m.layout, _ = m.layout.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case FileScanCompleteMsg:
		// Handle successful file scan
		m.logger.Debug("Save rules model - File scan completed", "files", msg.Files)
		m.markdownFiles = msg.Files
		m.state = StateFileSelection
		m.err = nil
		return m, nil

	case FileScanErrorMsg:
		// Handle file scan error
		m.logger.Error("Save rules model - File scan failed", "error", msg.Err)
		m.err = msg.Err
		m.state = StateError
		return m, nil

	case spinner.TickMsg:
		if m.state == StateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case StateError:
			if msg.String() == "r" {
				// Retry scanning
				m.state = StateLoading
				m.err = nil
				return m, tea.Batch(
					m.scanForFilesCmd(),
					m.spinner.Tick, // Restart spinner
				)
			}
		case StateFileSelection:
			// Handle file selection logic here
		}
	}

	return m, nil
}

func (m SaveRulesModel) View() string {
	switch m.state {
	case StateLoading:
		return m.viewLoading()
	case StateError:
		return m.viewError()
	case StateFileSelection:
		return m.viewFileSelection()
	default:
		return "Unknown state"
	}
}

func (m SaveRulesModel) viewLoading() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File",
		Subtitle: "Scanning for markdown files...",
		HelpText: "Please wait while we scan your directory",
	})

	content := fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), styles.SpinnerStyle.Render("Scanning..."))

	return m.layout.Render(content)
}

func (m SaveRulesModel) viewError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File - Error",
		Subtitle: "Failed to scan directory",
		HelpText: "Press 'r' to retry • Esc to go back",
	})

	errorText := "Failed to scan directory"
	if m.err != nil {
		errorText = m.err.Error()
	}

	return m.layout.Render(errorText)
}

func (m SaveRulesModel) viewFileSelection() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File",
		Subtitle: "Select a markdown file to save",
		HelpText: "↑/↓ to navigate • Enter to select • Esc to go back",
	})

	if len(m.markdownFiles) == 0 {
		return m.layout.Render("No markdown files found in current directory")
	}

	content := "Found markdown files:\n\n"
	for i, file := range m.markdownFiles {
		content += "  " + file + "\n"
		if i >= 10 { // Limit display
			content += "  ... and more\n"
			break
		}
	}

	return m.layout.Render(content)
}

// COMMANDS

// Command to scan for files asynchronously
func (m SaveRulesModel) scanForFilesCmd() tea.Cmd {
	m.logger.Debug("File scan started")
	return func() tea.Msg {
		files, err := filemanager.ScanCurrDirectory()
		if err != nil || len(files) == 0 {
			return FileScanErrorMsg{Err: err}
		}
		m.logger.Debug("File scan completed", "files", files)
		return FileScanCompleteMsg{Files: files}
	}
}
