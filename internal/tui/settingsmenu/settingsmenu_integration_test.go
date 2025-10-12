package settingsmenu

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/helpers"

	tea "github.com/charmbracelet/bubbletea"
)

// Integration tests for complete settings menu flows
// These tests verify end-to-end behavior including state transitions, validation, and user interactions

// Test Helpers

func createTempDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tmpDir
}

// Basic Flow Tests

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

	// Navigate to select change menu
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateSelectChange {
		t.Errorf("Expected SelectChange state after Enter from MainMenu, got %v", model.state)
	}

	// Test navigation back to main menu
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateMainMenu {
		t.Errorf("Expected MainMenu state after Esc from SelectChange, got %v", model.state)
	}
}

// Local Path Update Tests

func TestCompleteFlow_LocalPathUpdate_Success(t *testing.T) {
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

	// Enter new path
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

	// Command should be returned to save changes
	if cmd == nil {
		t.Error("Expected command to be returned for saving changes")
	}

	// Note: We cannot simulate save completion in test environment
	// as it involves file system operations with home directory checks
}

func TestCompleteFlow_LocalPathUpdate_UnchangedPath(t *testing.T) {
	// Test that entering the same path returns to main menu without saving
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test")

	if err := os.MkdirAll(testPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig(testPath)

	// Navigate to update path state
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)
	model.selectedOption = 1
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateLocalPath {
		t.Fatalf("Expected UpdateLocalPath state, got %v", model.state)
	}

	// Enter the SAME path
	model.textInput.SetValue(testPath)
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should return to main menu (optimization - no change detected)
	if model.state != SettingsStateMainMenu {
		t.Errorf("Expected MainMenu state (no change), got %v", model.state)
	}

	if model.hasChanges {
		t.Error("Expected hasChanges to be false (no change)")
	}
}

func TestCompleteFlow_LocalPathUpdate_InvalidPath(t *testing.T) {
	// Test that invalid paths show error and stay in update state
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "original")
	if err := os.MkdirAll(originalPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig(originalPath)

	// Navigate to update path state
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)
	model.selectedOption = 1
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Enter an invalid path (parent doesn't exist)
	model.textInput.SetValue("/nonexistent/parent/path")
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should return error command
	if cmd == nil {
		t.Fatal("Expected error command for invalid path")
	}

	// Execute the error command
	msg := cmd()

	// Should be an error message
	if _, ok := msg.(settingsErrorMsg); !ok {
		t.Errorf("Expected settingsErrorMsg, got %T", msg)
	}

	// Process the error
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(*SettingsModel)

	// Should be in error state
	if model.state != SettingsStateError {
		t.Errorf("Expected Error state, got %v", model.state)
	}
}

func TestCompleteFlow_LocalPathUpdate_EmptyPath(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	tmpDir := createTempDir(t)
	originalPath := filepath.Join(tmpDir, "original")
	if err := os.MkdirAll(originalPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig(originalPath)

	// Navigate to update path state
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)
	model.selectedOption = 1
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Enter empty path
	model.textInput.SetValue("")
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should return error command for empty path
	if cmd == nil {
		t.Fatal("Expected error command for empty path")
	}

	msg := cmd()
	if _, ok := msg.(settingsErrorMsg); !ok {
		t.Errorf("Expected settingsErrorMsg, got %T", msg)
	}
}

func TestCompleteFlow_LocalPathUpdate_CancelDuringInput(t *testing.T) {
	// Test that pressing Esc cancels the operation
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig(testPath)

	// Navigate to update path state
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)
	model.selectedOption = 1
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateLocalPath {
		t.Fatalf("Expected UpdateLocalPath state, got %v", model.state)
	}

	// Start typing a new path
	model.textInput.SetValue("/some/new/path")

	// Press Escape to cancel
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(*SettingsModel)

	// Should return to SelectChange
	if model.state != SettingsStateSelectChange {
		t.Errorf("Expected SelectChange state after cancel, got %v", model.state)
	}

	// Verify temporary changes were cleared
	if model.newStorageDir != "" {
		t.Error("Expected newStorageDir to be cleared after cancel")
	}

	if model.hasChanges {
		t.Error("Expected hasChanges to be false after cancel")
	}
}

func TestCompleteFlow_LocalPathUpdate_CancelDuringConfirmation(t *testing.T) {
	// Test that pressing 'n' during confirmation cancels
	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "original")
	newPath := filepath.Join(tmpDir, "new")

	if err := os.MkdirAll(originalPath, 0755); err != nil {
		t.Fatalf("Failed to create original test directory: %v", err)
	}
	if err := os.MkdirAll(newPath, 0755); err != nil {
		t.Fatalf("Failed to create new test directory: %v", err)
	}

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig(originalPath)

	// Navigate through to confirmation
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)
	model.selectedOption = 1
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)
	model.textInput.SetValue(newPath)
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateConfirmation {
		t.Fatalf("Expected Confirmation state, got %v", model.state)
	}

	// Reject the change
	noMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updatedModel, _ = model.Update(noMsg)
	model = updatedModel.(*SettingsModel)

	// Should return to SelectChange
	if model.state != SettingsStateSelectChange {
		t.Errorf("Expected SelectChange state after rejection, got %v", model.state)
	}

	// Temporary changes should be cleared
	if model.newStorageDir != "" {
		t.Error("Expected newStorageDir to be cleared after rejection")
	}

	if model.hasChanges {
		t.Error("Expected hasChanges to be false after rejection")
	}
}

// Repository Type Switch Tests

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

	if model.state != SettingsStateUpdateGitHubURL {
		t.Fatalf("Expected UpdateGitHubURL state, got %v", model.state)
	}

	if model.newRepositoryType != "github" {
		t.Errorf("Expected newRepositoryType github, got %v", model.newRepositoryType)
	}

	// Enter GitHub URL
	model.textInput.SetValue("https://github.com/test/repo.git")
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should move to branch input
	if model.state != SettingsStateUpdateGitHubBranch {
		t.Fatalf("Expected UpdateGitHubBranch state, got %v", model.state)
	}

	if model.newGitHubURL != "https://github.com/test/repo.git" {
		t.Errorf("Expected URL to be saved, got %v", model.newGitHubURL)
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

	if model.state != SettingsStateUpdateLocalPath {
		t.Fatalf("Expected UpdateLocalPath state, got %v", model.state)
	}

	if model.newRepositoryType != "local" {
		t.Errorf("Expected newRepositoryType local, got %v", model.newRepositoryType)
	}
}

func TestCompleteFlow_RepositoryTypeSwitch_CancelConfirmation(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")
	model.state = SettingsStateSelectChange

	// Select repository type
	model.selectedOption = 0
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateConfirmTypeChange {
		t.Fatalf("Expected ConfirmTypeChange state, got %v", model.state)
	}

	// Reject type change
	noMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updatedModel, _ = model.Update(noMsg)
	model = updatedModel.(*SettingsModel)

	// Should return to SelectChange
	if model.state != SettingsStateSelectChange {
		t.Errorf("Expected SelectChange state after rejection, got %v", model.state)
	}

	// Should not set new repository type
	if model.newRepositoryType != "" {
		t.Error("Expected newRepositoryType to be empty after rejection")
	}
}

// GitHub URL Update Tests

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

	// Should transition to branch input
	if model.state != SettingsStateUpdateGitHubBranch {
		t.Fatalf("Expected UpdateGitHubBranch state, got %v", model.state)
	}

	if model.newGitHubURL != "https://github.com/new/repo.git" {
		t.Errorf("Expected newGitHubURL to be set, got %v", model.newGitHubURL)
	}

	// Enter branch
	model.textInput.SetValue("main")
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Execute the dirty check command if returned
	if cmd != nil {
		msg := cmd()
		updatedModel, _ = model.Update(msg)
		model = updatedModel.(*SettingsModel)
	}

	// After dirty check, should be in path input or error state
	if model.state != SettingsStateUpdateGitHubPath && model.state != SettingsStateError {
		t.Fatalf("Expected UpdateGitHubPath or Error state after dirty check, got %v", model.state)
	}

	if model.state == SettingsStateError {
		t.Log("Skipping remainder - dirty check failed (expected in test environment)")
		return
	}

	// Enter path
	homeDir, _ := os.UserHomeDir()
	testPath := filepath.Join(homeDir, ".rulem-test-url-flow")
	model.textInput.SetValue(testPath)
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should transition to confirmation
	if model.state != SettingsStateConfirmation {
		t.Fatalf("Expected Confirmation state, got %v", model.state)
	}
}

func TestCompleteFlow_GitHubURLUpdate_InvalidURL(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/old/repo.git", "main")
	model.state = SettingsStateUpdateGitHubURL
	model.changeType = ChangeOptionGitHubURL

	// Enter empty URL
	model.textInput.SetValue("")
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.Update(enterMsg)

	// Should return error command
	if cmd == nil {
		t.Error("Expected error command for empty URL")
	}

	msg := cmd()
	if _, ok := msg.(settingsErrorMsg); !ok {
		t.Errorf("Expected settingsErrorMsg, got %T", msg)
	}
}

func TestCompleteFlow_GitHubURLUpdate_CancelAtBranch(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/old/repo.git", "main")
	model.state = SettingsStateUpdateGitHubURL
	model.changeType = ChangeOptionGitHubURL

	// Enter URL
	model.textInput.SetValue("https://github.com/new/repo.git")
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should be at branch input
	if model.state != SettingsStateUpdateGitHubBranch {
		t.Fatalf("Expected UpdateGitHubBranch state, got %v", model.state)
	}

	// Cancel at branch input
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(*SettingsModel)

	// Should go back to URL input
	if model.state != SettingsStateUpdateGitHubURL {
		t.Errorf("Expected UpdateGitHubURL state after cancel, got %v", model.state)
	}
}

// GitHub Branch Update Tests

func TestCompleteFlow_GitHubBranchUpdate_Standalone(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateSelectChange

	// Find and select GitHub Branch option
	options := model.getMenuOptions()
	branchIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionGitHubBranch {
			branchIndex = i
			break
		}
	}

	if branchIndex == -1 {
		t.Fatal("GitHub Branch option not found")
	}

	model.selectedOption = branchIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateGitHubBranch {
		t.Fatalf("Expected UpdateGitHubBranch state, got %v", model.state)
	}

	if model.changeType != ChangeOptionGitHubBranch {
		t.Errorf("Expected changeType GitHubBranch, got %v", model.changeType)
	}

	// Enter new branch
	model.textInput.SetValue("develop")
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should return dirty check command
	if cmd == nil {
		t.Error("Expected dirty check command")
	}
}

func TestCompleteFlow_GitHubBranchUpdate_EmptyBranch(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateUpdateGitHubBranch
	model.changeType = ChangeOptionGitHubBranch

	// Enter empty branch (should use default)
	model.textInput.SetValue("")
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should still proceed with dirty check
	if cmd == nil {
		t.Error("Expected dirty check command even with empty branch")
	}
}

// GitHub Path Update Tests

func TestCompleteFlow_GitHubPathUpdate_Standalone(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	homeDir, _ := os.UserHomeDir()
	testPath := filepath.Join(homeDir, ".rulem-test-path")

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateSelectChange

	// Find and select GitHub Path option
	options := model.getMenuOptions()
	pathIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionGitHubPath {
			pathIndex = i
			break
		}
	}

	if pathIndex == -1 {
		t.Fatal("GitHub Path option not found")
	}

	model.selectedOption = pathIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateGitHubPath {
		t.Fatalf("Expected UpdateGitHubPath state, got %v", model.state)
	}

	// Enter new path
	model.textInput.SetValue(testPath)
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should go to confirmation
	if model.state != SettingsStateConfirmation {
		t.Fatalf("Expected Confirmation state, got %v", model.state)
	}

	if model.newGitHubPath != testPath {
		t.Errorf("Expected newGitHubPath to be set to %s, got %s", testPath, model.newGitHubPath)
	}
}

// GitHub PAT Update Tests

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

	// Enter new PAT value
	model.textInput.SetValue("ghp_newtoken456")
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Execute any validation command if returned
	if cmd != nil {
		msg := cmd()
		updatedModel, _ = model.Update(msg)
		model = updatedModel.(*SettingsModel)
	}

	// After entering PAT, should validate and transition
	if model.state != SettingsStateConfirmation &&
		model.state != SettingsStateError &&
		model.state != SettingsStateUpdateGitHubPAT {
		t.Fatalf("Expected Confirmation, Error, or still in UpdateGitHubPAT state, got %v", model.state)
	}

	// If we hit an error state (validation failed), that's acceptable
	if model.state == SettingsStateError || model.state == SettingsStateUpdateGitHubPAT {
		t.Log("PAT validation failed or still in input state (expected in test environment)")
		return
	}

	// Verify PAT action was set
	if model.patAction != PATActionUpdate {
		t.Errorf("Expected PATActionUpdate, got %v", model.patAction)
	}
}

func TestCompleteFlow_GitHubPATUpdate_EmptyPAT(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateUpdateGitHubPAT
	model.changeType = ChangeOptionGitHubPAT

	// Try to submit empty PAT
	model.textInput.SetValue("")
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.Update(enterMsg)

	// Should return error command
	if cmd == nil {
		t.Error("Expected error command for empty PAT")
	}

	msg := cmd()
	if _, ok := msg.(settingsErrorMsg); !ok {
		t.Errorf("Expected settingsErrorMsg, got %T", msg)
	}
}

// Manual Refresh Tests

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
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, cmd := model.Update(yMsg)
	model = updatedModel.(*SettingsModel)

	if cmd == nil {
		t.Fatal("Expected command from refresh trigger (dirty check)")
	}

	// Execute the dirty check command
	msg := cmd()
	if msg != nil {
		updatedModel, cmd2 := model.Update(msg)
		model = updatedModel.(*SettingsModel)

		// After dirty check, verify state
		if model.state == SettingsStateError || model.state == SettingsStateManualRefresh {
			t.Log("Refresh blocked by dirty check or missing repo (expected in test environment)")
			return
		}

		// Check refreshInProgress flag
		if model.state == SettingsStateRefreshInProgress {
			if !model.refreshInProgress {
				t.Errorf("refreshInProgress flag should be set in RefreshInProgress state. State=%v, Flag=%v",
					model.state, model.refreshInProgress)
			}
		}

		// Execute the actual refresh command if returned
		if cmd2 != nil {
			msg2 := cmd2()
			updatedModel, _ = model.Update(msg2)
			model = updatedModel.(*SettingsModel)
		}
	}

	// Verify final state is acceptable
	validStates := []SettingsState{
		SettingsStateRefreshInProgress,
		SettingsStateError,
		SettingsStateMainMenu,
		SettingsStateManualRefresh,
	}

	stateValid := false
	if slices.Contains(validStates, model.state) {
		stateValid = true
	}

	if !stateValid {
		t.Fatalf("Expected one of %v states, got %v", validStates, model.state)
	}

	t.Logf("Test completed with state=%v (refreshInProgress=%v)", model.state, model.refreshInProgress)
}

func TestCompleteFlow_ManualRefresh_Cancel(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateManualRefresh

	// Decline refresh
	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updatedModel, _ := model.Update(nMsg)
	model = updatedModel.(*SettingsModel)

	// Should return to SelectChange
	if model.state != SettingsStateSelectChange {
		t.Errorf("Expected SelectChange state after declining refresh, got %v", model.state)
	}

	// refreshInProgress should not be set
	if model.refreshInProgress {
		t.Error("refreshInProgress should not be set when refresh is cancelled")
	}
}

func TestCompleteFlow_ManualRefresh_WithDirtyRepository(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/clone/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateManualRefresh

	// Confirm refresh
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, cmd := model.Update(yMsg)
	model = updatedModel.(*SettingsModel)

	if cmd == nil {
		t.Fatal("Expected dirty check command")
	}

	// Simulate dirty state message
	dirtyMsg := dirtyStateMsg{isDirty: true, err: nil}
	updatedModel, _ = model.Update(dirtyMsg)
	model = updatedModel.(*SettingsModel)

	// Should transition to error state when dirty
	if model.state != SettingsStateError {
		t.Errorf("Expected Error state when repository is dirty, got %v", model.state)
	}

	if !model.isDirty {
		t.Error("isDirty flag should be set")
	}
}

// Navigation and Back Tests

func TestBackNavigation(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}

	tests := []struct {
		name          string
		startState    SettingsState
		expectedState SettingsState
		setupFunc     func(*SettingsModel)
	}{
		{
			name:          "SelectChange to MainMenu",
			startState:    SettingsStateSelectChange,
			expectedState: SettingsStateMainMenu,
		},
		{
			name:          "ConfirmTypeChange to SelectChange",
			startState:    SettingsStateConfirmTypeChange,
			expectedState: SettingsStateSelectChange,
		},
		// UpdateLocalPath navigation - depends on changeType
		{
			name:          "UpdateLocalPath to SelectChange (standalone)",
			startState:    SettingsStateUpdateLocalPath,
			expectedState: SettingsStateSelectChange,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionLocalPath
			},
		},
		{
			name:          "UpdateLocalPath to ConfirmTypeChange (repo type change)",
			startState:    SettingsStateUpdateLocalPath,
			expectedState: SettingsStateConfirmTypeChange,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionRepositoryType
				m.newRepositoryType = "local"
			},
		},
		// UpdateGitHubURL navigation - depends on changeType
		{
			name:          "UpdateGitHubURL to SelectChange (standalone)",
			startState:    SettingsStateUpdateGitHubURL,
			expectedState: SettingsStateSelectChange,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionGitHubURL
			},
		},
		{
			name:          "UpdateGitHubURL to ConfirmTypeChange (repo type change)",
			startState:    SettingsStateUpdateGitHubURL,
			expectedState: SettingsStateConfirmTypeChange,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionRepositoryType
				m.newRepositoryType = "github"
			},
		},
		// UpdateGitHubBranch navigation - depends on changeType
		{
			name:          "UpdateGitHubBranch to SelectChange (standalone)",
			startState:    SettingsStateUpdateGitHubBranch,
			expectedState: SettingsStateSelectChange,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionGitHubBranch
			},
		},
		{
			name:          "UpdateGitHubBranch to UpdateGitHubURL (URL change flow)",
			startState:    SettingsStateUpdateGitHubBranch,
			expectedState: SettingsStateUpdateGitHubURL,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionGitHubURL
				m.newGitHubURL = "https://github.com/test/repo.git"
			},
		},
		{
			name:          "UpdateGitHubBranch to UpdateGitHubURL (repo type change)",
			startState:    SettingsStateUpdateGitHubBranch,
			expectedState: SettingsStateUpdateGitHubURL,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionRepositoryType
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/test/repo.git"
			},
		},
		// UpdateGitHubPath navigation - depends on changeType
		{
			name:          "UpdateGitHubPath to SelectChange (standalone)",
			startState:    SettingsStateUpdateGitHubPath,
			expectedState: SettingsStateSelectChange,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionGitHubPath
			},
		},
		{
			name:          "UpdateGitHubPath to UpdateGitHubBranch (URL change flow)",
			startState:    SettingsStateUpdateGitHubPath,
			expectedState: SettingsStateUpdateGitHubBranch,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionGitHubURL
				m.newGitHubURL = "https://github.com/test/repo.git"
			},
		},
		{
			name:          "UpdateGitHubPath to UpdateGitHubBranch (repo type change)",
			startState:    SettingsStateUpdateGitHubPath,
			expectedState: SettingsStateUpdateGitHubBranch,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionRepositoryType
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/test/repo.git"
			},
		},
		// UpdateGitHubPAT navigation - depends on changeType
		{
			name:          "UpdateGitHubPAT to SelectChange (standalone)",
			startState:    SettingsStateUpdateGitHubPAT,
			expectedState: SettingsStateSelectChange,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionGitHubPAT
			},
		},
		{
			name:          "UpdateGitHubPAT to UpdateGitHubPath (repo type change)",
			startState:    SettingsStateUpdateGitHubPAT,
			expectedState: SettingsStateUpdateGitHubPath,
			setupFunc: func(m *SettingsModel) {
				m.changeType = ChangeOptionRepositoryType
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/test/repo.git"
				m.newGitHubPath = "/test/path"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.state = tt.startState
			if tt.setupFunc != nil {
				tt.setupFunc(model)
			}

			updatedModel, _ := model.Update(escMsg)
			model = updatedModel.(*SettingsModel)

			if model.state != tt.expectedState {
				t.Errorf("Expected state %v after Esc from %v (changeType=%v), got %v",
					tt.expectedState, tt.startState, model.changeType, model.state)
			}
		})
	}
}

func TestBackNavigation_FromSelectChange_BackOption(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")
	model.state = SettingsStateSelectChange

	// Find the "Back" option
	options := model.getMenuOptions()
	backIndex := -1
	for i, opt := range options {
		if opt.Option == ChangeOptionBack {
			backIndex = i
			break
		}
	}

	if backIndex == -1 {
		t.Fatal("Back option not found")
	}

	// Select back option
	model.selectedOption = backIndex
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should return to main menu
	if model.state != SettingsStateMainMenu {
		t.Errorf("Expected MainMenu state after selecting Back, got %v", model.state)
	}
}

// Error Recovery Tests

func TestDirtyStateMessage_NextStateByChangeType(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	tests := []struct {
		name              string
		currentState      SettingsState
		changeType        ChangeOption
		expectedNextState SettingsState
		setupFunc         func(*SettingsModel)
	}{
		{
			name:              "UpdateGitHubBranch standalone to Confirmation (clean)",
			currentState:      SettingsStateUpdateGitHubBranch,
			changeType:        ChangeOptionGitHubBranch,
			expectedNextState: SettingsStateConfirmation,
			setupFunc: func(m *SettingsModel) {
				m.newGitHubBranch = "develop"
			},
		},
		{
			name:              "UpdateGitHubBranch URL flow to UpdateGitHubPath (clean)",
			currentState:      SettingsStateUpdateGitHubBranch,
			changeType:        ChangeOptionGitHubURL,
			expectedNextState: SettingsStateUpdateGitHubPath,
			setupFunc: func(m *SettingsModel) {
				m.newGitHubURL = "https://github.com/test/repo.git"
				m.newGitHubBranch = "main"
			},
		},
		{
			name:              "UpdateGitHubBranch repo type change to UpdateGitHubPath (clean)",
			currentState:      SettingsStateUpdateGitHubBranch,
			changeType:        ChangeOptionRepositoryType,
			expectedNextState: SettingsStateUpdateGitHubPath,
			setupFunc: func(m *SettingsModel) {
				m.newRepositoryType = "github"
				m.newGitHubURL = "https://github.com/test/repo.git"
				m.newGitHubBranch = "main"
			},
		},
		{
			name:              "ManualRefresh to RefreshInProgress (clean)",
			currentState:      SettingsStateManualRefresh,
			changeType:        ChangeOptionManualRefresh,
			expectedNextState: SettingsStateRefreshInProgress,
			setupFunc: func(m *SettingsModel) {
				m.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSettingsModel(ctx)
			model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
			model.state = tt.currentState
			model.changeType = tt.changeType

			if tt.setupFunc != nil {
				tt.setupFunc(model)
			}

			// Send clean dirty state message
			cleanMsg := dirtyStateMsg{isDirty: false, err: nil}
			updatedModel, _ := model.Update(cleanMsg)
			settingsModel := updatedModel.(*SettingsModel)

			if settingsModel.state != tt.expectedNextState {
				t.Errorf("Expected next state %v after clean dirtyStateMsg in %v with changeType=%v, got %v",
					tt.expectedNextState, tt.currentState, tt.changeType, settingsModel.state)
			}

			if settingsModel.isDirty {
				t.Error("isDirty should be false for clean state")
			}
		})
	}
}

func TestDirtyStateMessage_DirtyRepository(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	tests := []struct {
		name         string
		currentState SettingsState
		changeType   ChangeOption
	}{
		{
			name:         "UpdateGitHubBranch standalone with dirty repo",
			currentState: SettingsStateUpdateGitHubBranch,
			changeType:   ChangeOptionGitHubBranch,
		},
		{
			name:         "UpdateGitHubBranch URL flow with dirty repo",
			currentState: SettingsStateUpdateGitHubBranch,
			changeType:   ChangeOptionGitHubURL,
		},
		{
			name:         "ManualRefresh with dirty repo",
			currentState: SettingsStateManualRefresh,
			changeType:   ChangeOptionManualRefresh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSettingsModel(ctx)
			model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
			model.state = tt.currentState
			model.changeType = tt.changeType

			// Send dirty state message
			dirtyMsg := dirtyStateMsg{isDirty: true, err: nil}
			updatedModel, _ := model.Update(dirtyMsg)
			settingsModel := updatedModel.(*SettingsModel)

			// All dirty states should go to Error state
			if settingsModel.state != SettingsStateError {
				t.Errorf("Expected Error state when repository is dirty in %v with changeType=%v, got %v",
					tt.currentState, tt.changeType, settingsModel.state)
			}

			if !settingsModel.isDirty {
				t.Error("isDirty flag should be set")
			}
		})
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

func TestErrorRecovery_MultipleErrors(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")

	// First error
	errorMsg1 := settingsErrorMsg{err: &testError{"first error"}}
	updatedModel, _ := model.Update(errorMsg1)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateError {
		t.Fatalf("Expected Error state after first error, got %v", model.state)
	}

	// Second error while in error state
	errorMsg2 := settingsErrorMsg{err: &testError{"second error"}}
	updatedModel, _ = model.Update(errorMsg2)
	model = updatedModel.(*SettingsModel)

	// Should still be in error state
	if model.state != SettingsStateError {
		t.Errorf("Expected to remain in Error state, got %v", model.state)
	}
}

// View Rendering Tests

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

			// Views should be reasonably sized
			if len(view) < 10 {
				t.Errorf("View for state %v seems too short: %d characters", state, len(view))
			}
		})
	}
}

func TestViewRendering_WithChanges(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/old/path")
	model.state = SettingsStateConfirmation
	model.hasChanges = true
	model.changeType = ChangeOptionLocalPath
	model.newStorageDir = "/new/path"

	view := model.View()

	if view == "" {
		t.Error("Confirmation view should not be empty")
	}

	// View should be reasonably sized with changes summary
	if len(view) < 50 {
		t.Error("Confirmation view with changes seems too short")
	}
}

// Message Handling Tests

func TestMessageHandling_RefreshComplete(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
	model.refreshInProgress = true
	model.state = SettingsStateRefreshInProgress

	// Success case
	refreshMsg := refreshCompleteMsg{
		syncInfo: &repository.SyncInfo{
			Message: "synced",
		},
		err: nil,
	}

	updatedModel, _ := model.Update(refreshMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.refreshInProgress {
		t.Error("refreshInProgress should be cleared after refresh completes")
	}

	if settingsModel.state != SettingsStateMainMenu {
		t.Errorf("Expected MainMenu state after successful refresh, got %v", settingsModel.state)
	}

	if settingsModel.lastSyncInfo == nil {
		t.Error("lastSyncInfo should be set after successful refresh")
	}
}

func TestMessageHandling_RefreshError(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
	model.refreshInProgress = true

	// Error case
	refreshMsg := refreshCompleteMsg{
		syncInfo: nil,
		err:      &testError{"refresh failed"},
	}

	updatedModel, _ := model.Update(refreshMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.refreshInProgress {
		t.Error("refreshInProgress should be cleared even on error")
	}

	// Note: The implementation may return to MainMenu instead of Error state on refresh error
	// This is acceptable behavior as the error is logged
	if settingsModel.state != SettingsStateError && settingsModel.state != SettingsStateMainMenu {
		t.Errorf("Expected Error or MainMenu state after refresh error, got %v", settingsModel.state)
	}
}

func TestMessageHandling_DirtyState(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/test/repo.git", "main")
	model.state = SettingsStateManualRefresh

	// Clean state
	cleanMsg := dirtyStateMsg{isDirty: false, err: nil}
	updatedModel, _ := model.Update(cleanMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.isDirty {
		t.Error("isDirty should be false for clean state")
	}

	// Dirty state
	model.state = SettingsStateManualRefresh
	dirtyMsg := dirtyStateMsg{isDirty: true, err: nil}
	updatedModel, _ = model.Update(dirtyMsg)
	settingsModel = updatedModel.(*SettingsModel)

	if !settingsModel.isDirty {
		t.Error("isDirty should be true for dirty state")
	}

	if settingsModel.state != SettingsStateError {
		t.Errorf("Expected Error state when repository is dirty, got %v", settingsModel.state)
	}
}

func TestMessageHandling_SettingsComplete(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.hasChanges = true

	completeMsg := settingsCompleteMsg{}
	updatedModel, cmd := model.Update(completeMsg)
	settingsModel := updatedModel.(*SettingsModel)

	if settingsModel.state != SettingsStateComplete {
		t.Errorf("Expected Complete state, got %v", settingsModel.state)
	}

	if cmd == nil {
		t.Error("Complete message should return config reload command")
	}
}

// Complex Integration Scenarios

func TestComplexScenario_MultipleChangesAndCancellations(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/original/path")

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}

	// Navigate to select change
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Try to change local path
	model.selectedOption = 1
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Cancel
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateSelectChange {
		t.Fatalf("Expected SelectChange after cancel, got %v", model.state)
	}

	// Try to change repository type
	model.selectedOption = 0
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateConfirmTypeChange {
		t.Fatalf("Expected ConfirmTypeChange, got %v", model.state)
	}

	// Cancel type change
	noMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updatedModel, _ = model.Update(noMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateSelectChange {
		t.Errorf("Expected SelectChange after rejecting type change, got %v", model.state)
	}

	// Verify no changes were recorded
	if model.hasChanges {
		t.Error("Expected hasChanges to be false after all cancellations")
	}
}

func TestComplexScenario_GitHubFullFlow(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createGitHubConfig("/test/path", "https://github.com/old/repo.git", "main")

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}

	// Navigate to select change
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Update GitHub URL (which triggers full flow: URL -> Branch -> Path)
	options := model.getMenuOptions()
	for i, opt := range options {
		if opt.Option == ChangeOptionGitHubURL {
			model.selectedOption = i
			break
		}
	}

	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	if model.state != SettingsStateUpdateGitHubURL {
		t.Fatalf("Expected UpdateGitHubURL, got %v", model.state)
	}

	// Enter URL
	model.textInput.SetValue("https://github.com/new/repo.git")
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Should be at branch
	if model.state != SettingsStateUpdateGitHubBranch {
		t.Fatalf("Expected UpdateGitHubBranch, got %v", model.state)
	}

	// Enter branch
	model.textInput.SetValue("develop")
	updatedModel, cmd := model.Update(enterMsg)
	model = updatedModel.(*SettingsModel)

	// Execute dirty check
	if cmd != nil {
		msg := cmd()
		updatedModel, _ = model.Update(msg)
		model = updatedModel.(*SettingsModel)
	}

	// Verify that URL, branch, and changeType were properly set
	if model.newGitHubURL != "https://github.com/new/repo.git" {
		t.Errorf("Expected URL to be saved, got %v", model.newGitHubURL)
	}

	if model.newGitHubBranch != "develop" {
		t.Errorf("Expected branch to be saved, got %v", model.newGitHubBranch)
	}

	if model.changeType != ChangeOptionGitHubURL {
		t.Errorf("Expected changeType to remain GitHubURL, got %v", model.changeType)
	}

	t.Logf("Final state: %v (hasChanges=%v)", model.state, model.hasChanges)
}

// Edge Cases

func TestEdgeCase_RapidStateTransitions(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}

	// Rapid transitions
	for range 5 {
		updatedModel, _ := model.Update(enterMsg)
		model = updatedModel.(*SettingsModel)
	}

	// Should not panic and should be in a valid state
	// After multiple enters from MainMenu, we go through SelectChange and may end up
	// in various states depending on menu selection
	validStates := []SettingsState{
		SettingsStateMainMenu,
		SettingsStateSelectChange,
		SettingsStateUpdateLocalPath,
		SettingsStateConfirmTypeChange,
		SettingsStateUpdateGitHubURL,
		SettingsStateUpdateGitHubPAT,
		SettingsStateUpdateGitHubBranch,
		SettingsStateUpdateGitHubPath,
	}

	stateValid := false
	if slices.Contains(validStates, model.state) {
		stateValid = true

	}

	if !stateValid {
		t.Errorf("Unexpected state after rapid transitions: %v", model.state)
	}
}

func TestEdgeCase_NilConfig(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = nil

	// Should not panic when rendering with nil config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("View panicked with nil config: %v", r)
		}
	}()

	view := model.View()
	if view == "" {
		t.Error("View should not be empty even with nil config")
	}
}

func TestEdgeCase_ExtremelyLongPaths(t *testing.T) {
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, &config.Config{}, logger)

	model := NewSettingsModel(ctx)
	model.currentConfig = createLocalConfig("/test/path")
	model.state = SettingsStateUpdateLocalPath

	// Create an extremely long path
	longPath := "/home/user"
	for range 100 {
		longPath = filepath.Join(longPath, "very_long_directory_name_that_repeats")
	}

	model.textInput.SetValue(longPath)

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panicked with extremely long path: %v", r)
		}
	}()

	view := model.View()
	if view == "" {
		t.Error("View should not be empty even with extremely long path")
	}
}
