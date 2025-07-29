package saverulesmodel

import (
	"fmt"
	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type SaveFileModelState int

const (
	StateLoading SaveFileModelState = iota
	StateFileSelection
	StateFileNameInput
	StateConfirmation
	StateSaving
	StateSuccess
	StateError
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
		FilePath    string
		NewFileName *string
		Overwrite   bool
	}

	SaveFileCompleteMsg struct {
		DestPath string
	}

	SaveFileErrorMsg struct {
		Err              error
		IsOverwriteError bool
	}
)

// List item for file selection
type fileItem struct {
	filename string
}

func (i fileItem) Title() string       { return i.filename }
func (i fileItem) Description() string { return " " }
func (i fileItem) FilterValue() string { return i.filename }

type SaveRulesModel struct {
	config *config.Config
	logger *logging.AppLogger
	state  SaveFileModelState

	// Layout for consistent UI
	layout  components.LayoutModel
	spinner spinner.Model

	// UI Components
	fileList  list.Model
	nameInput textinput.Model

	// Data
	markdownFiles    []string
	selectedFile     string
	newFileName      string
	destinationPath  string
	err              error
	isOverwriteError bool

	// FileManager instance
	fileManager *filemanager.FileManager
}

func NewSaveRulesModel(cfg *config.Config, logger *logging.AppLogger) SaveRulesModel {
	// Create layout
	layout := components.NewLayout(components.LayoutConfig{
		MarginX:  2,
		MarginY:  1,
		MaxWidth: 0,
	})

	// Initialize spinner
	s := spinner.New()
	s.Style = styles.SpinnerStyle
	s.Spinner = spinner.Pulse

	// Initialize file list
	fileList := list.New([]list.Item{}, list.NewDefaultDelegate(), 100, 20)
	fileList.Title = ""
	fileList.SetShowTitle(false)
	fileList.SetShowStatusBar(false)
	fileList.SetFilteringEnabled(true)
	fileList.SetShowHelp(false)

	// Initialize text input for filename
	nameInput := textinput.New()
	nameInput.Placeholder = "Enter new filename (optional)"
	nameInput.CharLimit = 255
	nameInput.Width = 50

	// Initialize FileManager
	fm, err := filemanager.NewFileManager(cfg.StorageDir, logger)
	if err != nil {
		logger.Error("Failed to initialize FileManager", "error", err)
		// We'll handle this in the first view
	}

	return SaveRulesModel{
		config:           cfg,
		logger:           logger,
		state:            StateLoading,
		layout:           layout,
		spinner:          s,
		fileList:         fileList,
		nameInput:        nameInput,
		markdownFiles:    []string{},
		selectedFile:     "",
		newFileName:      "",
		destinationPath:  "",
		err:              nil,
		isOverwriteError: false,
		fileManager:      fm,
	}
}

func (m SaveRulesModel) Init() tea.Cmd {
	if m.fileManager == nil {
		return func() tea.Msg {
			return SaveFileErrorMsg{
				Err:              fmt.Errorf("FileManager not initialized - invalid storage directory"),
				IsOverwriteError: false,
			}
		}
	}

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

	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update list size
		listWidth := msg.Width - 6   // Account for margins and borders
		listHeight := msg.Height - 8 // Leave room for title, help, etc.
		m.fileList.SetSize(listWidth, listHeight)
		return m, nil

	case FileScanCompleteMsg:
		// Handle successful file scan
		m.logger.Debug("Save rules model - File scan completed", "files", msg.Files)
		m.markdownFiles = msg.Files
		m.state = StateFileSelection
		m.err = nil

		// Populate file list
		items := make([]list.Item, len(msg.Files))
		for i, file := range msg.Files {
			items[i] = fileItem{filename: file}
		}
		m.fileList.SetItems(items)

		// Ensure the list has reasonable dimensions if not set yet
		if m.fileList.Width() == 0 || m.fileList.Height() == 0 {
			// Set default dimensions - these will be updated by WindowSizeMsg later
			m.fileList.SetSize(100, 20)
		}

		// Update the list after setting items
		var listCmd tea.Cmd
		m.fileList, listCmd = m.fileList.Update(msg)

		return m, listCmd

	case FileScanErrorMsg:
		// Handle file scan error
		m.logger.Error("Save rules model - File scan failed", "error", msg.Err)
		m.err = msg.Err
		m.state = StateError
		m.isOverwriteError = false
		return m, nil

	case SaveFileCompleteMsg:
		// Handle successful save
		m.logger.Info("File saved successfully", "dest", msg.DestPath)
		m.destinationPath = msg.DestPath
		m.state = StateSuccess
		m.err = nil
		return m, nil

	case SaveFileErrorMsg:
		// Handle save error
		m.logger.Error("File save failed", "error", msg.Err, "isOverwrite", msg.IsOverwriteError)
		m.err = msg.Err
		m.isOverwriteError = msg.IsOverwriteError
		if msg.IsOverwriteError {
			m.state = StateConfirmation
		} else {
			m.state = StateError
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == StateLoading || m.state == StateSaving {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case StateFileSelection:
			switch msg.String() {
			case "enter":
				// When enter is pressed when not filtering, then select the file
				if m.fileList.FilterState() != list.Filtering {
					if selectedItem, ok := m.fileList.SelectedItem().(fileItem); ok {
						m.selectedFile = selectedItem.filename
						m.logger.Debug("File selected", "file", m.selectedFile)

						// Auto-populate the filename input with the selected filename
						m.newFileName = selectedItem.filename
						m.nameInput.SetValue(selectedItem.filename)
						m.nameInput.Focus()

						m.state = StateFileNameInput
						return m, textinput.Blink
					}
				}

				// When filtering, pass enter to the list
				m.fileList, cmd = m.fileList.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			default:
				// Update the file list for navigation/filtering
				m.fileList, cmd = m.fileList.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case StateFileNameInput:
			switch msg.String() {
			case "enter":
				// Proceed with save operation
				m.newFileName = strings.TrimSpace(m.nameInput.Value())
				if m.newFileName == "" {
					m.newFileName = m.selectedFile // Use original filename if empty
				}

				m.nameInput.Blur()
				m.state = StateSaving

				var newFileNamePtr *string
				if m.newFileName != m.selectedFile {
					newFileNamePtr = &m.newFileName
				}

				return m, tea.Batch(
					m.saveFileCmd(m.selectedFile, newFileNamePtr, false), // First try without overwrite
					m.spinner.Tick,
				)
			case "esc":
				// Go back to file selection
				m.nameInput.Blur()
				m.state = StateFileSelection
				return m, nil
			default:
				// Update text input
				m.nameInput, cmd = m.nameInput.Update(msg)
				m.newFileName = m.nameInput.Value()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case StateConfirmation:
			switch msg.String() {
			case "y":
				// User confirmed overwrite
				m.state = StateSaving

				var newFileNamePtr *string
				if m.newFileName != m.selectedFile {
					newFileNamePtr = &m.newFileName
				}

				return m, tea.Batch(
					m.saveFileCmd(m.selectedFile, newFileNamePtr, true), // Try with overwrite
					m.spinner.Tick,
				)
			case "n", "esc":
				// User cancelled overwrite, go back to filename input
				m.nameInput.Focus()
				m.state = StateFileNameInput
				m.err = nil
				m.isOverwriteError = false
				return m, textinput.Blink
			}

		case StateError:
			switch msg.String() {
			case "r":
				if m.isOverwriteError {
					// Retry with different filename
					m.nameInput.Focus()
					m.state = StateFileNameInput
					m.err = nil
					m.isOverwriteError = false
					return m, textinput.Blink
				} else {
					// Retry scanning
					m.state = StateLoading
					m.err = nil
					return m, tea.Batch(
						m.scanForFilesCmd(),
						m.spinner.Tick,
					)
				}
			}

		case StateSuccess:
			switch msg.String() {
			case "m":
				// Return to main menu - use command approach
				return m, func() tea.Msg {
					return helpers.NavigateToMainMenuMsg{}
				}
			case "a":
				// Save another file - go back to file selection
				m.selectedFile = ""
				m.newFileName = ""
				m.destinationPath = ""
				m.nameInput.SetValue("")
				m.state = StateFileSelection
				return m, nil
			}
		}

	default:
		// Handle other message types
		switch m.state {
		case StateFileSelection:
			m.fileList, cmd = m.fileList.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case StateFileNameInput:
			m.nameInput, cmd = m.nameInput.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m SaveRulesModel) View() string {
	switch m.state {
	case StateLoading:
		return m.viewLoading()
	case StateFileSelection:
		return m.viewFileSelection()
	case StateFileNameInput:
		return m.viewFileNameInput()
	case StateConfirmation:
		return m.viewConfirmation()
	case StateSaving:
		return m.viewSaving()
	case StateSuccess:
		return m.viewSuccess()
	case StateError:
		return m.viewError()
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

func (m SaveRulesModel) viewFileSelection() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File",
		Subtitle: "Select a markdown file to save",
		HelpText: "↑/↓ to navigate • Enter to select • / to filter • Esc to go back",
	})

	if len(m.markdownFiles) == 0 {
		return m.layout.Render("No markdown files found in current directory")
	}

	return m.layout.Render(m.fileList.View())
}

func (m SaveRulesModel) viewFileNameInput() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File",
		Subtitle: fmt.Sprintf("Selected: %s", m.selectedFile),
		HelpText: "Enter filename (or keep default) • Enter to continue • Esc to go back",
	})

	content := fmt.Sprintf("File will be saved to: %s\n\n", m.config.StorageDir)
	content += "Filename:\n"
	content += m.nameInput.View()
	content += "\n\n"
	content += "Preview: " + m.nameInput.Value()

	return m.layout.Render(content)
}

func (m SaveRulesModel) viewConfirmation() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File - Confirm Overwrite",
		Subtitle: "File already exists",
		HelpText: "y to overwrite • n to change filename • Esc to cancel",
	})

	content := fmt.Sprintf("A file named '%s' already exists in the storage directory.\n\n", m.newFileName)
	content += "Do you want to overwrite it?\n\n"
	content += "Storage directory: " + m.config.StorageDir

	return m.layout.Render(content)
}

func (m SaveRulesModel) viewSaving() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File",
		Subtitle: "Saving file...",
		HelpText: "Please wait while we copy your file",
	})

	content := fmt.Sprintf("Copying '%s' to storage directory...\n\n", m.selectedFile)
	content += fmt.Sprintf("%s %s", m.spinner.View(), styles.SpinnerStyle.Render("Saving..."))

	return m.layout.Render(content)
}

func (m SaveRulesModel) viewSuccess() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File - Success",
		Subtitle: "File saved successfully!",
		HelpText: "m to return to main menu • a to save another file • Esc to go back",
	})

	content := "✅ File saved successfully!\n\n"
	content += fmt.Sprintf("Source: %s\n", m.selectedFile)
	content += fmt.Sprintf("Destination: %s\n\n", m.destinationPath)
	content += "The file has been copied to your rules storage directory."

	return m.layout.Render(content)
}

func (m SaveRulesModel) viewError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File - Error",
		Subtitle: "Operation failed",
		HelpText: "r to retry • Esc to go back",
	})

	errorText := "An error occurred"
	if m.err != nil {
		errorText = m.err.Error()
	}

	return m.layout.Render(errorText)
}

// COMMANDS

// Command to scan for files asynchronously
func (m SaveRulesModel) scanForFilesCmd() tea.Cmd {
	m.logger.Debug("File scan started")
	return func() tea.Msg {
		files, err := filemanager.ScanCurrDirectory()
		if err != nil {
			return FileScanErrorMsg{Err: err}
		}
		if len(files) == 0 {
			return FileScanErrorMsg{Err: fmt.Errorf("no markdown files found in current directory")}
		}
		m.logger.Debug("File scan completed", "files", files)
		return FileScanCompleteMsg{Files: files}
	}
}

// Command to save file asynchronously
func (m SaveRulesModel) saveFileCmd(filePath string, newFileName *string, overwrite bool) tea.Cmd {
	m.logger.Debug("Starting file save operation", "file", filePath, "newName", newFileName, "overwrite", overwrite)
	return func() tea.Msg {
		if m.fileManager == nil {
			return SaveFileErrorMsg{
				Err:              fmt.Errorf("FileManager not initialized"),
				IsOverwriteError: false,
			}
		}

		destPath, err := m.fileManager.CopyFileToStorage(filePath, newFileName, overwrite)
		if err != nil {
			// Check if this is an overwrite error
			isOverwriteError := strings.Contains(err.Error(), "already exists")
			return SaveFileErrorMsg{
				Err:              err,
				IsOverwriteError: isOverwriteError,
			}
		}

		return SaveFileCompleteMsg{DestPath: destPath}
	}
}
