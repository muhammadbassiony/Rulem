package settingsmenu

import (
	"os"
	"path/filepath"
	"testing"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/tui/helpers"

	tea "github.com/charmbracelet/bubbletea"
)

// Integration tests for complete settings menu flows

func TestSettingsMenuBasicFlow(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)

	// Test initial state
	if model.state != SettingsStateMainMenu {
		t.Errorf("Expected initial state SettingsStateMainMenu, got %v", model.state)
	}

	// Initialize
	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}

	// Load config
	testConfig := createLocalConfig("/test/path")
	configMsg := config.LoadConfigMsg{Config: testConfig, Error: nil}
	updatedModel, _ := model.Update(configMsg)
	model = updatedModel.(*SettingsModel)

	if model.currentConfig == nil {
		t.Error("Config should be loaded")
	}
}

func TestCompleteFlow_LocalPathUpdate(t *testing.T) {
	// Create temporary directories for testing
	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "original")
	newPath := filepath.Join(tmpDir, "new")

	// Create both directories
	if err := os.MkdirAll(originalPath, 0755); err != nil {
		t.Fatalf("Failed to create original test directory: %v", err)
	}

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig(originalPath)

	// Start at main menu
	if model.state != SettingsStateMainMenu {
		t.Fatalf("Expected initial state MainMenu, got %v", model.state)
	}

	// Navigate to select change
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateSelectChange {
		t.Fatalf("Expected SelectChange state, got %v", model.state)
	}

	// Select local path option (index 1 after repository type)
	model.selectedOption = 1
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateLocalPath {
		t.Fatalf("Expected UpdateLocalPath state, got %v", model.state)
	}

	if model.changeType != ChangeOptionLocalPath {
		t.Errorf("Expected changeType LocalPath, got %v", model.changeType)
	}

	// Enter new path (use valid path that exists)
	model.textInput.SetValue(newPath)
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should transition to confirmation after successful validation
	if model.state != SettingsStateConfirmation {
		t.Fatalf("Expected Confirmation state after validation, got %v", model.state)
	}

	// Verify that hasChanges flag is set
	if !model.hasChanges {
		t.Error("Expected hasChanges to be true")
	}

	// Verify new path is stored
	if model.newStorageDir != newPath {
		t.Errorf("Expected newStorageDir to be %s, got %s", newPath, model.newStorageDir)
	}

	// Confirm the change
	yesMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, cmd := model.Update(yesMsg)
	model = updatedModel.(*SettingsModel)
	if model.state != SettingsStateConfirmation {
		t.Fatalf("Expected Confirmation state after saving, got %v", model.state)
	}

	// Command should be returned to save changes
	if cmd == nil {
		t.Error("Expected command to be returned for saving changes")
	}

	// We cannot simulate save completion
	// as it involves file system operations and t.TempDir creates outside home dir
	// which breaks our security checks

}

func TestCompleteFlow_RepositoryTypeSwitch_LocalToGitHub(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/original/path")
	model.state = SettingsStateMainMenu

	// Navigate to select change
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateSelectChange {
		t.Fatalf("Expected SelectChange state, got %v", model.state)
	}

	// Select repository type (index 0)
	model.selectedOption = 0
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateConfirmTypeChange {
		t.Fatalf("Expected ConfirmTypeChange state, got %v", model.state)
	}

	// Confirm type change
	yesMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, _ = model.Update(yesMsg)
	model = updatedModel.(*SettingsModel)

	// if model.state != SettingsStateRepositoryTypeSelect {
	// 	t.Fatalf("Expected RepositoryTypeSelect state, got %v", model.state)
	// }

	// // Select GitHub (index 1)
	// model.selectedOption = 1
	// updatedModel, _ = model.Update(enterMsg)
	// model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateGitHubURL {
		t.Fatalf("Expected UpdateGitHubURL state, got %v", model.state)
	}

	if model.newRepositoryType != "github" {
		t.Errorf("Expected newRepositoryType github, got %v", model.newRepositoryType)
	}
}

func TestCompleteFlow_RepositoryTypeSwitch_GitHubToLocal(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateMainMenu

	// Navigate to select change
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Select repository type (index 0)
	model.selectedOption = 0
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateConfirmTypeChange {
		t.Fatalf("Expected ConfirmTypeChange state, got %v", model.state)
	}

	// Confirm type change
	yesMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, _ = model.Update(yesMsg)
	model = updatedModel.(*SettingsModel)

	// if model.state != SettingsStateRepositoryTypeSelect {
	// 	t.Fatalf("Expected RepositoryTypeSelect state, got %v", model.state)
	// }

	// // Select Local (index 0)
	// model.selectedOption = 0
	// updatedModel, _ = model.Update(enterMsg)
	// model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateLocalPath {
		t.Fatalf("Expected UpdateLocalPath state, got %v", model.state)
	}

	if model.newRepositoryType != "local" {
		t.Errorf("Expected newRepositoryType local, got %v", model.newRepositoryType)
	}
}

func TestCompleteFlow_GitHubURLUpdate(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/old/repo.git", "main")
	model.state = SettingsStateSelectChange

	// Find and select GitHub URL option
	options := model.getMenuOptions()
	urlIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionGitHubURL {
			urlIndex = i
			break
		}
	}

	if urlIndex == -1 {
		t.Fatal("GitHub URL option not found")
	}

	model.selectedOption = urlIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateGitHubURL {
		t.Fatalf("Expected UpdateGitHubURL state, got %v", model.state)
	}

	// Enter new URL
	model.textInput.SetValue("https://github.com/new/repo.git")
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should transition to confirmation
	if model.state != SettingsStateConfirmation {
		t.Fatalf("Expected Confirmation state, got %v", model.state)
	}

	if model.newGitHubURL != "https://github.com/new/repo.git" {
		t.Errorf("Expected newGitHubURL to be set, got %v", model.newGitHubURL)
	}
}

func TestCompleteFlow_GitHubPATUpdate(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateSelectChange

	// Find and select GitHub PAT option
	options := model.getMenuOptions()
	patIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionGitHubPAT {
			patIndex = i
			break
		}
	}

	if patIndex == -1 {
		t.Fatal("GitHub PAT option not found")
	}

	model.selectedOption = patIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateGitHubPAT {
		t.Fatalf("Expected UpdateGitHubPAT state, got %v", model.state)
	}

	// Enter new PAT
	model.textInput.SetValue("ghp_newtoken123456789")
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should transition to confirmation
	if model.state != SettingsStateConfirmation {
		t.Fatalf("Expected Confirmation state, got %v", model.state)
	}

	if model.patAction != PATActionUpdate {
		t.Errorf("Expected patAction Update, got %v", model.patAction)
	}
}

func TestCompleteFlow_GitHubPATRemoval(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateUpdateGitHubPAT

	// Press Ctrl+D to remove PAT
	ctrlDMsg := tea.KeyMsg{Type: tea.KeyCtrlD}
	updatedModel, _ := model.Update(ctrlDMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateConfirmation {
		t.Fatalf("Expected Confirmation state, got %v", model.state)
	}

	if model.patAction != PATActionRemove {
		t.Errorf("Expected patAction Remove, got %v", model.patAction)
	}

	if model.newGitHubPAT != "" {
		t.Errorf("Expected newGitHubPAT to be empty, got %v", model.newGitHubPAT)
	}
}

func TestCompleteFlow_ManualRefresh(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateSelectChange

	// Find and select Manual Refresh option
	options := model.getMenuOptions()
	refreshIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionManualRefresh {
			refreshIndex = i
			break
		}
	}

	if refreshIndex == -1 {
		t.Fatal("Manual Refresh option not found")
	}

	model.selectedOption = refreshIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateManualRefresh {
		t.Fatalf("Expected ManualRefresh state, got %v", model.state)
	}

	// Confirm refresh
	yesMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, cmd := model.Update(yesMsg)
	model = updatedModel.(*SettingsModel)

	if cmd == nil {
		t.Error("Refresh confirmation should return a command")
	}

	if model.state != SettingsStateRefreshInProgress {
		t.Fatalf("Expected RefreshInProgress state, got %v", model.state)
	}

	if !model.refreshInProgress {
		t.Error("refreshInProgress flag should be set")
	}
}

func TestBackNavigation(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")

	// Test escape from SelectChange
	model.state = SettingsStateSelectChange
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := model.Update(escMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateMainMenu {
		t.Errorf("Escape from SelectChange should go to MainMenu, got %v", model.state)
	}

	// Test escape from ConfirmTypeChange
	model.state = SettingsStateConfirmTypeChange
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateSelectChange {
		t.Errorf("Escape from ConfirmTypeChange should go to SelectChange, got %v", model.state)
	}

	// Test escape from UpdateLocalPath
	model.state = SettingsStateUpdateLocalPath
	model.changeType = ChangeOptionLocalPath
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateSelectChange {
		t.Errorf("Escape from UpdateLocalPath should go to SelectChange, got %v", model.state)
	}
}

func TestErrorRecovery(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")

	// Transition to error state
	errorMsg := settingsErrorMsg{err: &testError{"test error"}}
	updatedModel, _ := model.Update(errorMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateError {
		t.Fatalf("Expected Error state, got %v", model.state)
	}

	// Press any key to continue
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if cmd == nil {
		t.Error("Error state should allow navigation away")
	}
}

func TestViewsRenderWithoutPanic(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")

	states := []SettingsState{
		SettingsStateMainMenu,
		SettingsStateSelectChange,
		SettingsStateConfirmTypeChange,
		SettingsStateUpdateLocalPath,
		SettingsStateUpdateGitHubPAT,
		SettingsStateUpdateGitHubURL,
		SettingsStateUpdateGitHubBranch,
		SettingsStateUpdateGitHubPath,
		SettingsStateManualRefresh,
		SettingsStateRefreshInProgress,
		SettingsStateConfirmation,
		SettingsStateComplete,
		SettingsStateError,
	}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			model.state = state

			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("View panicked for state %v: %v", state, r)
				}
			}()

			view := model.View()
			if view == "" {
				t.Errorf("View for state %v is empty", state)
			}
		})
	}
}
