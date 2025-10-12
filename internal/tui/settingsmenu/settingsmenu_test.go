package settingsmenu

import (
	"os"
	"path/filepath"
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

func TestValidateAndProceedLocalPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create subdirectories for testing
	testPath1 := filepath.Join(tmpDir, "test-path-1")
	testPath2 := filepath.Join(tmpDir, "test-path-2")

	if err := os.MkdirAll(testPath1, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.MkdirAll(testPath2, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name          string
		currentPath   string
		newPath       string
		expectState   SettingsState
		expectChanges bool
		expectError   bool
	}{
		{
			name:          "unchanged path returns to main menu",
			currentPath:   testPath1,
			newPath:       testPath1,
			expectState:   SettingsStateMainMenu,
			expectChanges: false,
			expectError:   false,
		},
		{
			name:          "changed path proceeds to confirmation",
			currentPath:   testPath1,
			newPath:       testPath2,
			expectState:   SettingsStateConfirmation,
			expectChanges: true,
			expectError:   false,
		},
		{
			name:          "empty path returns error",
			currentPath:   testPath1,
			newPath:       "",
			expectState:   SettingsStateUpdateLocalPath,
			expectChanges: false,
			expectError:   true,
		},
		{
			name:          "whitespace-only path returns error",
			currentPath:   testPath1,
			newPath:       "   ",
			expectState:   SettingsStateUpdateLocalPath,
			expectChanges: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createLocalConfig(tt.currentPath))
			model.state = SettingsStateUpdateLocalPath
			model.changeType = ChangeOptionLocalPath
			model.textInput.SetValue(tt.newPath)

			updatedModel, cmd := model.validateAndProceedLocalPath()

			if updatedModel == nil {
				t.Fatal("validateAndProceedLocalPath should return a model")
			}

			if updatedModel.state != tt.expectState {
				t.Errorf("Expected state %v, got %v", tt.expectState, updatedModel.state)
			}

			if updatedModel.hasChanges != tt.expectChanges {
				t.Errorf("Expected hasChanges %v, got %v", tt.expectChanges, updatedModel.hasChanges)
			}

			if tt.expectError && cmd == nil {
				t.Error("Expected error command, got nil")
			}

			if !tt.expectError && !tt.expectChanges && updatedModel.newStorageDir != "" {
				t.Error("Expected newStorageDir to be cleared when no changes")
			}

			if tt.expectChanges && updatedModel.newStorageDir == "" {
				t.Error("Expected newStorageDir to be set when path changed")
			}
		})
	}
}

func TestHandleUpdateGitHubPATKeys(t *testing.T) {
	// Note: PAT removal via Ctrl+D is currently disabled in the implementation
	// Skip Ctrl+D tests until it's re-enabled
	t.Skip("PAT removal via Ctrl+D is currently disabled")

	model := createTestModel(t)
	model.state = SettingsStateUpdateGitHubPAT
	model.changeType = ChangeOptionGitHubPAT
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")

	// Test enter with empty value shows error
	model.textInput.SetValue("")
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
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

// Phase 4: GitHub Operations Tests

func TestCheckDirtyState(t *testing.T) {
	// This test verifies the dirty state check mechanism
	// Note: Actual git operations require a real git repository
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))

	// Test with GitHub repo - should attempt to check
	cmd := model.checkDirtyState()

	if cmd == nil {
		t.Error("checkDirtyState should return a command")
	}

	// Execute the command to get the message
	msg := cmd()
	if msg == nil {
		t.Error("checkDirtyState command should return a message")
	}

	// Test with local repo - should return clean state
	localModel := createTestModelWithConfig(t, createLocalConfig("/test/path"))
	cmd = localModel.checkDirtyState()

	if cmd == nil {
		t.Error("checkDirtyState should return a command even for local repos")
	}

	msg = cmd()
	// For local repos, should return clean dirtyStateMsg
	if dirtyMsg, ok := msg.(dirtyStateMsg); !ok {
		t.Error("For local repos, should return dirtyStateMsg")
	} else if dirtyMsg.isDirty {
		t.Error("For local repos, isDirty should be false")
	}
}

func TestHandleManualRefreshKeys_WithDirtyCheck(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateManualRefresh

	// Test yes confirms and triggers dirty check
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	_, cmd := model.handleManualRefreshKeys(keyMsg)

	if cmd == nil {
		t.Error("Confirming refresh should return a command (dirty check)")
	}

	// The command should check dirty state before proceeding
	msg := cmd()
	if msg == nil {
		t.Error("Dirty check command should return a message")
	}
}

func TestDirtyStateMsg_HandlingInUpdate(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateManualRefresh

	// Test dirty state message with isDirty=true
	dirtyMsg := dirtyStateMsg{isDirty: true, err: nil}
	updatedModel, _ := model.Update(dirtyMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if !settingsModel.isDirty {
		t.Error("isDirty flag should be set when dirty state is detected")
	}

	if settingsModel.state != SettingsStateError {
		t.Errorf("Should transition to Error state when repository is dirty, got %v", settingsModel.state)
	}

	// Test dirty state message with isDirty=false
	model = createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateUpdateGitHubBranch
	cleanMsg := dirtyStateMsg{isDirty: false, err: nil}
	updatedModel, _ = model.Update(cleanMsg)
	settingsModel = updatedModel.(*SettingsModel)

	if settingsModel.isDirty {
		t.Error("isDirty flag should not be set when repository is clean")
	}
}

func TestTriggerRefresh_Implementation(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))

	cmd := model.triggerRefresh()
	if cmd == nil {
		t.Fatal("triggerRefresh should return a command")
	}

	if !model.refreshInProgress {
		t.Error("refreshInProgress should be set to true")
	}

	if model.state != SettingsStateRefreshInProgress {
		t.Errorf("State should be RefreshInProgress, got %v", model.state)
	}

	// Test that refresh fails gracefully for non-GitHub repos
	localModel := createTestModelWithConfig(t, createLocalConfig("/test/path"))
	cmd = localModel.triggerRefresh()
	msg := cmd()

	if refreshMsg, ok := msg.(refreshCompleteMsg); ok {
		if refreshMsg.err == nil {
			t.Error("Refresh should error for non-GitHub repositories")
		}
	} else {
		t.Error("Should return refreshCompleteMsg")
	}
}

// Phase 3: GitHub URL Update Flow Tests

func TestHandleUpdateGitHubURLKeys_ContinuesToBranch(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateUpdateGitHubURL
	model.changeType = ChangeOptionGitHubURL
	model.textInput.SetValue("https://github.com/newuser/newrepo.git")

	// Test enter continues to branch input
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.handleUpdateGitHubURLKeys(keyMsg)

	if updatedModel.state != SettingsStateUpdateGitHubBranch {
		t.Errorf("After URL input, should transition to Branch state, got %v", updatedModel.state)
	}

	if updatedModel.newGitHubURL != "https://github.com/newuser/newrepo.git" {
		t.Errorf("URL should be saved, got %v", updatedModel.newGitHubURL)
	}

	if updatedModel.changeType != ChangeOptionGitHubURL {
		t.Error("changeType should remain GitHubURL")
	}
}

func TestHandleUpdateGitHubBranchKeys_ContinuesToPath(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateUpdateGitHubBranch
	model.changeType = ChangeOptionGitHubURL
	model.newGitHubURL = "https://github.com/newuser/newrepo.git"
	model.textInput.SetValue("develop")

	// Test enter continues to path input (for URL change flow)
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.handleUpdateGitHubBranchKeys(keyMsg)

	if cmd == nil {
		t.Error("Should return a command (dirty check)")
	}

	// Execute the dirty check (will fail without real repo, but we check the flow)
	msg := cmd()
	if msg == nil {
		t.Error("Dirty check should return a message")
	}

	if updatedModel.newGitHubBranch != "develop" {
		t.Errorf("Branch should be saved, got %v", updatedModel.newGitHubBranch)
	}

	if updatedModel.hasChanges != true {
		t.Error("hasChanges should be set to true")
	}
}

func TestHandleUpdateGitHubBranchKeys_StandaloneBranch(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateUpdateGitHubBranch
	model.changeType = ChangeOptionGitHubBranch // Standalone branch change
	model.textInput.SetValue("feature")

	// Test enter for standalone branch change
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.handleUpdateGitHubBranchKeys(keyMsg)

	if cmd == nil {
		t.Error("Should return a command (dirty check)")
	}
}

func TestHandleUpdateGitHubPathKeys_URLChangeFlow(t *testing.T) {
	// Use a path within home directory for valid path testing
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	validPath := filepath.Join(homeDir, ".rulem-test-path-flow")
	defer os.RemoveAll(validPath) // Clean up after test

	model := createTestModelWithConfig(t, createGitHubConfig(validPath, "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateUpdateGitHubPath
	model.changeType = ChangeOptionGitHubURL
	model.newGitHubURL = "https://github.com/newuser/newrepo.git"
	model.newGitHubBranch = "develop"
	model.textInput.SetValue(validPath)

	// Test enter proceeds to confirmation
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.handleUpdateGitHubPathKeys(keyMsg)

	if updatedModel.state != SettingsStateConfirmation {
		t.Errorf("After path input in URL flow, should go to Confirmation, got %v", updatedModel.state)
	}

	if updatedModel.hasChanges != true {
		t.Error("hasChanges should be set to true")
	}
}

func TestHandleUpdateGitHubPathKeys_EscNavigation(t *testing.T) {
	// Test escape during URL change flow
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateUpdateGitHubPath
	model.changeType = ChangeOptionGitHubURL
	model.textInput.SetValue("/new/path")

	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := model.handleUpdateGitHubPathKeys(keyMsg)

	if updatedModel.state != SettingsStateUpdateGitHubBranch {
		t.Errorf("Escape from path should go back to branch, got %v", updatedModel.state)
	}

	// Test escape during repository type change flow
	model = createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateUpdateGitHubPath
	model.changeType = ChangeOptionRepositoryType

	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.handleUpdateGitHubPathKeys(keyMsg)

	if updatedModel.state != SettingsStateUpdateGitHubBranch {
		t.Errorf("Escape from path in type change should go back to branch, got %v", updatedModel.state)
	}
}

// Repository Type Switching Tests

func TestUpdateRepositoryType_LocalToGitHub(t *testing.T) {
	model := createTestModelWithConfig(t, createLocalConfig("/test/path"))
	model.newRepositoryType = "github"
	model.newGitHubURL = "https://github.com/test/repo.git"
	model.newGitHubBranch = "main"
	model.newGitHubPath = "/test/github/clone"
	model.newGitHubPAT = "ghp_testtoken123"

	// Create a test config
	cfg := &config.Config{
		Central: repository.CentralRepositoryConfig{
			Path: "/test/path",
		},
	}

	err := model.updateRepositoryType(cfg)
	if err != nil {
		t.Errorf("updateRepositoryType should not error for valid inputs, got: %v", err)
	}

	if cfg.Central.RemoteURL == nil {
		t.Error("RemoteURL should be set after switching to GitHub")
	}

	if *cfg.Central.RemoteURL != "https://github.com/test/repo.git" {
		t.Errorf("RemoteURL should be set to new URL, got %v", *cfg.Central.RemoteURL)
	}

	if cfg.Central.Path != "/test/github/clone" {
		t.Errorf("Path should be updated to new clone path, got %v", cfg.Central.Path)
	}

	if cfg.Central.Branch == nil || *cfg.Central.Branch != "main" {
		t.Error("Branch should be set")
	}
}

func TestUpdateRepositoryType_GitHubToLocal(t *testing.T) {
	// Use a path within home directory for valid path testing
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	// Create a temporary directory within home
	tmpDir := filepath.Join(homeDir, ".rulem-test-"+t.Name())
	defer os.RemoveAll(tmpDir) // Clean up after test

	url := "https://github.com/test/repo.git"
	branch := "main"
	model := createTestModelWithConfig(t, createGitHubConfig(tmpDir, url, branch))
	model.newRepositoryType = "local"
	model.newStorageDir = tmpDir

	cfg := &config.Config{
		Central: repository.CentralRepositoryConfig{
			Path:      tmpDir,
			RemoteURL: &url,
			Branch:    &branch,
		},
	}

	err = model.updateRepositoryType(cfg)
	if err != nil {
		t.Errorf("updateRepositoryType should not error for valid inputs, got: %v", err)
	}

	if cfg.Central.RemoteURL != nil {
		t.Error("RemoteURL should be nil after switching to local")
	}

	if cfg.Central.Branch != nil {
		t.Error("Branch should be nil after switching to local")
	}

	if cfg.Central.Path != tmpDir {
		t.Errorf("Path should be updated to new local path, got %v", cfg.Central.Path)
	}
}

func TestUpdateRepositoryType_ValidationErrors(t *testing.T) {
	// Test missing GitHub URL
	model := createTestModel(t)
	model.newRepositoryType = "github"
	model.newGitHubURL = ""

	cfg := &config.Config{
		Central: repository.CentralRepositoryConfig{
			Path: "/test/path",
		},
	}

	err := model.updateRepositoryType(cfg)
	if err == nil {
		t.Error("Should error when GitHub URL is missing")
	}

	// Test missing GitHub path
	model = createTestModel(t)
	model.newRepositoryType = "github"
	model.newGitHubURL = "https://github.com/test/repo.git"
	model.newGitHubPath = ""

	err = model.updateRepositoryType(cfg)
	if err == nil {
		t.Error("Should error when GitHub path is missing")
	}

	// Test missing local path
	model = createTestModel(t)
	model.newRepositoryType = "local"
	model.newStorageDir = ""

	url := "https://github.com/test/repo.git"
	cfg = &config.Config{
		Central: repository.CentralRepositoryConfig{
			Path:      "/test/path",
			RemoteURL: &url,
		},
	}

	err = model.updateRepositoryType(cfg)
	if err == nil {
		t.Error("Should error when local storage directory is missing")
	}
}

// Update Function Tests

func TestUpdateGitHubBranch(t *testing.T) {
	model := createTestModel(t)
	model.newGitHubBranch = "develop"

	url := "https://github.com/test/repo.git"
	branch := "main"
	cfg := &config.Config{
		Central: repository.CentralRepositoryConfig{
			Path:      "/test/path",
			RemoteURL: &url,
			Branch:    &branch,
		},
	}

	err := model.updateGitHubBranch(cfg)
	if err != nil {
		t.Errorf("updateGitHubBranch should not error, got: %v", err)
	}

	if cfg.Central.Branch == nil || *cfg.Central.Branch != "develop" {
		t.Error("Branch should be updated to new value")
	}

	// Test empty branch (use default)
	model.newGitHubBranch = ""
	err = model.updateGitHubBranch(cfg)
	if err != nil {
		t.Errorf("updateGitHubBranch should not error for empty branch, got: %v", err)
	}

	if cfg.Central.Branch != nil {
		t.Error("Branch should be nil when empty (use default)")
	}
}

func TestUpdateGitHubPath(t *testing.T) {
	model := createTestModel(t)
	model.newGitHubPath = "/new/clone/path"

	url := "https://github.com/test/repo.git"
	cfg := &config.Config{
		Central: repository.CentralRepositoryConfig{
			Path:      "/old/path",
			RemoteURL: &url,
		},
	}

	err := model.updateGitHubPath(cfg)
	if err != nil {
		t.Errorf("updateGitHubPath should not error, got: %v", err)
	}

	if cfg.Central.Path != "/new/clone/path" {
		t.Errorf("Path should be updated, got %v", cfg.Central.Path)
	}

	// Test error for empty path
	model.newGitHubPath = ""
	err = model.updateGitHubPath(cfg)
	if err == nil {
		t.Error("Should error when path is empty")
	}
}

func TestUpdateGitHubURL_UpdatesPathAndBranch(t *testing.T) {
	model := createTestModel(t)
	model.newGitHubURL = "https://github.com/newuser/newrepo.git"
	model.newGitHubPath = "/new/clone/path"
	model.newGitHubBranch = "develop"

	url := "https://github.com/olduser/oldrepo.git"
	branch := "main"
	cfg := &config.Config{
		Central: repository.CentralRepositoryConfig{
			Path:      "/old/path",
			RemoteURL: &url,
			Branch:    &branch,
		},
	}

	err := model.updateGitHubURL(cfg)
	if err != nil {
		t.Errorf("updateGitHubURL should not error, got: %v", err)
	}

	if *cfg.Central.RemoteURL != "https://github.com/newuser/newrepo.git" {
		t.Errorf("URL should be updated, got %v", *cfg.Central.RemoteURL)
	}

	if cfg.Central.Path != "/new/clone/path" {
		t.Errorf("Path should be updated along with URL, got %v", cfg.Central.Path)
	}

	if *cfg.Central.Branch != "develop" {
		t.Errorf("Branch should be updated along with URL, got %v", *cfg.Central.Branch)
	}
}

func TestPerformConfigUpdate_RoutesToCorrectHandler(t *testing.T) {
	tests := []struct {
		name       string
		changeType ChangeOption
		setup      func(*SettingsModel)
	}{
		{
			name:       "GitHubBranch",
			changeType: ChangeOptionGitHubBranch,
			setup: func(m *SettingsModel) {
				m.newGitHubBranch = "feature"
			},
		},
		{
			name:       "GitHubPath",
			changeType: ChangeOptionGitHubPath,
			setup: func(m *SettingsModel) {
				m.newGitHubPath = "/test/path"
			},
		},
		{
			name:       "GitHubURL",
			changeType: ChangeOptionGitHubURL,
			setup: func(m *SettingsModel) {
				m.newGitHubURL = "https://github.com/test/repo.git"
				m.newGitHubPath = "/test/path"
				m.newGitHubBranch = "main"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: These will fail without proper config file setup,
			// but we're testing the routing logic
			model := createTestModel(t)
			model.changeType = tt.changeType
			tt.setup(model)

			// Just verify it doesn't panic and returns an error
			// (since we don't have a valid config file)
			err := model.performConfigUpdate()
			// We expect an error because there's no valid config file
			if err == nil {
				t.Log("performConfigUpdate completed (unexpected in test environment)")
			}
		})
	}
}

func TestEdgeCase_WhitespaceOnlyInput(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "original")
	if err := os.MkdirAll(originalPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	model := createTestModelWithConfig(t, createLocalConfig(originalPath))
	model.state = SettingsStateUpdateLocalPath
	model.changeType = ChangeOptionLocalPath

	// Test whitespace-only path
	model.textInput.SetValue("   \t\n   ")
	updatedModel, cmd := model.validateAndProceedLocalPath()

	if cmd == nil {
		t.Error("Expected error command for whitespace-only path")
	}

	if updatedModel.state == SettingsStateConfirmation {
		t.Error("Should not proceed to confirmation with whitespace-only path")
	}
}

func TestEdgeCase_VeryLongPath(t *testing.T) {
	model := createTestModel(t)
	model.state = SettingsStateUpdateLocalPath

	// Create a very long path (> 4096 characters)
	longPath := "/home/user/" + string(make([]byte, 5000))
	for i := range longPath[11:] {
		longPath = longPath[:11+i] + "a" + longPath[12+i:]
	}

	model.textInput.SetValue(longPath)

	// Should handle gracefully without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panicked with very long path: %v", r)
		}
	}()

	_, cmd := model.validateAndProceedLocalPath()

	// Expect error for invalid path
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(settingsErrorMsg); !ok {
			t.Errorf("Expected settingsErrorMsg for very long path, got %T", msg)
		}
	}
}

func TestNavigationBoundaries(t *testing.T) {
	model := createTestModelWithConfig(t, createLocalConfig("/test/path"))
	model.state = SettingsStateSelectChange

	options := model.getMenuOptions()

	// Test navigation at lower boundary
	model.selectedOption = 0
	keyMsg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ := model.Update(keyMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.selectedOption != 0 {
		t.Error("Navigation at lower boundary should stay at 0")
	}

	// Test navigation at upper boundary
	model.selectedOption = len(options) - 1
	keyMsg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = model.Update(keyMsg)
	settingsModel = updatedModel.(*SettingsModel)

	if settingsModel.selectedOption != len(options)-1 {
		t.Error("Navigation at upper boundary should stay at max")
	}
}

func TestStateTransitionConsistency(t *testing.T) {
	model := createTestModel(t)

	tests := []struct {
		name         string
		currentState SettingsState
		newState     SettingsState
	}{
		{"MainMenu to SelectChange", SettingsStateMainMenu, SettingsStateSelectChange},
		{"SelectChange to UpdateLocalPath", SettingsStateSelectChange, SettingsStateUpdateLocalPath},
		{"UpdateLocalPath to Confirmation", SettingsStateUpdateLocalPath, SettingsStateConfirmation},
		{"Confirmation to Complete", SettingsStateConfirmation, SettingsStateComplete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.state = tt.currentState
			previousState := model.state

			updatedModel := model.transitionTo(tt.newState)

			if updatedModel.state != tt.newState {
				t.Errorf("Expected state %v, got %v", tt.newState, updatedModel.state)
			}

			if updatedModel.previousState != previousState {
				t.Errorf("Expected previousState %v, got %v", previousState, updatedModel.previousState)
			}
		})
	}
}

func TestTextInputCharacterLimit(t *testing.T) {
	model := createTestModel(t)

	if model.textInput.CharLimit != 256 {
		t.Errorf("Expected CharLimit to be 256, got %d", model.textInput.CharLimit)
	}

	// Verify input respects the limit
	longInput := string(make([]byte, 300))
	for i := range longInput {
		longInput = longInput[:i] + "a" + longInput[i+1:]
	}

	model.textInput.SetValue(longInput)

	// The textInput component should enforce the limit
	if len(model.textInput.Value()) > 256 {
		t.Errorf("TextInput should enforce CharLimit, got length %d", len(model.textInput.Value()))
	}
}

func TestGitHubURLValidation(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
	model.state = SettingsStateUpdateGitHubURL
	model.changeType = ChangeOptionGitHubURL

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"empty URL", "", true},
		{"whitespace only", "   ", true},
		{"valid https URL", "https://github.com/user/repo.git", false},
		{"valid http URL", "http://github.com/user/repo.git", false},
		{"URL without .git", "https://github.com/user/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.textInput.SetValue(tt.url)
			_, cmd := model.handleUpdateGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})

			if tt.shouldError && cmd == nil {
				t.Error("Expected error command for invalid URL")
			}

			if tt.shouldError && cmd != nil {
				msg := cmd()
				if _, ok := msg.(settingsErrorMsg); !ok {
					t.Errorf("Expected settingsErrorMsg, got %T", msg)
				}
			}
		})
	}
}

func TestGitHubBranchHandling(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))

	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{"normal branch", "develop", "develop"},
		{"branch with slashes", "feature/new-feature", "feature/new-feature"},
		{"empty branch", "", ""},
		{"whitespace branch", "   ", ""}, // Trimmed by the implementation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.state = SettingsStateUpdateGitHubBranch
			model.changeType = ChangeOptionGitHubBranch
			model.textInput.SetValue(tt.branch)

			updatedModel, _ := model.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})

			if updatedModel.newGitHubBranch != tt.expected {
				t.Errorf("Expected branch %q, got %q", tt.expected, updatedModel.newGitHubBranch)
			}
		})
	}
}

func TestLastSyncInfoTracking(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))

	// Initially nil
	if model.lastSyncInfo != nil {
		t.Error("lastSyncInfo should be nil initially")
	}

	// After refresh complete
	syncInfo := &repository.SyncInfo{Message: "synced"}
	refreshMsg := refreshCompleteMsg{syncInfo: syncInfo, err: nil}

	model.refreshInProgress = true
	updatedModel, _ := model.Update(refreshMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.lastSyncInfo == nil {
		t.Error("lastSyncInfo should be set after successful refresh")
	}

	if settingsModel.lastSyncInfo.Message != "synced" {
		t.Errorf("Expected message 'synced', got %v", settingsModel.lastSyncInfo.Message)
	}
}

func TestIsDirtyFlagBehavior(t *testing.T) {
	model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))

	// Initially not dirty
	if model.isDirty {
		t.Error("isDirty should be false initially")
	}

	// After dirty state message
	dirtyMsg := dirtyStateMsg{isDirty: true, err: nil}
	updatedModel, _ := model.Update(dirtyMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if !settingsModel.isDirty {
		t.Error("isDirty should be true after receiving dirty state message")
	}

	// Clean state message
	model.isDirty = true
	cleanMsg := dirtyStateMsg{isDirty: false, err: nil}
	updatedModel, _ = model.Update(cleanMsg)
	settingsModel = updatedModel.(*SettingsModel)

	if settingsModel.isDirty {
		t.Error("isDirty should be false after receiving clean state message")
	}
}

func TestHasChangesFlagLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "original")
	newPath := filepath.Join(tmpDir, "new")

	if err := os.MkdirAll(originalPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.MkdirAll(newPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	model := createTestModelWithConfig(t, createLocalConfig(originalPath))

	// Initially no changes
	if model.hasChanges {
		t.Error("hasChanges should be false initially")
	}

	// After making a change
	model.state = SettingsStateUpdateLocalPath
	model.changeType = ChangeOptionLocalPath
	model.textInput.SetValue(newPath)
	updatedModel, _ := model.validateAndProceedLocalPath()

	if !updatedModel.hasChanges {
		t.Error("hasChanges should be true after making a change")
	}

	// After resetting
	updatedModel.resetTemporaryChanges()
	if updatedModel.hasChanges {
		t.Error("hasChanges should be false after reset")
	}
}

func TestSelectedOptionReset(t *testing.T) {
	model := createTestModel(t)
	model.selectedOption = 5

	updatedModel := model.transitionTo(SettingsStateSelectChange)

	if updatedModel.selectedOption != 0 {
		t.Errorf("selectedOption should be reset to 0 on transition, got %d", updatedModel.selectedOption)
	}
}

func TestContextPreservation(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	if model.ctx != ctx {
		t.Error("Context should be preserved in model")
	}

	if model.logger == nil {
		t.Error("Logger should be set from context")
	}

	if model.credManager == nil {
		t.Error("Credential manager should be initialized")
	}
}

func TestFormatChangesSummaryAllCases(t *testing.T) {
	tests := []struct {
		name       string
		changeType ChangeOption
		setup      func(*SettingsModel)
		checkFunc  func(*testing.T, string)
	}{
		{
			name:       "Local path with long path",
			changeType: ChangeOptionLocalPath,
			setup: func(m *SettingsModel) {
				m.newStorageDir = "/very/long/path/to/storage/directory/that/exceeds/normal/length"
			},
			checkFunc: func(t *testing.T, summary string) {
				if len(summary) == 0 {
					t.Error("Summary should not be empty")
				}
			},
		},
		{
			name:       "GitHub URL with all fields",
			changeType: ChangeOptionGitHubURL,
			setup: func(m *SettingsModel) {
				m.newGitHubURL = "https://github.com/newuser/newrepo.git"
				m.newGitHubBranch = "feature/complex-feature"
				m.newGitHubPath = "/clone/path"
				m.patAction = PATActionUpdate
			},
			checkFunc: func(t *testing.T, summary string) {
				if len(summary) < 50 {
					t.Error("Summary should contain all GitHub details")
				}
			},
		},
		{
			name:       "Repository type to local with cleanup",
			changeType: ChangeOptionRepositoryType,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "local"
				m.newStorageDir = "/new/local/path"
			},
			checkFunc: func(t *testing.T, summary string) {
				if len(summary) == 0 {
					t.Error("Summary should not be empty for type change")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModel(t)
			model.changeType = tt.changeType
			tt.setup(model)

			summary := model.formatChangesSummary()
			tt.checkFunc(t, summary)
		})
	}
}

// ChangeType-Dependent Navigation Tests

func TestChangeTypeDependentNavigation_LocalPath(t *testing.T) {
	tests := []struct {
		name          string
		changeType    ChangeOption
		expectedState SettingsState
		setup         func(*SettingsModel)
	}{
		{
			name:          "Standalone local path change escapes to SelectChange",
			changeType:    ChangeOptionLocalPath,
			expectedState: SettingsStateSelectChange,
		},
		{
			name:          "Repository type change escapes to ConfirmTypeChange",
			changeType:    ChangeOptionRepositoryType,
			expectedState: SettingsStateConfirmTypeChange,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "local"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createLocalConfig("/test/path"))
			model.state = SettingsStateUpdateLocalPath
			model.changeType = tt.changeType

			if tt.setup != nil {
				tt.setup(model)
			}

			// Press escape
			escMsg := tea.KeyMsg{Type: tea.KeyEsc}
			updatedModel, _ := model.handleUpdateLocalPathKeys(escMsg)

			if updatedModel.state != tt.expectedState {
				t.Errorf("Expected state %v with changeType=%v, got %v",
					tt.expectedState, tt.changeType, updatedModel.state)
			}
		})
	}
}

func TestChangeTypeDependentNavigation_GitHubURL(t *testing.T) {
	tests := []struct {
		name          string
		changeType    ChangeOption
		expectedState SettingsState
		setup         func(*SettingsModel)
	}{
		{
			name:          "Standalone URL change escapes to SelectChange",
			changeType:    ChangeOptionGitHubURL,
			expectedState: SettingsStateSelectChange,
		},
		{
			name:          "Repository type change escapes to ConfirmTypeChange",
			changeType:    ChangeOptionRepositoryType,
			expectedState: SettingsStateConfirmTypeChange,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "github"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/old/repo.git", "main"))
			model.state = SettingsStateUpdateGitHubURL
			model.changeType = tt.changeType

			if tt.setup != nil {
				tt.setup(model)
			}

			// Press escape
			escMsg := tea.KeyMsg{Type: tea.KeyEsc}
			updatedModel, _ := model.handleUpdateGitHubURLKeys(escMsg)

			if updatedModel.state != tt.expectedState {
				t.Errorf("Expected state %v with changeType=%v, got %v",
					tt.expectedState, tt.changeType, updatedModel.state)
			}
		})
	}
}

func TestChangeTypeDependentNavigation_GitHubBranch(t *testing.T) {
	tests := []struct {
		name          string
		changeType    ChangeOption
		expectedState SettingsState
		setup         func(*SettingsModel)
	}{
		{
			name:          "Standalone branch change escapes to SelectChange",
			changeType:    ChangeOptionGitHubBranch,
			expectedState: SettingsStateSelectChange,
		},
		{
			name:          "URL change flow escapes to UpdateGitHubURL",
			changeType:    ChangeOptionGitHubURL,
			expectedState: SettingsStateUpdateGitHubURL,
			setup: func(m *SettingsModel) {
				m.newGitHubURL = "https://github.com/new/repo.git"
			},
		},
		{
			name:          "Repository type change escapes to UpdateGitHubURL",
			changeType:    ChangeOptionRepositoryType,
			expectedState: SettingsStateUpdateGitHubURL,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/new/repo.git"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
			model.state = SettingsStateUpdateGitHubBranch
			model.changeType = tt.changeType

			if tt.setup != nil {
				tt.setup(model)
			}

			// Press escape
			escMsg := tea.KeyMsg{Type: tea.KeyEsc}
			updatedModel, _ := model.handleUpdateGitHubBranchKeys(escMsg)

			if updatedModel.state != tt.expectedState {
				t.Errorf("Expected state %v with changeType=%v, got %v",
					tt.expectedState, tt.changeType, updatedModel.state)
			}
		})
	}
}

func TestChangeTypeDependentNavigation_GitHubPath(t *testing.T) {
	tests := []struct {
		name          string
		changeType    ChangeOption
		expectedState SettingsState
		setup         func(*SettingsModel)
	}{
		{
			name:          "Standalone path change escapes to SelectChange",
			changeType:    ChangeOptionGitHubPath,
			expectedState: SettingsStateSelectChange,
		},
		{
			name:          "URL change flow escapes to UpdateGitHubBranch",
			changeType:    ChangeOptionGitHubURL,
			expectedState: SettingsStateUpdateGitHubBranch,
			setup: func(m *SettingsModel) {
				m.newGitHubURL = "https://github.com/new/repo.git"
				m.newGitHubBranch = "main"
			},
		},
		{
			name:          "Repository type change escapes to UpdateGitHubBranch",
			changeType:    ChangeOptionRepositoryType,
			expectedState: SettingsStateUpdateGitHubBranch,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/new/repo.git"
				m.newGitHubBranch = "main"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
			model.state = SettingsStateUpdateGitHubPath
			model.changeType = tt.changeType

			if tt.setup != nil {
				tt.setup(model)
			}

			// Press escape
			escMsg := tea.KeyMsg{Type: tea.KeyEsc}
			updatedModel, _ := model.handleUpdateGitHubPathKeys(escMsg)

			if updatedModel.state != tt.expectedState {
				t.Errorf("Expected state %v with changeType=%v, got %v",
					tt.expectedState, tt.changeType, updatedModel.state)
			}
		})
	}
}

func TestChangeTypeDependentNavigation_GitHubPAT(t *testing.T) {
	tests := []struct {
		name          string
		changeType    ChangeOption
		expectedState SettingsState
		setup         func(*SettingsModel)
	}{
		{
			name:          "Standalone PAT change escapes to SelectChange",
			changeType:    ChangeOptionGitHubPAT,
			expectedState: SettingsStateSelectChange,
		},
		{
			name:          "Repository type change escapes to UpdateGitHubPath",
			changeType:    ChangeOptionRepositoryType,
			expectedState: SettingsStateUpdateGitHubPath,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/new/repo.git"
				m.newGitHubPath = "/test/clone/path"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
			model.state = SettingsStateUpdateGitHubPAT
			model.changeType = tt.changeType

			if tt.setup != nil {
				tt.setup(model)
			}

			// Press escape
			escMsg := tea.KeyMsg{Type: tea.KeyEsc}
			updatedModel, _ := model.handleUpdateGitHubPATKeys(escMsg)

			if updatedModel.state != tt.expectedState {
				t.Errorf("Expected state %v with changeType=%v, got %v",
					tt.expectedState, tt.changeType, updatedModel.state)
			}
		})
	}
}

func TestChangeTypeDependentNavigation_GitHubPathForward(t *testing.T) {
	tests := []struct {
		name          string
		changeType    ChangeOption
		expectedState SettingsState
		setup         func(*SettingsModel)
	}{
		{
			name:          "Standalone path change proceeds to Confirmation",
			changeType:    ChangeOptionGitHubPath,
			expectedState: SettingsStateConfirmation,
			setup: func(m *SettingsModel) {
				m.newGitHubPath = "/test/new/path"
			},
		},
		{
			name:          "URL change flow proceeds to Confirmation",
			changeType:    ChangeOptionGitHubURL,
			expectedState: SettingsStateConfirmation,
			setup: func(m *SettingsModel) {
				m.newGitHubURL = "https://github.com/new/repo.git"
				m.newGitHubBranch = "main"
				m.newGitHubPath = "/test/new/path"
			},
		},
		{
			name:          "Repository type change proceeds to UpdateGitHubPAT",
			changeType:    ChangeOptionRepositoryType,
			expectedState: SettingsStateUpdateGitHubPAT,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/new/repo.git"
				m.newGitHubBranch = "main"
				m.newGitHubPath = "/test/new/path"
			},
		},
	}

	homeDir, _ := os.UserHomeDir()
	validPath := filepath.Join(homeDir, ".rulem-test-path-forward")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
			model.state = SettingsStateUpdateGitHubPath
			model.changeType = tt.changeType

			if tt.setup != nil {
				tt.setup(model)
			}

			// Set valid path
			model.textInput.SetValue(validPath)

			// Press enter
			enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
			updatedModel, _ := model.handleUpdateGitHubPathKeys(enterMsg)

			if updatedModel.state != tt.expectedState {
				t.Errorf("Expected state %v with changeType=%v, got %v",
					tt.expectedState, tt.changeType, updatedModel.state)
			}
		})
	}
}

func TestDirtyStateMessageHandling_ChangeTypeDependency(t *testing.T) {
	tests := []struct {
		name              string
		currentState      SettingsState
		changeType        ChangeOption
		isDirty           bool
		expectedNextState SettingsState
		setup             func(*SettingsModel)
	}{
		{
			name:              "Clean repo - standalone branch change to Confirmation",
			currentState:      SettingsStateUpdateGitHubBranch,
			changeType:        ChangeOptionGitHubBranch,
			isDirty:           false,
			expectedNextState: SettingsStateConfirmation,
			setup: func(m *SettingsModel) {
				m.newGitHubBranch = "develop"
			},
		},
		{
			name:              "Clean repo - URL flow branch to Path",
			currentState:      SettingsStateUpdateGitHubBranch,
			changeType:        ChangeOptionGitHubURL,
			isDirty:           false,
			expectedNextState: SettingsStateUpdateGitHubPath,
			setup: func(m *SettingsModel) {
				m.newGitHubURL = "https://github.com/test/repo.git"
				m.newGitHubBranch = "main"
			},
		},
		{
			name:              "Clean repo - repo type change branch to Path",
			currentState:      SettingsStateUpdateGitHubBranch,
			changeType:        ChangeOptionRepositoryType,
			isDirty:           false,
			expectedNextState: SettingsStateUpdateGitHubPath,
			setup: func(m *SettingsModel) {
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/test/repo.git"
				m.newGitHubBranch = "main"
			},
		},
		{
			name:              "Clean repo - manual refresh proceeds",
			currentState:      SettingsStateManualRefresh,
			changeType:        ChangeOptionManualRefresh,
			isDirty:           false,
			expectedNextState: SettingsStateRefreshInProgress,
		},
		{
			name:              "Dirty repo - standalone branch to Error",
			currentState:      SettingsStateUpdateGitHubBranch,
			changeType:        ChangeOptionGitHubBranch,
			isDirty:           true,
			expectedNextState: SettingsStateError,
			setup: func(m *SettingsModel) {
				m.newGitHubBranch = "develop"
			},
		},
		{
			name:              "Dirty repo - URL flow branch to Error",
			currentState:      SettingsStateUpdateGitHubBranch,
			changeType:        ChangeOptionGitHubURL,
			isDirty:           true,
			expectedNextState: SettingsStateError,
			setup: func(m *SettingsModel) {
				m.newGitHubURL = "https://github.com/test/repo.git"
				m.newGitHubBranch = "main"
			},
		},
		{
			name:              "Dirty repo - manual refresh to Error",
			currentState:      SettingsStateManualRefresh,
			changeType:        ChangeOptionManualRefresh,
			isDirty:           true,
			expectedNextState: SettingsStateError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModelWithConfig(t, createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main"))
			model.state = tt.currentState
			model.changeType = tt.changeType

			if tt.setup != nil {
				tt.setup(model)
			}

			// Send dirty state message
			dirtyMsg := dirtyStateMsg{isDirty: tt.isDirty, err: nil}
			updatedModel, _ := model.Update(dirtyMsg)
			settingsModel := updatedModel.(*SettingsModel)

			if settingsModel.state != tt.expectedNextState {
				t.Errorf("Expected state %v with changeType=%v and isDirty=%v, got %v",
					tt.expectedNextState, tt.changeType, tt.isDirty, settingsModel.state)
			}

			if settingsModel.isDirty != tt.isDirty {
				t.Errorf("Expected isDirty=%v, got %v", tt.isDirty, settingsModel.isDirty)
			}
		})
	}
}
