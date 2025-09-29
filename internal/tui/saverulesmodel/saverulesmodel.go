package saverulesmodel

import (
	"fmt"
	"strings"

	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/internal/tui/components/filepicker"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/styles"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type SaveFileModelState int

const (
	StateLoading       SaveFileModelState = iota // Scanning filesystem for markdown files
	StateFileSelection                           // Showing file picker with preview
	StateFileNameInput                           // Allowing user to override destination filename
	StateConfirmation                            // Confirming overwrite scenario
	StateSaving                                  // Performing save
	StateSuccess                                 // Save completed
	StateError                                   // Any error state
)

// Custom messages (internal domain-specific) for async operations and transitions.
type (
	FileScanCompleteMsg struct {
		Files []filemanager.FileItem
	}

	FileScanErrorMsg struct {
		Err error
	}

	SaveFileCompleteMsg struct {
		DestPath string
	}

	SaveFileErrorMsg struct {
		Err              error
		IsOverwriteError bool
	}
)

type SaveRulesModel struct {
	logger *logging.AppLogger
	state  SaveFileModelState

	// Layout (used for all states except the file selection which delegates to FilePicker's own layout)
	layout  components.LayoutModel
	spinner spinner.Model

	// Filepicker instance
	filePicker *filepicker.FilePicker

	// Filename input (optional rename)
	nameInput textinput.Model

	// Data
	markdownFiles    []filemanager.FileItem
	selectedFile     filemanager.FileItem
	newFileName      string // user-entered / confirmed destination filename
	destinationPath  string // resulting storage path
	err              error
	isOverwriteError bool

	// FileManager instance
	fileManager *filemanager.FileManager
}

func NewSaveRulesModel(ctx helpers.UIContext) SaveRulesModel {
	layout := components.NewLayout(components.LayoutConfig{
		MarginX:  2,
		MarginY:  1,
		MaxWidth: 100,
	})

	if ctx.HasValidDimensions() {
		windowMsg := tea.WindowSizeMsg{Width: ctx.Width, Height: ctx.Height}
		layout, _ = layout.Update(windowMsg)
	}

	s := spinner.New()
	s.Style = styles.SpinnerStyle
	s.Spinner = spinner.Pulse

	// Filename input configuration
	nameInput := textinput.New()
	nameInput.Placeholder = "Enter new filename (optional)"
	nameInput.CharLimit = 255
	nameInput.Width = 50

	// Prepare repository and get local path
	localPath, syncInfo, err := repository.PrepareRepository(ctx.Config.Central, ctx.Logger)
	if err != nil {
		ctx.Logger.Error("Failed to prepare repository", "error", err)
		return SaveRulesModel{
			logger:           ctx.Logger,
			state:            StateError,
			layout:           layout,
			spinner:          s,
			filePicker:       nil,
			nameInput:        nameInput,
			markdownFiles:    []filemanager.FileItem{},
			selectedFile:     filemanager.FileItem{},
			newFileName:      "",
			destinationPath:  "",
			err:              fmt.Errorf("repository preparation failed: %w", err),
			isOverwriteError: false,
			fileManager:      nil,
		}
	}

	// Surface sync information if available
	if syncInfo.Message != "" {
		ctx.Logger.Info("Repository status", "message", syncInfo.Message)
	}

	fm, err := filemanager.NewFileManager(localPath, ctx.Logger)
	if err != nil {
		ctx.Logger.Error("Failed to initialize FileManager", "error", err)
	}

	return SaveRulesModel{
		logger:           ctx.Logger,
		state:            StateLoading,
		layout:           layout,
		spinner:          s,
		filePicker:       nil, // created after scan
		nameInput:        nameInput,
		markdownFiles:    []filemanager.FileItem{},
		selectedFile:     filemanager.FileItem{},
		newFileName:      "",
		destinationPath:  "",
		err:              nil,
		isOverwriteError: false,
		fileManager:      fm,
	}
}

// Init starts asynchronous scanning for markdown files.
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
		m.spinner.Tick,
	)
}

func (m SaveRulesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	// Update layout
	m.layout, _ = m.layout.Update(msg)

	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		// Propagate window sizing to FilePicker if it exists & we are in selection state.
		if m.filePicker != nil {
			updated, fpCmd := m.filePicker.Update(message)
			if fpCmd != nil {
				cmds = append(cmds, fpCmd)
			}

			if fp, ok := updated.(*filepicker.FilePicker); ok {
				m.filePicker = fp
			}
		}
		return m, tea.Batch(cmds...)

	case FileScanCompleteMsg:
		m.logger.Debug("Save rules model - File scan completed", "files_count", len(message.Files))
		// Convert relative paths from ScanCurrDirectory to absolute paths for FilePicker
		m.markdownFiles = m.fileManager.ConvertToAbsolutePathsCWD(message.Files)
		m.state = StateFileSelection
		m.err = nil

		// Build FilePicker once files are available
		// Use reasonable default dimensions since we don't have actual terminal size yet
		// The FilePicker will be updated with correct dimensions when WindowSizeMsg is received
		ctx := helpers.NewUIContext(100, 30, nil, m.logger)
		fp := filepicker.NewFilePicker(
			"💾 Save Rules File",
			"Select a markdown file to save to your central rules repository (press Enter). \nUse / to filter, arrows to navigate, g to toggle formatting.",
			m.markdownFiles,
			ctx,
		)
		m.filePicker = &fp

		// Initialize FilePicker (schedule initial preview if any)
		fpInit := m.filePicker.Init()
		if fpInit != nil {
			cmds = append(cmds, fpInit)
		}
		return m, tea.Batch(cmds...)

	case FileScanErrorMsg:
		m.logger.Error("Save rules model - File scan failed", "error", message.Err)
		m.err = message.Err
		m.state = StateError
		m.isOverwriteError = false
		return m, nil

	case filepicker.FileSelectedMsg:
		// File chosen in picker; transition to filename entry
		m.logger.Debug("Save rules model - File selected from picker", "path", message.File.Path)
		m.selectedFile = message.File
		m.newFileName = message.File.Name

		// Prepare name change input
		m.nameInput.SetValue(m.newFileName)
		m.nameInput.Focus()
		m.state = StateFileNameInput
		return m, textinput.Blink

	case SaveFileCompleteMsg:
		m.logger.Info("File saved successfully", "dest", message.DestPath)
		m.destinationPath = message.DestPath
		m.state = StateSuccess
		m.err = nil
		return m, nil

	case SaveFileErrorMsg:
		m.logger.Error("File save failed", "error", message.Err, "isOverwrite", message.IsOverwriteError)
		m.err = message.Err
		m.isOverwriteError = message.IsOverwriteError
		// If it's an overwrite error, we need to confirm with the user
		// whether they want to proceed with overwriting the existing file.
		// So we return to the confirmation state.
		if message.IsOverwriteError {
			m.state = StateConfirmation
		} else {
			m.state = StateError
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == StateLoading || m.state == StateSaving {
			m.spinner, cmd = m.spinner.Update(message)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case StateFileSelection:
			// Intercept 'q' and 'esc' to return to main menu instead of quitting
			if message.String() == "q" || message.String() == "esc" {
				return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
			}

			// Delegate everything else to FilePicker
			if m.filePicker != nil {
				updated, fpCmd := m.filePicker.Update(message)
				if fpCmd != nil {
					cmds = append(cmds, fpCmd)
				}
				if fp, ok := updated.(*filepicker.FilePicker); ok {
					m.filePicker = fp
				}
			}
			return m, tea.Batch(cmds...)

		case StateFileNameInput:
			switch message.String() {
			case "enter":
				m.commitOrDefaultFilename()
				m.nameInput.Blur()
				m.state = StateSaving
				newNamePtr := m.optionalNewNamePtr()
				return m, tea.Batch(
					m.saveFileCmd(m.selectedFile.Path, newNamePtr, false),
					m.spinner.Tick,
				)
			case "esc":
				// Return to main menu instead of reverting to selection
				return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
			default:
				m.nameInput, cmd = m.nameInput.Update(message)
				m.newFileName = m.nameInput.Value()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			}

		case StateConfirmation:
			switch message.String() {
			case "y":
				m.state = StateSaving
				newNamePtr := m.optionalNewNamePtr()
				return m, tea.Batch(
					m.saveFileCmd(m.selectedFile.Path, newNamePtr, true),
					m.spinner.Tick,
				)
			case "n":
				m.nameInput.Focus()
				m.state = StateFileNameInput
				m.err = nil
				m.isOverwriteError = false
				return m, textinput.Blink
			case "esc":
				// Return to main menu
				return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
			}

		case StateError:
			switch message.String() {
			case "r":
				if m.isOverwriteError {
					m.nameInput.Focus()
					m.state = StateFileNameInput
					m.err = nil
					m.isOverwriteError = false
					return m, textinput.Blink
				}
				// Re-scan
				m.state = StateLoading
				m.err = nil
				return m, tea.Batch(
					m.scanForFilesCmd(),
					m.spinner.Tick,
				)
			case "esc":
				// Return to main menu
				return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
			}

		case StateSuccess:
			switch message.String() {
			case "m":
				return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
			case "a":
				// Reset only selection-related state; keep loaded file list to avoid re-scan
				m.selectedFile = filemanager.FileItem{}
				m.newFileName = ""
				m.destinationPath = ""
				m.nameInput.SetValue("")
				m.state = StateFileSelection
				return m, nil
			}
		}

	default:
		// Delegate unhandled messages in selection state to FilePicker
		if m.state == StateFileSelection && m.filePicker != nil {
			updated, fpCmd := m.filePicker.Update(msg)
			if fpCmd != nil {
				cmds = append(cmds, fpCmd)
			}
			if fp, ok := updated.(*filepicker.FilePicker); ok {
				m.filePicker = fp
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
		// FilePicker renders its own header/layout styling
		if m.filePicker == nil {
			return m.layout.Render("Initializing file picker...")
		}
		return m.filePicker.View()
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
	}

	return m.viewError()
}

// VIEWS

func (m SaveRulesModel) viewLoading() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File",
		Subtitle: "Scanning for markdown files...",
		HelpText: "Please wait while we scan your directory",
	})
	content := fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), styles.SpinnerStyle.Render("Scanning..."))
	return m.layout.Render(content)
}

func (m SaveRulesModel) viewFileNameInput() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File",
		Subtitle: fmt.Sprintf("Selected: %s", m.selectedFile.Name),
		HelpText: "Enter filename (or keep default) • Enter to continue • Esc to go back",
	})

	content := fmt.Sprintf("File will be saved to: %s\n\n", m.fileManager.GetStorageDir())
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
	content += "Storage directory: " + m.fileManager.GetStorageDir()
	return m.layout.Render(content)
}

func (m SaveRulesModel) viewSaving() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File",
		Subtitle: "Saving file...",
		HelpText: "Please wait while we copy your file",
	})
	content := fmt.Sprintf("Copying '%s' to storage directory...\n\n", m.selectedFile.Name)
	content += fmt.Sprintf("%s %s", m.spinner.View(), styles.SpinnerStyle.Render("Saving..."))
	return m.layout.Render(content)
}

func (m SaveRulesModel) viewSuccess() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File - Success",
		Subtitle: "File saved successfully!",
		HelpText: "m to return to main menu • a to save another file",
	})
	content := "✅ File saved successfully!\n\n"
	content += fmt.Sprintf("Source: %s\n", m.selectedFile.Path)
	content += fmt.Sprintf("Destination: %s\n\n", m.destinationPath)
	content += "The file has been copied to your rules storage directory."
	return m.layout.Render(content)
}

func (m SaveRulesModel) viewError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Save Rules File - Error",
		Subtitle: "Operation failed",
		HelpText: "r to retry • Esc to return to main menu",
	})
	errorText := "An error occurred"
	if m.err != nil {
		errorText = m.err.Error()
	}
	return m.layout.Render(errorText)
}

// HELPERS

// commitOrDefaultFilename ensures m.newFileName is populated (fallback to original selected file name).
func (m *SaveRulesModel) commitOrDefaultFilename() {
	m.newFileName = strings.TrimSpace(m.nameInput.Value())
	if m.newFileName == "" {
		m.newFileName = m.selectedFile.Name
	}
}

// optionalNewNamePtr returns a pointer only if user changed the name (so FileManager can preserve original otherwise).
func (m *SaveRulesModel) optionalNewNamePtr() *string {
	if m.newFileName != "" && m.newFileName != m.selectedFile.Name {
		return &m.newFileName
	}
	return nil
}

// COMMANDS

// scanForFilesCmd asynchronously scans current directory tree for markdown files.
func (m SaveRulesModel) scanForFilesCmd() tea.Cmd {
	m.logger.Debug("File scan started")
	return func() tea.Msg {
		files, err := m.fileManager.ScanCurrDirectory()
		if err != nil {
			return FileScanErrorMsg{Err: err}
		}
		return FileScanCompleteMsg{Files: files}
	}
}

// saveFileCmd copies the selected file into the storage directory (with optional rename + overwrite).
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
			isOverwriteError := strings.Contains(err.Error(), "already exists")
			return SaveFileErrorMsg{
				Err:              err,
				IsOverwriteError: isOverwriteError,
			}
		}
		return SaveFileCompleteMsg{DestPath: destPath}
	}
}
