// Package setupmenu provides the first-time setup flow for the rulem TUI application.
//
// This package implements a multi-step wizard that guides users through configuring
// their central rules repository. It supports both local directory storage and
// GitHub-backed repositories with secure Personal Access Token management.
//
// The setup flow consists of several states:
//   - Welcome: Introduction and overview
//   - Repository Type Selection: Choose between local directory or GitHub repository
//   - Local Flow: Storage directory configuration
//   - GitHub Flow: URL → Branch → Clone Path → PAT authentication
//   - Confirmation: Review and confirm settings
//   - Complete/Cancelled: Final state
//
// Key features:
//   - Reference-based model functions for state management
//   - Secure PAT storage via OS keyring (never stored in plain text)
//   - Comprehensive validation at each step with helpful error messages
//   - Back navigation support with Escape key
//   - Responsive layout using centralized components
package setupmenu

import (
	"context"
	"fmt"
	"path/filepath"
	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/components"
	"rulem/internal/tui/helpers"
	"rulem/internal/tui/styles"
	"rulem/pkg/fileops"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SetupState represents the current state of the setup process
type SetupState int

const (
	SetupStateWelcome        SetupState = iota // Initial welcome screen
	SetupStateRepositoryType                   // Choose local vs GitHub repository
	SetupStateStorageInput                     // Local directory path input
	SetupStateGitHubURL                        // GitHub repository URL input
	SetupStateGitHubBranch                     // Branch name input (optional)
	SetupStateGitHubPath                       // Local clone path input
	SetupStateGitHubPAT                        // Personal Access Token input (password-masked)
	SetupStateConfirmation                     // Review and confirm configuration
	SetupStateComplete                         // Setup successfully completed
	SetupStateCancelled                        // Setup was cancelled by user
)

// RepositoryType indicates whether the user chose local directory or GitHub repository storage.
type RepositoryType int

const (
	RepositoryTypeLocal  RepositoryType = iota // Use a local directory for rules storage
	RepositoryTypeGitHub                       // Use a GitHub repository (cloned locally)
)

// Custom messages for internal state transitions
type (
	setupErrorMsg    struct{ err error }
	setupCompleteMsg struct{}
)

// SetupModel manages the first-time setup wizard state and user interactions.
// It implements the tea.Model interface for Bubble Tea TUI framework.
//
// The model uses reference-based methods (pointer receivers) throughout to ensure
// state changes are properly propagated across the event loop.
type SetupModel struct {
	// Current state in the setup wizard flow
	state SetupState

	// Repository configuration
	repositoryType      RepositoryType // Local or GitHub
	repositoryTypeIndex int            // Selected index in repository type menu (0=Local, 1=GitHub)

	// Storage paths and repository details
	StorageDir   string // Path for local directory storage
	GitHubURL    string // GitHub repository URL (HTTPS or SSH format)
	GitHubBranch string // Branch name (empty = use default branch)
	GitHubPath   string // Local path where GitHub repo will be cloned
	GitHubPAT    string // Personal Access Token (stored in memory until final confirmation)

	// Flow control
	Cancelled bool               // True if user cancelled setup
	logger    *logging.AppLogger // Structured logging

	// Credential management
	credManager *repository.CredentialManager // Manages secure token storage

	// UI components
	textInput textinput.Model        // Reused text input for all input screens
	layout    components.LayoutModel // Centralized layout and styling
}

// NewSetupModel creates a new setup wizard model with initial state and UI components.
//
// The text input is configured with appropriate defaults and sized responsively
// based on the provided UI context dimensions.
func NewSetupModel(ctx helpers.UIContext) *SetupModel {
	ti := textinput.New()
	ti.Placeholder = repository.GetDefaultStorageDir()
	ti.Focus()
	ti.CharLimit = 256

	// Create centralized layout - will be reconfigured per state
	layout := components.NewLayout(components.LayoutConfig{
		MarginX:  2,
		MarginY:  1,
		MaxWidth: 100,
	})

	// Apply window sizing if available
	if ctx.HasValidDimensions() {
		windowMsg := tea.WindowSizeMsg{Width: ctx.Width, Height: ctx.Height}
		layout, _ = layout.Update(windowMsg)
		ti.Width = layout.InputWidth()
	}

	return &SetupModel{
		state:       SetupStateWelcome,
		textInput:   ti,
		layout:      layout,
		logger:      ctx.Logger,
		credManager: repository.NewCredentialManager(),
	}
}

// Init initializes the setup model when it's first started.
// Returns a command to start the text input cursor blinking.
func (m *SetupModel) Init() tea.Cmd {
	m.logger.Info("Setup model initialized")
	return textinput.Blink
}

// Update handles all incoming messages and delegates to appropriate state-specific handlers.
// This is the main event loop entry point for the Bubble Tea framework.
func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Log all messages for debugging
	m.logger.LogMessage(msg)

	// Update layout with window size changes
	m.layout, _ = m.layout.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update layout and text input width responsively
		m.layout, _ = m.layout.Update(msg)
		m.textInput.Width = m.layout.InputWidth()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case setupErrorMsg:
		m.layout = m.layout.SetError(msg.err)
		return m, nil

	case setupCompleteMsg:
		m.state = SetupStateComplete
		m.layout = m.layout.ClearError()
		return m, nil
	}

	return m, cmd
}

// updateTextInput updates the text input component and clears any displayed errors.
// This is called for all keyboard input that modifies the text field.
func (m *SetupModel) updateTextInput(msg tea.Msg) (*SetupModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	// Clear error on input change
	if m.layout.GetError() != nil {
		m.layout = m.layout.ClearError()
	}
	return m, cmd
}

// handleKeyPress routes key press events to the appropriate state-specific handler.
// Global quit keys (Ctrl+C, q) are checked first, then state-specific handlers are invoked.
func (m *SetupModel) handleKeyPress(msg tea.KeyMsg) (*SetupModel, tea.Cmd) {
	// Global quit commands
	if m.isQuitKey(msg) {
		return m.handleQuit()
	}

	switch m.state {
	case SetupStateWelcome:
		return m.handleWelcomeKeys(msg)
	case SetupStateRepositoryType:
		return m.handleRepositoryTypeKeys(msg)
	case SetupStateStorageInput:
		return m.handleStorageInputKeys(msg)
	case SetupStateGitHubURL:
		return m.handleGitHubURLKeys(msg)
	case SetupStateGitHubBranch:
		return m.handleGitHubBranchKeys(msg)
	case SetupStateGitHubPath:
		return m.handleGitHubPathKeys(msg)
	case SetupStateGitHubPAT:
		return m.handleGitHubPATKeys(msg)
	case SetupStateConfirmation:
		return m.handleConfirmationKeys(msg)
	default:
		return m, tea.Quit
	}
}

// State-specific key handlers
// Each handler manages keyboard input for its respective setup state.

// handleWelcomeKeys handles input on the welcome screen.
// Enter/Space: proceed to repository type selection
// Esc/q: quit setup
func (m *SetupModel) handleWelcomeKeys(msg tea.KeyMsg) (*SetupModel, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		m.state = SetupStateRepositoryType
		m.repositoryTypeIndex = 0 // Default to Local Directory
		m.layout = m.layout.ClearError()
		return m, nil
	case "esc", "q":
		return m.handleQuit()
	}
	return m, nil
}

// handleRepositoryTypeKeys handles input on the repository type selection screen.
// Up/Down/j/k: navigate between Local and GitHub options
// Enter/Space: select current option and proceed
// Esc/q: cancel setup
func (m *SetupModel) handleRepositoryTypeKeys(msg tea.KeyMsg) (*SetupModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.repositoryTypeIndex > 0 {
			m.repositoryTypeIndex--
		}
	case "down", "j":
		if m.repositoryTypeIndex < 1 {
			m.repositoryTypeIndex++
		}
	case "enter", " ":
		m.repositoryType = RepositoryType(m.repositoryTypeIndex)
		if m.repositoryType == RepositoryTypeLocal {
			defaultPath := repository.GetDefaultStorageDir()
			return m, m.resetTextInputForState(SetupStateStorageInput, defaultPath, defaultPath, textinput.EchoNormal)
		} else {
			return m, m.resetTextInputForState(SetupStateGitHubURL, "", "https://github.com/user/repo.git", textinput.EchoNormal)
		}
	case "esc", "q":
		return m.handleQuit()
	}
	return m, nil
}

// handleStorageInputKeys handles input on the local storage directory screen.
// Enter: validate path and proceed to confirmation
// Esc: go back to repository type selection
// Other keys: update text input
func (m *SetupModel) handleStorageInputKeys(msg tea.KeyMsg) (*SetupModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("storage_input_submit", m.textInput.Value())

		// Validate local storage path
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.Debug("Validating storage directory", "path", input)

		if err := fileops.ValidateStoragePath(input); err != nil {
			m.logger.Warn("Storage directory validation failed", "error", err)
			return m, func() tea.Msg { return setupErrorMsg{err} }
		}

		m.StorageDir = fileops.ExpandPath(input)
		m.logger.LogStateTransition("SetupModel", "SetupStateStorageInput", "SetupStateConfirmation")
		m.state = SetupStateConfirmation
		m.layout = m.layout.ClearError()
		return m, nil

	case "esc":
		m.state = SetupStateRepositoryType
		m.repositoryTypeIndex = 0 // Default to Local Directory
		m.layout = m.layout.ClearError()
		return m, nil
	default:
		return m.updateTextInput(msg)
	}
}

// handleGitHubURLKeys handles input on the GitHub repository URL screen.
// Enter: validate URL format and proceed to branch input
// Esc: go back to repository type selection
// Other keys: update text input
func (m *SetupModel) handleGitHubURLKeys(msg tea.KeyMsg) (*SetupModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("github_url_submit", m.textInput.Value())

		// Validate GitHub URL
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.Debug("Validating GitHub URL", "url", input)

		if input == "" {
			return m, func() tea.Msg { return setupErrorMsg{fmt.Errorf("repository URL cannot be empty")} }
		}

		// Basic URL format validation
		if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") && !strings.HasPrefix(input, "git@") {
			return m, func() tea.Msg {
				return setupErrorMsg{fmt.Errorf("invalid repository URL format - must start with https://, http://, or git@")}
			}
		}

		m.GitHubURL = input
		return m, m.resetTextInputForState(SetupStateGitHubBranch, "", "main (leave empty for default)", textinput.EchoNormal)

	case "esc":
		m.state = SetupStateRepositoryType
		m.repositoryTypeIndex = 0 // Default to Local Directory
		m.layout = m.layout.ClearError()
		return m, nil
	default:
		return m.updateTextInput(msg)
	}
}

// handleGitHubBranchKeys handles input on the branch name screen.
// Enter: accept branch (empty = use default) and proceed to clone path
// Esc: go back to URL input
// Other keys: update text input
func (m *SetupModel) handleGitHubBranchKeys(msg tea.KeyMsg) (*SetupModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.GitHubBranch = strings.TrimSpace(m.textInput.Value())
		m.logger.LogUserAction("github_branch_submit", m.GitHubBranch)
		return m, m.resetTextInputForState(SetupStateGitHubPath, "", m.deriveClonePathPlaceholder(), textinput.EchoNormal)
	case "esc":
		return m, m.resetTextInputForState(SetupStateGitHubURL, "", "https://github.com/user/repo.git", textinput.EchoNormal)
	default:
		return m.updateTextInput(msg)
	}
}

// handleGitHubPathKeys handles input on the local clone path screen.
// Enter: validate path and proceed to PAT input
// Esc: go back to branch input
// Other keys: update text input
func (m *SetupModel) handleGitHubPathKeys(msg tea.KeyMsg) (*SetupModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("github_path_submit", m.textInput.Value())

		// Validate clone path
		input := strings.TrimSpace(m.textInput.Value())
		m.logger.Debug("Validating GitHub clone path", "path", input)

		if err := fileops.ValidateStoragePath(input); err != nil {
			m.logger.Warn("GitHub clone path validation failed", "error", err)
			return m, func() tea.Msg { return setupErrorMsg{err} }
		}

		m.GitHubPath = fileops.ExpandPath(input)
		return m, m.resetTextInputForState(SetupStateGitHubPAT, "", "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", textinput.EchoPassword)

	case "esc":
		return m, m.resetTextInputForState(SetupStateGitHubBranch, "", "main (leave empty for default)", textinput.EchoNormal)
	default:
		return m.updateTextInput(msg)
	}
}

// handleGitHubPATKeys handles input on the Personal Access Token screen.
// Enter: validate PAT format, store in OS keyring, and proceed to confirmation
// Esc: go back to clone path input
// Other keys: update text input (displayed as asterisks via EchoPassword mode)
func (m *SetupModel) handleGitHubPATKeys(msg tea.KeyMsg) (*SetupModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.logger.LogUserAction("github_pat_submit", "[REDACTED]")

		input := strings.TrimSpace(m.textInput.Value())
		m.logger.Debug("Validating GitHub PAT")

		if input == "" {
			return m, func() tea.Msg { return setupErrorMsg{fmt.Errorf("Personal Access Token cannot be empty")} }
		}

		// Validate token format
		if err := m.credManager.ValidateGitHubToken(input); err != nil {
			m.logger.Warn("GitHub PAT format validation failed", "error", err)
			return m, func() tea.Msg { return setupErrorMsg{err} }
		}

		// Validate token works with repository using go-git
		m.logger.Debug("Validating GitHub PAT with repository", "repo_url", m.GitHubURL)
		ctx := context.Background()
		if err := m.credManager.ValidateGitHubTokenWithRepo(ctx, input, m.GitHubURL); err != nil {
			m.logger.Warn("GitHub PAT repository validation failed", "error", err)
			return m, func() tea.Msg { return setupErrorMsg{err} }
		}

		m.logger.Debug("GitHub PAT validated successfully")

		// Store token in memory (will be saved to keyring only on final confirmation)
		m.GitHubPAT = input
		m.logger.LogStateTransition("SetupModel", "SetupStateGitHubPAT", "SetupStateConfirmation")
		m.state = SetupStateConfirmation
		m.layout = m.layout.ClearError()
		return m, nil

	case "esc":
		return m, m.resetTextInputForState(SetupStateGitHubPath, "", m.deriveClonePathPlaceholder(), textinput.EchoNormal)
	default:
		return m.updateTextInput(msg)
	}
}

// handleConfirmationKeys handles input on the confirmation screen.
// y/Y/Enter: accept configuration and create config file
// n/N: go back to previous input screen (storage or PAT depending on type)
// q: cancel setup
// Esc: go back to previous input screen
func (m *SetupModel) handleConfirmationKeys(msg tea.KeyMsg) (*SetupModel, tea.Cmd) {
	switch msg.String() {
	case "n", "N":
		m.logger.LogUserAction("confirmation_reject", "going back")
		if m.repositoryType == RepositoryTypeLocal {
			defaultPath := repository.GetDefaultStorageDir()
			return m, m.resetTextInputForState(SetupStateStorageInput, defaultPath, defaultPath, textinput.EchoNormal)
		} else {
			return m, m.resetTextInputForState(SetupStateGitHubPAT, "", "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", textinput.EchoPassword)
		}
	case "y", "Y", "enter":
		m.logger.LogUserAction("confirmation_accept", "creating config")
		return m, m.createConfig()
	case "q":
		return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
	case "esc":
		// Go back to previous screen based on repository type
		if m.repositoryType == RepositoryTypeLocal {
			defaultPath := repository.GetDefaultStorageDir()
			return m, m.resetTextInputForState(SetupStateStorageInput, defaultPath, defaultPath, textinput.EchoNormal)
		} else {
			return m, m.resetTextInputForState(SetupStateGitHubPAT, "", "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", textinput.EchoPassword)
		}
	}
	return m, nil
}

// createConfig returns a Bubble Tea command that creates the configuration file.
// This runs asynchronously to avoid blocking the UI during file operations.
func (m *SetupModel) createConfig() tea.Cmd {
	return func() tea.Msg {
		m.logger.Info("Creating configuration", "storage_dir", m.StorageDir)
		if err := m.performConfigCreation(); err != nil {
			m.logger.Error("Configuration creation failed", "error", err)
			return setupErrorMsg{err}
		}
		m.logger.Info("Configuration created successfully")
		return setupCompleteMsg{}
	}
}

// handleQuit marks the setup as cancelled and navigates to the main menu.
func (m *SetupModel) handleQuit() (*SetupModel, tea.Cmd) {
	m.logger.Warn("Setup cancelled by user")
	m.Cancelled = true
	m.state = SetupStateCancelled
	return m, func() tea.Msg { return helpers.NavigateToMainMenuMsg{} }
}

// isQuitKey checks if the pressed key should trigger a quit action.
// Note: Escape is not included here as it's used for back navigation in most states.
func (m *SetupModel) isQuitKey(msg tea.KeyMsg) bool {
	return msg.String() == "ctrl+c" || msg.String() == "q"
}

// performConfigCreation creates and saves the configuration based on user choices.
// For local repositories, it creates the directory and saves the config.
// For GitHub repositories, it saves the config with remote settings and relies on
// the credential manager to have already stored the PAT during validation.
func (m *SetupModel) performConfigCreation() error {
	cfg := config.DefaultConfig()

	if m.repositoryType == RepositoryTypeLocal {
		// Local repository setup
		cfg.Central.Path = m.StorageDir
		cfg.Central.RemoteURL = nil
		cfg.Central.Branch = nil

		if err := config.UpdateCentralPath(&cfg, m.StorageDir); err != nil {
			return fmt.Errorf("failed to create local repository configuration: %w", err)
		}
	} else {
		// GitHub repository setup
		cfg.Central.Path = m.GitHubPath
		cfg.Central.RemoteURL = &m.GitHubURL

		if m.GitHubBranch != "" {
			cfg.Central.Branch = &m.GitHubBranch
		}

		// Store the PAT in OS keyring (only done at final confirmation)
		if m.GitHubPAT != "" {
			m.logger.Debug("Storing GitHub PAT in keyring")
			if err := m.credManager.StoreGitHubToken(m.GitHubPAT); err != nil {
				return fmt.Errorf("failed to store GitHub token: %w", err)
			}
		}

		// Save the config
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save GitHub repository configuration: %w", err)
		}
	}

	return nil
}

// View renders the appropriate screen based on the current setup state.
// This is the main rendering function for the Bubble Tea framework.
func (m *SetupModel) View() string {
	switch m.state {
	case SetupStateWelcome:
		return m.viewWelcome()
	case SetupStateRepositoryType:
		return m.viewRepositoryType()
	case SetupStateStorageInput:
		return m.viewStorageInput()
	case SetupStateGitHubURL:
		return m.viewGitHubURL()
	case SetupStateGitHubBranch:
		return m.viewGitHubBranch()
	case SetupStateGitHubPath:
		return m.viewGitHubPath()
	case SetupStateGitHubPAT:
		return m.viewGitHubPAT()
	case SetupStateConfirmation:
		return m.viewConfirmation()
	case SetupStateComplete:
		return m.viewComplete()
	case SetupStateCancelled:
		return m.viewCancelled()
	}
	return ""
}

// View rendering functions
// Each function renders the UI for its respective setup state using the centralized layout.

// viewWelcome renders the welcome/introduction screen.
func (m *SetupModel) viewWelcome() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔧 Welcome to Rulem!",
		Subtitle: "Let's set up your configuration.",
		HelpText: "Press Enter to continue, or Esc to cancel",
	})

	content := `This is your first time running Rulem. We need to configure a few settings to get you started.

We'll need to set up:
• Repository type (local directory or GitHub repository)
• Storage location for your migration rules

Your rules will be stored in a central location that can be either a local directory or synchronized with a GitHub repository for team collaboration.`

	return m.layout.Render(content)
}

// viewRepositoryType renders the repository type selection screen with two options:
// Local Directory and GitHub Repository. Visual indicators show the selected option.
func (m *SetupModel) viewRepositoryType() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📁 Repository Type",
		Subtitle: "How would you like to store your migration rules?",
		HelpText: "Use ↑/↓ to select • Press Enter to continue • Esc to cancel",
	})

	content := `Choose how you want to store and manage your migration rules:

`

	// Local Directory option
	localIndicator := "  "
	if m.repositoryTypeIndex == 0 {
		localIndicator = "▶ "
	}
	content += fmt.Sprintf("%s📁 Local Directory\n", localIndicator)
	content += "     Store rules in a local directory on this machine\n"
	content += "     Perfect for personal use or single-machine setups\n\n"

	// GitHub Repository option
	githubIndicator := "  "
	if m.repositoryTypeIndex == 1 {
		githubIndicator = "▶ "
	}
	content += fmt.Sprintf("%s🐙 GitHub Repository\n", githubIndicator)
	content += "     Sync rules with a GitHub repository for team collaboration\n"
	content += "     Enables sharing and version control across multiple machines\n"
	content += "     Requires a GitHub Personal Access Token for authentication"

	return m.layout.Render(content)
}

// viewStorageInput renders the local storage directory input screen.
func (m *SetupModel) viewStorageInput() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "📁 Local Storage Directory",
		Subtitle: "Where should we store your migration rules locally?",
		HelpText: "Press Enter to continue • Esc to go back • Use ~ for home directory",
	})

	explanation := `This directory will be used as a central location to save and organize your migration rules and configurations. Choose a path that is accessible and writable.`

	prompt := "Enter storage directory path:"
	input := styles.InputStyle.Render(m.textInput.View())

	content := fmt.Sprintf("%s\n\n%s\n%s", explanation, prompt, input)

	return m.layout.Render(content)
}

// viewGitHubURL renders the GitHub repository URL input screen.
// Includes security note about PAT storage in OS keyring.
func (m *SetupModel) viewGitHubURL() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🐙 GitHub Repository URL",
		Subtitle: "Enter the URL of your GitHub repository",
		HelpText: "Press Enter to continue • Esc to go back",
	})

	explanation := `Enter the HTTPS or SSH URL of your GitHub repository where your migration rules are stored.

🔒 Security Note: Your Personal Access Token will be securely stored in your OS keyring and never stored in plain text.

Examples:
• https://github.com/username/rules.git
• git@github.com:username/rules.git`

	prompt := "Repository URL:"
	input := styles.InputStyle.Render(m.textInput.View())

	content := fmt.Sprintf("%s\n\n%s\n%s", explanation, prompt, input)

	return m.layout.Render(content)
}

// viewGitHubBranch renders the branch name input screen.
// Users can leave this empty to use the repository's default branch.
func (m *SetupModel) viewGitHubBranch() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🌿 Branch",
		Subtitle: "Which branch should we use? (optional)",
		HelpText: "Press Enter to continue • Esc to go back • Leave empty for default branch",
	})

	explanation := `Specify the branch to use for your rules. If left empty, the repository's default branch will be used automatically.

Common branch names: main, master, develop`

	prompt := "Branch name (optional):"
	input := styles.InputStyle.Render(m.textInput.View())

	content := fmt.Sprintf("%s\n\n%s\n%s", explanation, prompt, input)

	return m.layout.Render(content)
}

// viewGitHubPath renders the local clone path input screen.
func (m *SetupModel) viewGitHubPath() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "💾 Clone Path",
		Subtitle: "Where should we clone the repository locally?",
		HelpText: "Press Enter to continue • Esc to go back • Use ~ for home directory",
	})

	explanation := `This is where the GitHub repository will be cloned and cached locally. The path should be accessible and writable.

The repository will be kept in sync automatically, and you can work with the files as if they were local.`

	prompt := "Local clone path:"
	input := styles.InputStyle.Render(m.textInput.View())

	content := fmt.Sprintf("%s\n\n%s\n%s", explanation, prompt, input)

	return m.layout.Render(content)
}

// viewGitHubPAT renders the Personal Access Token input screen.
// The text input is in EchoPassword mode, displaying asterisks instead of the actual token.
// Includes comprehensive security messaging about OS keyring storage.
func (m *SetupModel) viewGitHubPAT() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🔑 GitHub Personal Access Token",
		Subtitle: "Enter your GitHub Personal Access Token",
		HelpText: "Press Enter to continue • Esc to go back",
	})

	explanation := `A Personal Access Token (PAT) is required to access your GitHub repository.

🔒 Security Assurance:
• Your token is securely stored in your OS keyring (Keychain/Credential Manager)
• Never stored in plain text files or configuration
• Can be updated or removed anytime in Settings

Required scopes: 'repo' (for private repos) or 'public_repo' (for public repos)

Generate a token at: https://github.com/settings/tokens`

	prompt := "Personal Access Token:"
	input := styles.InputStyle.Render(m.textInput.View())

	content := fmt.Sprintf("%s\n\n%s\n%s", explanation, prompt, input)

	return m.layout.Render(content)
}

// viewConfirmation renders the configuration review and confirmation screen.
// Displays different information based on whether local or GitHub repository was chosen.
func (m *SetupModel) viewConfirmation() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "✅ Confirm Configuration",
		Subtitle: "Please review your settings:",
		HelpText: "Press y to confirm • n to go back • q/Esc to cancel",
	})

	var settings string
	if m.repositoryType == RepositoryTypeLocal {
		settings = fmt.Sprintf(`Repository Type: Local Directory
Storage Directory: %s`, m.StorageDir)
	} else {
		branch := m.GitHubBranch
		if branch == "" {
			branch = "(default branch)"
		}
		settings = fmt.Sprintf(`Repository Type: GitHub Repository
Repository URL: %s
Branch: %s
Local Clone Path: %s
Personal Access Token: [Securely stored in OS keyring]`, m.GitHubURL, branch, m.GitHubPath)
	}

	prompt := "Is this correct? (Y/n)"
	content := fmt.Sprintf("%s\n\n%s", settings, prompt)

	return m.layout.Render(content)
}

// viewComplete renders the success screen after configuration is created.
// Displays different information based on the repository type chosen.
func (m *SetupModel) viewComplete() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "🎉 Setup Complete!",
		Subtitle: "Rulem has been configured successfully.",
		HelpText: "Press any key to continue",
	})

	var content string
	if m.repositoryType == RepositoryTypeLocal {
		content = fmt.Sprintf(`Configuration saved successfully!

Repository Type: Local Directory
Storage Directory: %s

You can now start using rulem to manage your migration rules. The application will store all your rules and configurations in the directory you specified.`, m.StorageDir)
	} else {
		branch := m.GitHubBranch
		if branch == "" {
			branch = "(default branch)"
		}
		content = fmt.Sprintf(`Configuration saved successfully!

Repository Type: GitHub Repository
Repository URL: %s
Branch: %s
Local Clone Path: %s

🔒 Your Personal Access Token has been securely stored in your OS keyring.

You can now start using rulem to manage your migration rules. The repository will be automatically synchronized, and you can work with the files locally while keeping them in sync with GitHub.`, m.GitHubURL, branch, m.GitHubPath)
	}

	return m.layout.Render(content)
}

// viewCancelled renders the cancellation screen when setup is aborted.
func (m *SetupModel) viewCancelled() string {
	m.layout = m.layout.SetConfig(components.LayoutConfig{
		Title:    "❌ Setup Cancelled",
		Subtitle: "Rulem will not be configured.",
		HelpText: "Press any key to continue",
	})

	content := `Setup was cancelled. Rulem has not been configured and will need to be set up before you can use it.`

	return m.layout.Render(content)
}

// Helper functions

// resetTextInputForState is a helper function that resets the text input and transitions to a new state.
// This reduces code duplication across state transition logic.
//
// Parameters:
//   - state: The target state to transition to
//   - value: The initial value for the text input
//   - placeholder: The placeholder text to display
//   - echoMode: The echo mode (Normal or Password) for the input
//
// Returns a Bubble Tea command for the text input blink animation.
func (m *SetupModel) resetTextInputForState(state SetupState, value, placeholder string, echoMode textinput.EchoMode) tea.Cmd {
	m.state = state
	m.textInput.Reset()
	m.textInput.SetValue(value)
	m.textInput.Placeholder = placeholder
	m.textInput.EchoMode = echoMode
	m.textInput.Focus()
	m.layout = m.layout.ClearError()
	return textinput.Blink
}

// deriveClonePathPlaceholder returns a smart default clone path based on the repository URL.
// It extracts the repository name from the URL and creates a path in the default storage directory.
// If URL parsing fails, it returns the default storage directory.
func (m *SetupModel) deriveClonePathPlaceholder() string {
	defaultPath := repository.GetDefaultStorageDir()
	if m.GitHubURL != "" {
		if info, err := repository.ParseGitURL(m.GitHubURL); err == nil && info.Repo != "" {
			return filepath.Join(repository.GetDefaultStorageDir(), info.Repo)
		}
	}
	return defaultPath
}
