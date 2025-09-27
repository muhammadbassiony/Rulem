package importrulesmenu

import (
	"fmt"
	"rulem/internal/editors"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/tui/components"
	"rulem/internal/tui/components/filepicker"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Key event constants
const (
	KeyQuit   = "q"
	KeyEscape = "esc"
	KeyEnter  = "enter"
	KeyYes    = "y"
	KeyNo     = "n"
	KeyRetry  = "r"
	KeyMenu   = "m"
	KeyAgain  = "a"
)

type CopyModeOption int

const (
	CopyModeOptionCopy CopyModeOption = iota
	CopyModeOptionLink
)

type CopyMode struct {
	title       string
	description string
	copyMode    CopyModeOption
}

func (c CopyMode) Title() string       { return c.title }
func (c CopyMode) Description() string { return c.description }
func (c CopyMode) FilterValue() string {
	return c.title + " " + c.description
}

type ImportRulesModelState int

const (
	StateLoading             ImportRulesModelState = iota // Scanning storage repo for markdown files
	StateFileSelection                                    // User is selecting files to import
	StateEditorSelection                                  // User is selecting an editor
	StateImportModeSelection                              // User is selecting an import mode
	StateConfirmation                                     // User confirms the import operation
	StateImporting                                        // Files are being imported
	StateSuccess                                          // Import completed successfully
	StateError                                            // Any error state
)

type (
	FileScanCompleteMsg struct {
		Files []filemanager.FileItem
	}

	FileScanErrorMsg struct {
		Err error
	}

	ImportFileCompleteMsg struct {
		DestPath string
	}

	ImportFileErrorMsg struct {
		Err              error
		IsOverwriteError bool
	}
)

type ImportRulesModel struct {
	logger *logging.AppLogger
	state  ImportRulesModelState

	// Layout (used for all states except the file selection which delegates to FilePicker's own layout)
	layout  components.LayoutModel
	spinner spinner.Model

	// Filepicker instance
	filePicker *filepicker.FilePicker

	// Lists
	importModeList     list.Model
	selectedImportMode CopyMode
	editorList         list.Model
	selectedEditor     editors.EditorRuleConfig

	// FileManager instance
	fileManager      *filemanager.FileManager
	ruleFiles        []filemanager.FileItem // List of markdown files found in the storage directory
	selectedFile     filemanager.FileItem
	finalDestPath    string // Final destination path after successful import
	isOverwriteError bool

	err error
}

func NewImportRulesModel(ctx helpers.UIContext) *ImportRulesModel {
	// Initialize editors list (this will be constant)
	editorsSlice := editors.GetAllEditorRuleConfigs()
	editors := make([]list.Item, len(editorsSlice))
	for i, editor := range editorsSlice {
		editors[i] = editor
	}
	editorsList := list.New(editors, list.NewDefaultDelegate(), 0, 0)
	editorsList.Title = "" // We'll use the layout for titles
	editorsList.SetShowTitle(false)
	editorsList.SetShowStatusBar(false)
	editorsList.SetFilteringEnabled(true)
	editorsList.SetShowHelp(false) // We'll use the layout for help

	// Initialize import mode list
	copyModeOptions := []list.Item{
		CopyMode{
			title:       "Copy file",
			description: "Copy the rules file from the central repo to the current working directory. This will create an actual copy that is not synced with the version in the central repo",
			copyMode:    CopyModeOptionCopy,
		},
		CopyMode{
			title:       "Link file",
			description: "Create a symbolic link to the rules file in the central repo. This will keep the file in sync with the version in the central repo.",
			copyMode:    CopyModeOptionLink,
		},
	}
	importModeList := list.New(copyModeOptions, list.NewDefaultDelegate(), 0, 0)
	importModeList.Title = "" // We'll use the layout for titles
	importModeList.SetShowTitle(false)
	importModeList.SetShowStatusBar(false)
	importModeList.SetFilteringEnabled(true)
	importModeList.SetShowHelp(false) // We'll use the layout for help

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

	fm, err := filemanager.NewFileManager(ctx.Config.Central.Path, ctx.Logger)
	if err != nil {
		ctx.Logger.Error("Failed to initialize FileManager", "error", err)
	}

	return &ImportRulesModel{
		logger: ctx.Logger,
		state:  StateLoading,
		// editors:     editors,
		layout:           layout,
		spinner:          s,
		filePicker:       nil, // created after scan
		importModeList:   importModeList,
		editorList:       editorsList,
		fileManager:      fm,
		ruleFiles:        nil, // will be populated after scan
		selectedFile:     filemanager.FileItem{},
		isOverwriteError: false,
		err:              nil,
	}
}

func (m *ImportRulesModel) Init() tea.Cmd {
	if m.fileManager == nil {
		return func() tea.Msg {
			return ImportFileErrorMsg{
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

func (m *ImportRulesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update layout
	m.layout, _ = m.layout.Update(msg)

	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		// Propagate window resizing to lists
		width := m.layout.ContentWidth()
		height := m.layout.ContentHeight()

		m.importModeList.SetSize(width, height)
		m.editorList.SetSize(width, height)

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
		m.logger.Debug("Import rules model - File scan completed", "files_count", len(message.Files))
		// Convert relative paths from ScanCentralRepo to absolute paths for FilePicker
		m.ruleFiles = m.fileManager.ConvertToAbsolutePaths(message.Files)

		m.logger.Debug("Import rules model - converted scanned files paths to absolute")
		m.state = StateFileSelection
		m.err = nil

		// Build FilePicker once files are available
		ctx := helpers.NewUIContext(m.layout.ContentWidth(), m.layout.ContentHeight(), nil, m.logger)
		m.logger.Debug("Import rules model - Creating FilePicker", "content_width", m.layout.ContentWidth(), "content_height", m.layout.ContentHeight())

		// Files now have absolute paths suitable for FilePicker file reading
		fp := filepicker.NewFilePicker(
			"📄  Import rules",
			"Select a rule file to import from your central rules repository (press Enter). \nUse / to filter, arrows to navigate, g to toggle formatting.",
			m.ruleFiles,
			ctx,
		)
		m.filePicker = &fp

		// Initialize FilePicker (schedule initial preview if any)
		fpInit := m.filePicker.Init()
		if fpInit != nil {
			m.logger.Debug("Import rules model - FilePicker init command scheduled")
			cmds = append(cmds, fpInit)
		}
		return m, tea.Batch(cmds...)

	case FileScanErrorMsg:
		m.logger.Error("Import rules model - File scan failed", "error", message.Err)
		m.err = message.Err
		m.state = StateError
		m.isOverwriteError = false
		return m, nil

	case filepicker.FileSelectedMsg:
		m.logger.Debug("Import rules model - File selected from picker", "path", message.File.Path)
		m.selectedFile = message.File
		m.state = StateEditorSelection
		return m, nil

	case ImportFileCompleteMsg:
		m.logger.Info("File imported successfully", "dest", message.DestPath)
		m.finalDestPath = message.DestPath
		m.state = StateSuccess
		m.err = nil
		return m, nil

	case ImportFileErrorMsg:
		m.logger.Error("File import failed", "error", message.Err, "isOverwrite", message.IsOverwriteError)
		m.err = message.Err
		m.isOverwriteError = message.IsOverwriteError

		if message.IsOverwriteError {
			m.state = StateConfirmation
		} else {
			m.state = StateError
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == StateLoading || m.state == StateImporting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(message)
			return m, cmd
		}
		return m, nil

	case list.FilterMatchesMsg:
		switch m.state {
		case StateImportModeSelection:
			m.importModeList, cmd = m.importModeList.Update(message)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case StateEditorSelection:
			m.editorList, cmd = m.editorList.Update(message)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch m.state {
		case StateFileSelection:
			if message.String() == KeyQuit || message.String() == KeyEscape {
				return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
			}

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

		case StateEditorSelection:
			if m.editorList.FilterState() == list.Filtering {
				var cmd tea.Cmd
				m.editorList, cmd = m.editorList.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			}

			switch message.String() {
			case KeyQuit, KeyEscape:
				m.state = StateFileSelection
				return m, nil
			case KeyEnter:
				if !m.validateEditorSelection() {
					return m, nil
				}
				m.state = StateImportModeSelection
				if selectedItem := m.editorList.SelectedItem(); selectedItem != nil {
					if editor, ok := selectedItem.(editors.EditorRuleConfig); ok {
						m.selectedEditor = editor
						m.logger.Debug("Import Rules Menu - Selected editor", "name", m.selectedEditor.Name)
					}
				}
				return m, nil
			}

			m.editorList, cmd = m.editorList.Update(message)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case StateImportModeSelection:
			if m.importModeList.FilterState() == list.Filtering {
				var cmd tea.Cmd
				m.importModeList, cmd = m.importModeList.Update(message)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			}

			switch message.String() {
			case KeyQuit, KeyEscape:
				m.state = StateEditorSelection
				return m, nil
			case KeyEnter:
				if !m.validateImportModeSelection() {
					return m, nil
				}
				m.state = StateConfirmation
				if selectedItem := m.importModeList.SelectedItem(); selectedItem != nil {
					if mode, ok := selectedItem.(CopyMode); ok {
						m.selectedImportMode = mode
						m.logger.Debug("Import Rules Menu - Selected import mode", "mode", m.selectedImportMode.title)
					}
				}
				return m, nil
			}

			var cmd tea.Cmd
			m.importModeList, cmd = m.importModeList.Update(message)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case StateConfirmation:
			switch message.String() {
			case KeyQuit, KeyEscape:
				m.state = StateImportModeSelection
				return m, nil
			case KeyEnter, KeyYes:
				m.state = StateImporting
				m.logger.Debug("Import Rules Menu - Starting import", "file", m.selectedFile.Path, "editor", m.selectedEditor.Name, "mode", m.selectedImportMode.title)
				return m, tea.Batch(
					m.saveFileCmd(m.isOverwriteError),
					m.spinner.Tick,
				)
			case KeyNo:
				m.state = StateImportModeSelection
				return m, nil
			}
			return m, nil

		case StateError:
			switch message.String() {
			case KeyRetry:
				if m.isOverwriteError {
					m.state = StateConfirmation
					m.err = nil
					m.isOverwriteError = false
					return m, nil
				}
				m.state = StateLoading
				m.err = nil
				return m, tea.Batch(
					m.scanForFilesCmd(),
					m.spinner.Tick,
				)
			case KeyEscape:
				return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
			}
			return m, nil

		case StateSuccess:
			switch message.String() {
			case KeyMenu:
				return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
			case KeyAgain:
				m.resetSelectionState()
				m.state = StateFileSelection
				return m, nil
			}
			return m, nil
		}

	default:
		if m.state == StateFileSelection && m.filePicker != nil {
			updated, fpCmd := m.filePicker.Update(msg)
			if fpCmd != nil {
				cmds = append(cmds, fpCmd)
			}
			if fp, ok := updated.(*filepicker.FilePicker); ok {
				m.filePicker = fp
			}
		}
		return m, tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}

// validateEditorSelection ensures an editor is selected before proceeding
func (m *ImportRulesModel) validateEditorSelection() bool {
	selectedItem := m.editorList.SelectedItem()
	if selectedItem == nil {
		m.logger.Warn("No editor selected")
		return false
	}

	if _, ok := selectedItem.(editors.EditorRuleConfig); !ok {
		m.logger.Warn("Invalid editor selection type")
		return false
	}

	return true
}

// validateImportModeSelection ensures an import mode is selected before proceeding
func (m *ImportRulesModel) validateImportModeSelection() bool {
	selectedItem := m.importModeList.SelectedItem()
	if selectedItem == nil {
		m.logger.Warn("No import mode selected")
		return false
	}

	if _, ok := selectedItem.(CopyMode); !ok {
		m.logger.Warn("Invalid import mode selection type")
		return false
	}

	return true
}

// resetSelectionState resets selection-related state for importing another file
func (m *ImportRulesModel) resetSelectionState() {
	m.selectedFile = filemanager.FileItem{}
	m.selectedEditor = editors.EditorRuleConfig{}
	m.selectedImportMode = CopyMode{}
}

func (m *ImportRulesModel) View() string {
	switch m.state {
	case StateLoading:
		return m.viewLoading()
	case StateFileSelection:
		// FilePicker renders its own header/layout styling
		if m.filePicker == nil {
			return m.layout.Render("Initializing file picker...")
		}
		return m.filePicker.View()
	case StateEditorSelection:
		return m.viewEditorSelection()
	case StateImportModeSelection:
		return m.viewImportModeSelection()
	case StateConfirmation:
		return m.viewConfirmation()
	case StateImporting:
		return m.viewImporting()
	case StateSuccess:
		return m.viewSuccess()
	case StateError:
		return m.viewError()
	}

	return m.viewError()
}

// VIEWS

func (m *ImportRulesModel) viewLoading() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📄 Import Rules File",
		Subtitle: "Scanning central repository for rule files...",
		HelpText: "Please wait while we scan your central rules repository",
	})
	content := fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), styles.SpinnerStyle.Render("Scanning..."))
	return m.layout.Render(content)
}

func (m *ImportRulesModel) viewEditorSelection() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📄 Import Rules File",
		Subtitle: fmt.Sprintf("Selected: %s", m.selectedFile.Name),
		HelpText: "Select target editor • Enter to continue • / to filter • q/Esc to go back",
	})

	content := "Choose the editor configuration for this rule file:\n\n"
	content += m.editorList.View()

	return m.layout.Render(content)
}

func (m *ImportRulesModel) viewImportModeSelection() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📄 Import Rules File",
		Subtitle: fmt.Sprintf("File: %s | Editor: %s", m.selectedFile.Name, m.selectedEditor.Name),
		HelpText: "Select import mode • Enter to continue • / to filter • q/Esc to go back",
	})

	content := "Choose how to import the rule file:\n\n"
	content += m.importModeList.View()

	return m.layout.Render(content)
}

func (m *ImportRulesModel) viewConfirmation() string {
	subtitle := "Confirm import operation"
	helpText := "y to proceed • n to go back • Esc to cancel"

	if m.isOverwriteError {
		subtitle = "File already exists - Confirm overwrite"
		helpText = "y to overwrite • n to go back • Esc to cancel"
	}

	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📄 Import Rules File - Confirmation",
		Subtitle: subtitle,
		HelpText: helpText,
	})

	// Generate the destination path using the selected editor config
	destPath := m.selectedEditor.GenerateRuleFileFullPath(m.selectedFile.Name)

	content := fmt.Sprintf("Source File: %s\n", m.selectedFile.Name)
	content += fmt.Sprintf("Destination: %s\n", destPath)
	content += fmt.Sprintf("Editor: %s\n", m.selectedEditor.Name)
	content += fmt.Sprintf("Import Mode: %s\n\n", m.selectedImportMode.title)

	if m.isOverwriteError {
		content += "A file with this name already exists in the current directory.\n\n"
		content += "Do you want to overwrite it?\n"
	} else {
		content += "Proceed with importing this file?\n"
	}

	return m.layout.Render(content)
}

func (m *ImportRulesModel) viewImporting() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📄 Import Rules File",
		Subtitle: "Importing file...",
		HelpText: "Please wait while we import your file",
	})

	actionText := "Copying"
	if m.selectedImportMode.copyMode == CopyModeOptionLink {
		actionText = "Creating symbolic link for"
	}

	content := fmt.Sprintf("%s '%s' to current directory...\n\n", actionText, m.selectedFile.Name)
	content += fmt.Sprintf("%s %s", m.spinner.View(), styles.SpinnerStyle.Render("Importing..."))
	return m.layout.Render(content)
}

func (m *ImportRulesModel) viewSuccess() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📄 Import Rules File - Success",
		Subtitle: "File imported successfully!",
		HelpText: "m to return to main menu • a to import another file",
	})

	actionText := "copied"
	if m.selectedImportMode.copyMode == CopyModeOptionLink {
		actionText = "linked"
	}

	content := "✅ File imported successfully!\n\n"
	content += fmt.Sprintf("Source: %s\n", m.selectedFile.Name)
	content += fmt.Sprintf("Destination: %s\n", m.finalDestPath)
	content += fmt.Sprintf("Editor: %s\n", m.selectedEditor.Name)
	content += fmt.Sprintf("Import Mode: %s\n\n", m.selectedImportMode.title)
	content += fmt.Sprintf("The file has been %s to your current working directory.", actionText)
	return m.layout.Render(content)
}

func (m *ImportRulesModel) viewError() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📄 Import Rules File - Error",
		Subtitle: "Operation failed",
		HelpText: "r to retry • Esc to return to main menu",
	})
	errorText := "An error occurred"
	if m.err != nil {
		errorText = m.err.Error()
	}
	return m.layout.Render(errorText)
}

// COMMANDS

// scanForFilesCmd asynchronously scans central storage repo tree for markdown files.
func (m *ImportRulesModel) scanForFilesCmd() tea.Cmd {
	m.logger.Debug("Import rules - File scan started for central repository")
	return func() tea.Msg {
		files, err := m.fileManager.ScanCentralRepo()
		if err != nil {
			m.logger.Error("Import rules - File scan failed", "error", err)
			return FileScanErrorMsg{Err: err}
		}
		return FileScanCompleteMsg{Files: files}
	}
}

func (m *ImportRulesModel) saveFileCmd(overwrite bool) tea.Cmd {
	m.logger.Debug("Importing file", "path", m.selectedFile.Path, "mode", m.selectedImportMode.title, "editor", m.selectedEditor.Title(), "overwrite", overwrite)
	return func() tea.Msg {
		// Path of the file in storage, now absolute from ScanCentralRepo
		storagePath := m.selectedFile.Path

		// Use the selected editor config to generate the destination file path
		// this will be relative to the CWD.
		destFilePath := m.selectedEditor.GenerateRuleFileFullPath(m.selectedFile.Name)

		var finalDestPath string
		var err error
		switch m.selectedImportMode.copyMode {
		case CopyModeOptionCopy:
			// Copy the file to the current working directory
			m.logger.Debug("Calling CopyFileFromStorage", "storagePath", storagePath, "destFilePath", destFilePath)
			finalDestPath, err = m.fileManager.CopyFileFromStorage(storagePath, destFilePath, overwrite)
			if err != nil {
				m.logger.Error("Failed to copy file from storage", "error", err, "storagePath", storagePath, "destFdestFilePathilename", destFilePath)
				isOverwriteError := strings.Contains(err.Error(), "already exists")
				return ImportFileErrorMsg{Err: err, IsOverwriteError: isOverwriteError}
			}
			m.logger.Info("File copied successfully", "dest", finalDestPath)

		case CopyModeOptionLink:
			// Create a symbolic link to the file in the current working directory
			m.logger.Debug("Calling CreateSymlinkFromStorage", "storagePath", storagePath, "destFilePath", destFilePath)
			finalDestPath, err = m.fileManager.CreateSymlinkFromStorage(storagePath, destFilePath, overwrite)
			if err != nil {
				m.logger.Error("Failed to create symlink from storage", "error", err, "storagePath", storagePath, "destFilePath", destFilePath)
				isOverwriteError := strings.Contains(err.Error(), "already exists")
				return ImportFileErrorMsg{Err: err, IsOverwriteError: isOverwriteError}
			}
			m.logger.Info("Symlink created successfully", "dest", finalDestPath)

		}
		return ImportFileCompleteMsg{DestPath: finalDestPath}
	}
}
