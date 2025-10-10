package settingsmenu

import (
	"testing"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/helpers"

	tea "github.com/charmbracelet/bubbletea"
)

// Test helpers

func createTestModel(t *testing.T) *SettingsModel {
	t.Helper()
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)
	return NewSettingsModel(ctx)
}

func createTestModelWithConfig(t *testing.T, cfg *config.Config) *SettingsModel {
	t.Helper()
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, cfg, logger)
	model := NewSettingsModel(ctx)
	model.currentConfig = cfg
	return model
}

func createLocalConfig(path string) *config.Config {
	return &config.Config{
		Central: repository.CentralRepositoryConfig{
			Path: path,
		},
	}
}

func createGitHubConfig(path, url, branch string) *config.Config {
	return &config.Config{
		Central: repository.CentralRepositoryConfig{
			Path:      path,
			RemoteURL: &url,
			Branch:    &branch,
		},
	}
}

// Phase 1: Foundation Tests

func TestNewSettingsModel(t *testing.T) {
	model := createTestModel(t)

	if model == nil {
		t.Fatal("NewSettingsModel returned nil")
	}

	if model.state != SettingsStateMainMenu {
		t.Errorf("Expected initial state to be SettingsStateMainMenu, got %v", model.state)
	}

	if model.logger == nil {
		t.Error("Logger not properly set")
	}

	if model.credManager == nil {
		t.Error("Credential manager not properly set")
	}

	if len(model.repositoryOptions) != 2 {
		t.Errorf("Expected 2 repository options, got %d", len(model.repositoryOptions))
	}

	if model.textInput.CharLimit != 256 {
		t.Errorf("Expected char limit to be 256, got %d", model.textInput.CharLimit)
	}
}

func TestSettingsModelInit(t *testing.T) {
	model := createTestModel(t)
	cmd := model.Init()

	if cmd == nil {
		t.Error("Init should return a command (load config + blink)")
	}
}

func TestSettingsState_String(t *testing.T) {
	tests := []struct {
		state    SettingsState
		expected string
	}{
		{SettingsStateMainMenu, "MainMenu"},
		{SettingsStateSelectChange, "SelectChange"},
		{SettingsStateConfirmTypeChange, "ConfirmTypeChange"},
		{SettingsStateRepositoryTypeSelect, "RepositoryTypeSelect"},
		{SettingsStateUpdateLocalPath, "UpdateLocalPath"},
		{SettingsStateUpdateGitHubPAT, "UpdateGitHubPAT"},
		{SettingsStateUpdateGitHubURL, "UpdateGitHubURL"},
		{SettingsStateUpdateGitHubBranch, "UpdateGitHubBranch"},
		{SettingsStateUpdateGitHubPath, "UpdateGitHubPath"},
		{SettingsStateManualRefresh, "ManualRefresh"},
		{SettingsStateRefreshInProgress, "RefreshInProgress"},
		{SettingsStateConfirmation, "Confirmation"},
		{SettingsStateComplete, "Complete"},
		{SettingsStateError, "Error"},
		{SettingsState(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("State.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTransitionTo(t *testing.T) {
	model := createTestModel(t)
	model.state = SettingsStateMainMenu
	model.selectedOption = 5

	updatedModel := model.transitionTo(SettingsStateSelectChange)

	if updatedModel.state != SettingsStateSelectChange {
		t.Errorf("Expected state to be SettingsStateSelectChange, got %v", updatedModel.state)
	}

	if updatedModel.previousState != SettingsStateMainMenu {
		t.Errorf("Expected previousState to be SettingsStateMainMenu, got %v", updatedModel.previousState)
	}

	if updatedModel.selectedOption != 0 {
		t.Errorf("Expected selectedOption to be reset to 0, got %d", updatedModel.selectedOption)
	}
}

func TestTransitionBack(t *testing.T) {
	model := createTestModel(t)
	model.state = SettingsStateSelectChange
	model.previousState = SettingsStateMainMenu

	updatedModel := model.transitionBack()

	if updatedModel.state != SettingsStateMainMenu {
		t.Errorf("Expected state to be SettingsStateMainMenu, got %v", updatedModel.state)
	}
}

func TestResetTemporaryChanges(t *testing.T) {
	model := createTestModel(t)
	model.newRepositoryType = "github"
	model.newStorageDir = "/test/path"
	model.newGitHubURL = "https://github.com/test/repo.git"
	model.newGitHubBranch = "main"
	model.newGitHubPath = "/test/clone"
	model.newGitHubPAT = "ghp_test"
	model.patAction = PATActionUpdate
	model.hasChanges = true

	model.resetTemporaryChanges()

	if model.newRepositoryType != "" {
		t.Error("newRepositoryType should be reset")
	}
	if model.newStorageDir != "" {
		t.Error("newStorageDir should be reset")
	}
	if model.newGitHubURL != "" {
		t.Error("newGitHubURL should be reset")
	}
	if model.newGitHubBranch != "" {
		t.Error("newGitHubBranch should be reset")
	}
	if model.newGitHubPath != "" {
		t.Error("newGitHubPath should be reset")
	}
	if model.newGitHubPAT != "" {
		t.Error("newGitHubPAT should be reset")
	}
	if model.patAction != PATActionNone {
		t.Error("patAction should be reset")
	}
	if model.hasChanges {
		t.Error("hasChanges should be reset")
	}
}

func TestIsGitHubRepo(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "local config",
			config:   createLocalConfig("/test/path"),
			expected: false,
		},
		{
			name:     "github config",
			config:   createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModel(t)
			model.currentConfig = tt.config

			if got := model.isGitHubRepo(); got != tt.expected {
				t.Errorf("isGitHubRepo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetMenuOptions_LocalRepo(t *testing.T) {
	model := createTestModelWithConfig(t, createLocalConfig("/test/path"))

	options := model.getMenuOptions()

	// Local repo should have: Repository Type, Local Path, Back
	if len(options) != 3 {
		t.Errorf("Expected 3 options for local repo, got %d", len(options))
	}

	// Check that repository type is first
	if options[0].Option != ChangeOptionRepositoryType {
		t.Error("First option should be ChangeOptionRepositoryType")
	}

	// Check that local path is second
	if options[1].Option != ChangeOptionLocalPath {
		t.Error("Second option should be ChangeOptionLocalPath")
	}

	// Check that back is last
	if options[len(options)-1].Option != ChangeOptionBack {
		t.Error("Last option should be ChangeOptionBack")
	}
}

func TestGetMenuOptions_GitHubRepo(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))

	options := model.getMenuOptions()

	// GitHub repo should have: Repository Type, URL, Branch, Path, PAT, Refresh, Back
	if len(options) != 7 {
		t.Errorf("Expected 7 options for GitHub repo, got %d", len(options))
	}

	// Verify all GitHub options are present
	hasURL := false
	hasBranch := false
	hasPath := false
	hasPAT := false
	hasRefresh := false

	for _, opt := range options {
		switch opt.Option {
		case ChangeOptionGitHubURL:
			hasURL = true
		case ChangeOptionGitHubBranch:
			hasBranch = true
		case ChangeOptionGitHubPath:
			hasPath = true
		case ChangeOptionGitHubPAT:
			hasPAT = true
		case ChangeOptionManualRefresh:
			hasRefresh = true
		}
	}

	if !hasURL {
		t.Error("GitHub menu should include URL option")
	}
	if !hasBranch {
		t.Error("GitHub menu should include Branch option")
	}
	if !hasPath {
		t.Error("GitHub menu should include Path option")
	}
	if !hasPAT {
		t.Error("GitHub menu should include PAT option")
	}
	if !hasRefresh {
		t.Error("GitHub menu should include Refresh option")
	}
}

// Phase 2: Repository Type Switching Tests

func TestHandleMainMenuKeys(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		expectedState SettingsState
	}{
		{"enter proceeds", "enter", SettingsStateSelectChange},
		{"space proceeds", " ", SettingsStateSelectChange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModel(t)
			model.state = SettingsStateMainMenu

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "enter" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
			}

			updatedModel, _ := model.handleMainMenuKeys(keyMsg)

			if updatedModel.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, updatedModel.state)
			}
		})
	}
}

func TestHandleSelectChangeKeys_Navigation(t *testing.T) {
	model := createTestModelWithConfig(t, createLocalConfig("/test/path"))
	model.state = SettingsStateSelectChange
	options := model.getMenuOptions()

	// Test down navigation
	initialOption := model.selectedOption
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updatedModel, _ := model.handleSelectChangeKeys(keyMsg)

	if updatedModel.selectedOption != initialOption+1 {
		t.Errorf("Down navigation should increment selectedOption")
	}

	// Test up navigation
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	updatedModel, _ = updatedModel.handleSelectChangeKeys(keyMsg)

	if updatedModel.selectedOption != initialOption {
		t.Errorf("Up navigation should decrement selectedOption")
	}

	// Test boundary - can't go below 0
	model.selectedOption = 0
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	updatedModel, _ = model.handleSelectChangeKeys(keyMsg)

	if updatedModel.selectedOption != 0 {
		t.Errorf("Up navigation at 0 should stay at 0")
	}

	// Test boundary - can't go above max
	model.selectedOption = len(options) - 1
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updatedModel, _ = model.handleSelectChangeKeys(keyMsg)

	if updatedModel.selectedOption != len(options)-1 {
		t.Errorf("Down navigation at max should stay at max")
	}
}

func TestHandleSelectChangeKeys_Selection(t *testing.T) {
	tests := []struct {
		name          string
		option        ChangeOption
		expectedState SettingsState
	}{
		{"select back", ChangeOptionBack, SettingsStateMainMenu},
		{"select repository type", ChangeOptionRepositoryType, SettingsStateConfirmTypeChange},
		{"select local path", ChangeOptionLocalPath, SettingsStateUpdateLocalPath},
		{"select github pat", ChangeOptionGitHubPAT, SettingsStateUpdateGitHubPAT},
		{"select github url", ChangeOptionGitHubURL, SettingsStateUpdateGitHubURL},
		{"select github branch", ChangeOptionGitHubBranch, SettingsStateUpdateGitHubBranch},
		{"select github path", ChangeOptionGitHubPath, SettingsStateUpdateGitHubPath},
		{"select manual refresh", ChangeOptionManualRefresh, SettingsStateManualRefresh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
			model.state = SettingsStateSelectChange

			// Find the option in the menu
			options := model.getMenuOptions()
			targetIndex := -1
			for i, opt := range options {
				if opt.Option == tt.option {
					targetIndex = i
					break
				}
			}

			if targetIndex == -1 && tt.option != ChangeOptionLocalPath {
				t.Skipf("Option %v not available in GitHub config", tt.option)
			}

			if targetIndex >= 0 {
				model.selectedOption = targetIndex

				keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
				updatedModel, _ := model.handleSelectChangeKeys(keyMsg)

				if updatedModel.state != tt.expectedState {
					t.Errorf("Expected state %v, got %v", tt.expectedState, updatedModel.state)
				}

				if updatedModel.changeType != tt.option {
					t.Errorf("Expected changeType %v, got %v", tt.option, updatedModel.changeType)
				}
			}
		})
	}
}

func TestHandleConfirmTypeChangeKeys(t *testing.T) {
	tests := []struct {
		name          string
		keyType       tea.KeyType
		runes         []rune
		expectedState SettingsState
	}{
		{"yes confirms", tea.KeyRunes, []rune("y"), SettingsStateRepositoryTypeSelect},
		{"Y confirms", tea.KeyRunes, []rune("Y"), SettingsStateRepositoryTypeSelect},
		{"enter confirms", tea.KeyEnter, nil, SettingsStateRepositoryTypeSelect},
		{"no cancels", tea.KeyRunes, []rune("n"), SettingsStateSelectChange},
		{"N cancels", tea.KeyRunes, []rune("N"), SettingsStateSelectChange},
		{"esc cancels", tea.KeyEsc, nil, SettingsStateSelectChange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModel(t)
			model.state = SettingsStateConfirmTypeChange

			var keyMsg tea.KeyMsg
			if tt.runes != nil {
				keyMsg = tea.KeyMsg{Type: tt.keyType, Runes: tt.runes}
			} else {
				keyMsg = tea.KeyMsg{Type: tt.keyType}
			}

			updatedModel, _ := model.handleConfirmTypeChangeKeys(keyMsg)

			if updatedModel.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, updatedModel.state)
			}
		})
	}
}

func TestHandleRepositoryTypeSelectKeys(t *testing.T) {
	model := createTestModel(t)
	model.state = SettingsStateRepositoryTypeSelect

	// Test selecting local repository
	model.selectedOption = 0 // Local is first
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.handleRepositoryTypeSelectKeys(keyMsg)

	if updatedModel.newRepositoryType != "local" {
		t.Errorf("Expected newRepositoryType to be 'local', got %v", updatedModel.newRepositoryType)
	}

	if updatedModel.state != SettingsStateUpdateLocalPath {
		t.Errorf("Expected state to be UpdateLocalPath, got %v", updatedModel.state)
	}

	// Test selecting GitHub repository
	model.selectedOption = 1 // GitHub is second
	keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.handleRepositoryTypeSelectKeys(keyMsg)

	if updatedModel.newRepositoryType != "github" {
		t.Errorf("Expected newRepositoryType to be 'github', got %v", updatedModel.newRepositoryType)
	}

	if updatedModel.state != SettingsStateUpdateGitHubURL {
		t.Errorf("Expected state to be UpdateGitHubURL, got %v", updatedModel.state)
	}
}

func TestHandleUpdateLocalPathKeys(t *testing.T) {
	model := createTestModel(t)
	model.state = SettingsStateUpdateLocalPath
	model.changeType = ChangeOptionLocalPath
	model.textInput.SetValue("/valid/path")

	// Test enter submits
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.handleUpdateLocalPathKeys(keyMsg)

	// Should attempt validation (which may fail without proper filesystem)
	// but the key handler should process it
	if updatedModel == nil {
		t.Error("handleUpdateLocalPathKeys should return a model")
	}

	// Test escape cancels
	model.state = SettingsStateUpdateLocalPath
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.handleUpdateLocalPathKeys(keyMsg)

	if updatedModel.state != SettingsStateSelectChange {
		t.Errorf("Escape should return to SelectChange, got %v", updatedModel.state)
	}
}

func TestHandleUpdateGitHubPATKeys(t *testing.T) {
	model := createTestModel(t)
	model.state = SettingsStateUpdateGitHubPAT
	model.changeType = ChangeOptionGitHubPAT

	// Test Ctrl+D removes PAT
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlD}
	updatedModel, _ := model.handleUpdateGitHubPATKeys(keyMsg)

	if updatedModel.patAction != PATActionRemove {
		t.Errorf("Ctrl+D should set patAction to Remove, got %v", updatedModel.patAction)
	}

	if updatedModel.state != SettingsStateConfirmation {
		t.Errorf("Ctrl+D should transition to Confirmation, got %v", updatedModel.state)
	}

	// Test enter with value updates PAT
	model.state = SettingsStateUpdateGitHubPAT
	model.textInput.SetValue("ghp_testtoken123")
	keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.handleUpdateGitHubPATKeys(keyMsg)

	if updatedModel.patAction != PATActionUpdate {
		t.Errorf("Enter with value should set patAction to Update, got %v", updatedModel.patAction)
	}

	if updatedModel.newGitHubPAT != "ghp_testtoken123" {
		t.Errorf("Expected newGitHubPAT to be set, got %v", updatedModel.newGitHubPAT)
	}

	// Test enter with empty value shows error
	model.state = SettingsStateUpdateGitHubPAT
	model.textInput.SetValue("")
	keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.handleUpdateGitHubPATKeys(keyMsg)

	if cmd == nil {
		t.Error("Empty PAT should return error command")
	}
}

func TestHandleManualRefreshKeys(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateManualRefresh

	// Test yes confirms
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, cmd := model.handleManualRefreshKeys(keyMsg)

	if cmd == nil {
		t.Error("Confirming refresh should return a command")
	}

	// Test no cancels
	model.state = SettingsStateManualRefresh
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updatedModel, _ = model.handleManualRefreshKeys(keyMsg)

	if updatedModel.state != SettingsStateSelectChange {
		t.Errorf("No should return to SelectChange, got %v", updatedModel.state)
	}
}

func TestHandleConfirmationKeys(t *testing.T) {
	model := createTestModel(t)
	model.state = SettingsStateConfirmation
	model.hasChanges = true

	// Test yes saves
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	_, cmd := model.handleConfirmationKeys(keyMsg)

	if cmd == nil {
		t.Error("Confirming should return save command")
	}

	// Test no discards
	model.state = SettingsStateConfirmation
	model.newStorageDir = "/test/path"
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updatedModel, _ := model.handleConfirmationKeys(keyMsg)

	if updatedModel.newStorageDir != "" {
		t.Error("No should reset temporary changes")
	}

	if updatedModel.state != SettingsStateSelectChange {
		t.Errorf("No should return to SelectChange, got %v", updatedModel.state)
	}
}

func TestUpdateTextInput(t *testing.T) {
	model := createTestModel(t)
	model.hasChanges = false

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
	updatedModel, cmd := model.updateTextInput(keyMsg)

	if cmd == nil {
		t.Error("updateTextInput should return a command (cursor blink)")
	}

	if !updatedModel.hasChanges {
		t.Error("updateTextInput should mark hasChanges as true")
	}
}

func TestViewRendering(t *testing.T) {
	tests := []struct {
		name  string
		state SettingsState
	}{
		{"main menu", SettingsStateMainMenu},
		{"select change", SettingsStateSelectChange},
		{"confirm type change", SettingsStateConfirmTypeChange},
		{"repository type select", SettingsStateRepositoryTypeSelect},
		{"update local path", SettingsStateUpdateLocalPath},
		{"update github pat", SettingsStateUpdateGitHubPAT},
		{"update github url", SettingsStateUpdateGitHubURL},
		{"update github branch", SettingsStateUpdateGitHubBranch},
		{"update github path", SettingsStateUpdateGitHubPath},
		{"manual refresh", SettingsStateManualRefresh},
		{"refresh in progress", SettingsStateRefreshInProgress},
		{"confirmation", SettingsStateConfirmation},
		{"complete", SettingsStateComplete},
		{"error", SettingsStateError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createLocalConfig("/test/path"))
			model.state = tt.state

			view := model.View()

			if view == "" {
				t.Errorf("View for state %v should not be empty", tt.state)
			}

			if len(view) < 10 {
				t.Errorf("View for state %v seems too short: %d characters", tt.state, len(view))
			}
		})
	}
}

func TestFormatCurrentConfig_Local(t *testing.T) {
	model := createTestModelWithConfig(t, createLocalConfig("/test/path"))

	formatted := model.formatCurrentConfig()

	if formatted == "" {
		t.Error("formatCurrentConfig should not return empty string")
	}

	// Should contain local directory info
	if len(formatted) < 20 {
		t.Error("Formatted config seems too short")
	}
}

func TestFormatCurrentConfig_GitHub(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))

	formatted := model.formatCurrentConfig()

	if formatted == "" {
		t.Error("formatCurrentConfig should not return empty string")
	}

	// Should contain GitHub-specific info
	if len(formatted) < 50 {
		t.Error("Formatted GitHub config seems too short")
	}
}

func TestFormatChangesSummary(t *testing.T) {
	tests := []struct {
		name       string
		changeType ChangeOption
		setup      func(*SettingsModel)
	}{
		{
			name:       "repository type change to local",
			changeType: ChangeOptionRepositoryType,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "local"
				m.newStorageDir = "/new/path"
			},
		},
		{
			name:       "repository type change to github",
			changeType: ChangeOptionRepositoryType,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/test/repo.git"
				m.newGitHubBranch = "main"
				m.newGitHubPath = "/clone/path"
				m.patAction = PATActionUpdate
			},
		},
		{
			name:       "local path change",
			changeType: ChangeOptionLocalPath,
			setup: func(m *SettingsModel) {
				m.newStorageDir = "/new/path"
			},
		},
		{
			name:       "github url change",
			changeType: ChangeOptionGitHubURL,
			setup: func(m *SettingsModel) {
				m.newGitHubURL = "https://github.com/new/repo.git"
			},
		},
		{
			name:       "pat update",
			changeType: ChangeOptionGitHubPAT,
			setup: func(m *SettingsModel) {
				m.patAction = PATActionUpdate
			},
		},
		{
			name:       "pat remove",
			changeType: ChangeOptionGitHubPAT,
			setup: func(m *SettingsModel) {
				m.patAction = PATActionRemove
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModel(t)
			model.changeType = tt.changeType
			tt.setup(model)

			summary := model.formatChangesSummary()

			if summary == "" {
				t.Error("formatChangesSummary should not return empty string")
			}

			if len(summary) < 10 {
				t.Error("Summary seems too short")
			}
		})
	}
}

func TestGetCleanupWarning(t *testing.T) {
	// Test local repo warning
	localModel := createTestModelWithConfig(t, createLocalConfig("/test/path"))
	localWarning := localModel.getCleanupWarning()

	if localWarning == "" {
		t.Error("Local repo warning should not be empty")
	}

	// Test GitHub repo warning
	githubModel := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	githubWarning := githubModel.getCleanupWarning()

	if githubWarning == "" {
		t.Error("GitHub repo warning should not be empty")
	}

	// GitHub warning should be different (more detailed)
	if len(githubWarning) <= len(localWarning) {
		t.Error("GitHub warning should be more detailed than local warning")
	}
}

// func TestPluralizeHelper(t *testing.T) {
// 	tests := []struct {
// 		count    int
// 		singular string
// 		plural   string
// 		expected string
// 	}{
// 		{0, "item", "items", "0 items"},
// 		{1, "item", "items", "1 item"},
// 		{2, "item", "items", "2 items"},
// 		{10, "change", "changes", "10 changes"},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.expected, func(t *testing.T) {
// 			result := pluralize(tt.count, tt.singular, tt.plural)
// 			if result != tt.expected {
// 				t.Errorf("pluralize(%d, %q, %q) = %q, want %q", tt.count, tt.singular, tt.plural, result, tt.expected)
// 			}
// 		})
// 	}
// }

func TestMessageHandling(t *testing.T) {
	model := createTestModel(t)

	// Test error message
	errorMsg := settingsErrorMsg{err: &testError{"test error"}}
	updatedModel, _ := model.Update(errorMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.state != SettingsStateError {
		t.Errorf("Error message should transition to Error state, got %v", settingsModel.state)
	}

	// Test complete message
	model = createTestModel(t)
	completeMsg := settingsCompleteMsg{}
	updatedModel, cmd := model.Update(completeMsg)
	settingsModel = updatedModel.(*SettingsModel)

	if settingsModel.state != SettingsStateComplete {
		t.Errorf("Complete message should transition to Complete state, got %v", settingsModel.state)
	}

	if cmd == nil {
		t.Error("Complete message should return config reload command")
	}

	// Test refresh complete message
	model = createTestModel(t)
	model.refreshInProgress = true
	refreshMsg := refreshCompleteMsg{syncInfo: &repository.SyncInfo{}, err: nil}
	updatedModel, _ = model.Update(refreshMsg)
	settingsModel = updatedModel.(*SettingsModel)

	if settingsModel.refreshInProgress {
		t.Error("Refresh complete should clear refreshInProgress flag")
	}

	if settingsModel.state != SettingsStateMainMenu {
		t.Errorf("Refresh complete should return to MainMenu, got %v", settingsModel.state)
	}
}

// Helper types for testing

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
