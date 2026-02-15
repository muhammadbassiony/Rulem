package saverulesmodel

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"rulem/internal/config"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/repository"
	"rulem/internal/tui/components/filepicker"
	"rulem/internal/tui/helpers"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func createTestConfigWithPath(path string) *config.Config {
	return &config.Config{
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "test-repo-123456",
				Name:      "Test Repository",
				Type:      repository.RepositoryTypeLocal,
				CreatedAt: 1234567890,
				Path:      path,
			},
		},
	}
}

// Test utilities

func createTestLogger() *logging.AppLogger {
	logger, _ := logging.NewTestLogger()
	return logger
}

func createTestStorageDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "rulem-save-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

func createTestWorkingDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "rulem-work-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp work dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

func createTestUIContext(t *testing.T) helpers.UIContext {
	storageDir := createTestStorageDir(t)

	// Create and change to a temporary working directory to isolate test files
	workDir := createTestWorkingDir(t)

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to work directory
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Failed to change to work directory: %v", err)
	}

	// Set up cleanup to restore original directory
	t.Cleanup(func() {
		os.Chdir(originalWd)
	})

	cfg := createTestConfigWithPath(storageDir)

	logger := createTestLogger()

	return helpers.NewUIContext(80, 24, cfg, logger)
}

func createTestFiles(t *testing.T, workDir string) map[string]string {
	// Create test files in working directory
	testFiles := map[string]string{
		"rule1.md":           "# Rule 1\nThis is the first rule.",
		"rule2.md":           "# Rule 2\nThis is the second rule.",
		"docs/nested.md":     "# Nested Rule\nThis is a nested rule file.",
		"config/settings.md": "# Configuration\nSettings and configuration rules.",
		"README.md":          "# Project README\nOverview of the project.",
		"CHANGELOG.md":       "# Changelog\nProject version history.",
		"scripts/deploy.md":  "# Deployment\nDeployment scripts and procedures.",
		"tests/README.md":    "# Test Documentation\nHow to run tests.",
		".hidden/secret.md":  "# Hidden Rules\nSecret configuration rules.",
	}

	for relPath, content := range testFiles {
		fullPath := filepath.Join(workDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", relPath, err)
		}
	}

	return testFiles
}

func createTestModel(t *testing.T) SaveRulesModel {
	ctx := createTestUIContext(t)
	return NewSaveRulesModel(ctx)
}

func createTestModelWithFiles(t *testing.T) (SaveRulesModel, []filemanager.FileItem, map[string]string) {
	// Create working directory with test files
	workDir := createTestWorkingDir(t)
	testFiles := createTestFiles(t, workDir)

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to work directory
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Failed to change to work directory: %v", err)
	}

	// Set up cleanup to restore original directory
	t.Cleanup(func() {
		os.Chdir(originalWd)
	})

	// Create model - storage dir is separate from work dir for SaveRulesModel
	storageDir := createTestStorageDir(t)
	cfg := createTestConfigWithPath(storageDir)
	logger := createTestLogger()
	ctx := helpers.NewUIContext(80, 24, cfg, logger)
	model := NewSaveRulesModel(ctx)

	// Get the files from scan
	files, err := model.fileManager.ScanCurrDirectory()
	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	return model, files, testFiles
}

// Integration Tests

func TestCompleteSaveWorkflow(t *testing.T) {
	model, files, testFiles := createTestModelWithFiles(t)

	// 1. Start with loading state and receive scan complete
	if model.state != StateLoading {
		t.Errorf("Expected initial state %v, got %v", StateLoading, model.state)
	}

	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 2. Should be in file selection state
	if model.state != StateFileSelection {
		t.Errorf("Expected state %v, got %v", StateFileSelection, model.state)
	}

	// 3. Select a file
	fileSelectedMsg := filepicker.FileSelectedMsg{File: model.markdownFiles[0]}
	updatedModel, _ = model.Update(fileSelectedMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 4. Should be in filename input state
	if model.state != StateFileNameInput {
		t.Errorf("Expected state %v, got %v", StateFileNameInput, model.state)
	}

	// 5. Enter custom filename and submit
	model.nameInput.SetValue("custom-rule.md")
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(keyMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 6. Should transition to saving state
	if model.state != StateSaving {
		t.Errorf("Expected state %v, got %v", StateSaving, model.state)
	}

	// 7. Simulate successful save
	destPath := filepath.Join(model.fileManager.GetStorageDir(), "custom-rule.md")
	successMsg := SaveFileCompleteMsg{DestPath: destPath}
	updatedModel, _ = model.Update(successMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 8. Should be in success state
	if model.state != StateSuccess {
		t.Errorf("Expected state %v, got %v", StateSuccess, model.state)
	}

	if model.destinationPath != destPath {
		t.Errorf("Expected destination path %s, got %s", destPath, model.destinationPath)
	}

	// Verify workflow completed with correct state transitions
	if len(testFiles) == 0 {
		t.Error("Should have test files")
	}
}

func TestSaveWorkflowWithOverwrite(t *testing.T) {
	model, files, _ := createTestModelWithFiles(t)

	// Get storage directory and create a conflicting file
	storageDir := model.fileManager.GetStorageDir()
	conflictFile := filepath.Join(storageDir, "conflict.md")
	if err := os.WriteFile(conflictFile, []byte("existing content"), 0644); err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}

	// 1. Start with loading state and receive scan complete
	if model.state != StateLoading {
		t.Errorf("Expected initial state %v, got %v", StateLoading, model.state)
	}

	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 2. Select a file
	if model.state != StateFileSelection {
		t.Errorf("Expected state %v, got %v", StateFileSelection, model.state)
	}

	fileSelectedMsg := filepicker.FileSelectedMsg{File: model.markdownFiles[0]}
	updatedModel, _ = model.Update(fileSelectedMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 3. Should be in filename input state
	if model.state != StateFileNameInput {
		t.Errorf("Expected state %v, got %v", StateFileNameInput, model.state)
	}

	// 4. Enter conflicting filename and submit
	model.nameInput.SetValue("conflict.md")
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(keyMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 5. Should transition to saving state
	if model.state != StateSaving {
		t.Errorf("Expected state %v, got %v", StateSaving, model.state)
	}

	// 6. Simulate save error due to conflict
	errorMsg := SaveFileErrorMsg{
		Err:              errors.New("destination file already exists: conflict.md (use overwrite=true to replace)"),
		IsOverwriteError: true,
	}
	updatedModel, _ = model.Update(errorMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 7. Should be in confirmation state
	if model.state != StateConfirmation {
		t.Errorf("Expected state %v, got %v", StateConfirmation, model.state)
	}

	if !model.isOverwriteError {
		t.Error("Should be marked as overwrite error")
	}

	// 8. Confirm overwrite
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, _ = model.Update(keyMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 9. Should transition back to saving state
	if model.state != StateSaving {
		t.Errorf("Expected state %v after confirmation, got %v", StateSaving, model.state)
	}

	// 10. Simulate successful save
	successMsg := SaveFileCompleteMsg{DestPath: conflictFile}
	updatedModel, _ = model.Update(successMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// 11. Should be in success state
	if model.state != StateSuccess {
		t.Errorf("Expected state %v, got %v", StateSuccess, model.state)
	}

	if model.destinationPath != conflictFile {
		t.Errorf("Expected destination path %s, got %s", conflictFile, model.destinationPath)
	}
}

func TestCancelWorkflow(t *testing.T) {
	model, files, _ := createTestModelWithFiles(t)

	// Process scan complete to get to file selection
	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Test escape from file selection
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd := model.Update(keyMsg)

	if cmd == nil {
		t.Error("Should return navigation command for escape")
	}

	// Select file and go to filename input
	fileSelectedMsg := filepicker.FileSelectedMsg{File: model.markdownFiles[0]}
	updatedModel, _ = model.Update(fileSelectedMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Test escape from filename input
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd = model.Update(keyMsg)

	if cmd == nil {
		t.Error("Should return navigation command for escape")
	}

	// Test escape from confirmation state
	model.state = StateConfirmation
	model.isOverwriteError = true
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd = model.Update(keyMsg)

	if cmd == nil {
		t.Error("Should return navigation command for escape")
	}

	// Test escape from error state
	model.state = StateError
	model.err = errors.New("test error")
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd = model.Update(keyMsg)

	if cmd == nil {
		t.Error("Should return navigation command for escape")
	}
}

func TestNavigationWorkflow(t *testing.T) {
	model, files, _ := createTestModelWithFiles(t)

	// Get to success state
	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	model.state = StateSuccess
	model.destinationPath = "/test/path/file.md"

	// Test main menu navigation
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")}
	updatedModel, cmd := model.Update(keyMsg)

	if cmd == nil {
		t.Error("Should return navigation command for main menu")
	}

	// Test save another file
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	updatedModel, _ = model.Update(keyMsg)
	result, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	if result.state != StateFileSelection {
		t.Errorf("Expected state %v after 'save another', got %v", StateFileSelection, result.state)
	}

	// Verify state is reset
	if result.selectedFile != (filemanager.FileItem{}) {
		t.Error("Selected file should be reset")
	}
	if result.newFileName != "" {
		t.Error("New filename should be reset")
	}
	if result.destinationPath != "" {
		t.Error("Destination path should be reset")
	}
}

// TestWindowResizeInErrorState verifies that window resize messages don't cause
// nil pointer dereference when the model is in error state (e.g., repository
// preparation failed). This is a regression test for a bug where
// repositoryList.SetSize() was called without checking if repos were prepared.
func TestWindowResizeInErrorState(t *testing.T) {
	// Create a model in error state with nil preparedRepos (simulating repo prep failure)
	model := SaveRulesModel{
		logger:        createTestLogger(),
		state:         StateError,
		err:           errors.New("repository validation failed"),
		preparedRepos: nil, // Critical: no prepared repos
	}

	// Send a window resize message - this should NOT panic
	resizeMsg := tea.WindowSizeMsg{Width: 100, Height: 50}

	// This should not panic
	var recovered interface{}
	func() {
		defer func() {
			recovered = recover()
		}()
		model.Update(resizeMsg)
	}()

	if recovered != nil {
		t.Errorf("Window resize in error state caused panic: %v", recovered)
	}
}

func TestErrorScenarios(t *testing.T) {
	model := createTestModel(t)

	// Test error state and retry
	model.state = StateError
	model.err = errors.New("test error")

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	updatedModel, cmd := model.Update(keyMsg)
	result, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	if result.state != StateLoading {
		t.Errorf("Expected state %v after retry, got %v", StateLoading, result.state)
	}

	if result.err != nil {
		t.Error("Error should be cleared after retry")
	}

	if cmd == nil {
		t.Error("Should return scan command for retry")
	}

	// Test overwrite error handling
	model.state = StateConfirmation
	model.isOverwriteError = true
	model.err = errors.New("file exists")

	// Test "no" response
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updatedModel, _ = model.Update(keyMsg)
	result, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	if result.state != StateFileNameInput {
		t.Errorf("Expected state %v after 'no', got %v", StateFileNameInput, result.state)
	}

	if result.err != nil {
		t.Error("Error should be cleared after 'no'")
	}

	if result.isOverwriteError {
		t.Error("Overwrite error flag should be cleared")
	}
}

// Unit Tests for Model Creation and Initialization

func TestNewSaveRulesModel(t *testing.T) {
	model := createTestModel(t)

	if model.state != StateLoading {
		t.Errorf("Expected initial state %v, got %v", StateLoading, model.state)
	}

	if model.fileManager == nil {
		t.Error("FileManager should be initialized")
	}

	if len(model.markdownFiles) != 0 {
		t.Error("Should start with no markdown files")
	}

	if model.filePicker != nil {
		t.Error("FilePicker should not be initialized at start")
	}

	if model.newFileName != "" {
		t.Error("New filename should be empty at start")
	}

	if model.destinationPath != "" {
		t.Error("Destination path should be empty at start")
	}

	if model.err != nil {
		t.Error("Error should be nil at start")
	}

	if model.isOverwriteError {
		t.Error("Overwrite error flag should be false at start")
	}

	// Verify components are properly initialized
	if model.layout.ContentWidth() == 0 {
		t.Error("Layout should be configured")
	}

	if model.nameInput.CharLimit == 0 {
		t.Error("Name input should be configured")
	}
}

func TestSaveRulesModel_Init(t *testing.T) {
	model := createTestModel(t)

	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}

	// Test with nil FileManager
	ctx := createTestUIContext(t)
	model = NewSaveRulesModel(ctx)
	model.fileManager = nil

	cmd = model.Init()
	if cmd == nil {
		t.Error("Init should return a command even with nil FileManager")
	}

	// Execute the command to get the error message
	msg := cmd()
	errorMsg, ok := msg.(SaveFileErrorMsg)
	if !ok {
		t.Errorf("Expected SaveFileErrorMsg, got %T", msg)
	}

	if errorMsg.Err == nil {
		t.Error("Expected error for nil FileManager")
	}

	if !strings.Contains(errorMsg.Err.Error(), "FileManager not initialized") {
		t.Errorf("Expected 'FileManager not initialized' error, got: %v", errorMsg.Err)
	}
}

// Message Handling Tests

func TestSaveRulesModel_FileScanCompleteMsg(t *testing.T) {
	model := createTestModel(t)

	files := []filemanager.FileItem{
		{Name: "test1.md", Path: "test1.md"},
		{Name: "test2.md", Path: "test2.md"},
	}

	msg := FileScanCompleteMsg{Files: files}
	updatedModel, cmd := model.Update(msg)

	result, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if result.state != StateFileSelection {
		t.Errorf("Expected state %v, got %v", StateFileSelection, result.state)
	}

	if len(result.markdownFiles) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(result.markdownFiles))
	}

	if result.filePicker == nil {
		t.Error("FilePicker should be created")
	}

	if result.err != nil {
		t.Error("Error should be cleared")
	}

	// FilePicker.Init() may return a command for initial preview
	_ = cmd
}

func TestSaveRulesModel_FileScanErrorMsg(t *testing.T) {
	model := createTestModel(t)

	testError := errors.New("scan failed")
	msg := FileScanErrorMsg{Err: testError}
	updatedModel, cmd := model.Update(msg)

	result, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if result.state != StateError {
		t.Errorf("Expected state %v, got %v", StateError, result.state)
	}

	if result.err != testError {
		t.Errorf("Expected error %v, got %v", testError, result.err)
	}

	if result.isOverwriteError {
		t.Error("Should not be marked as overwrite error")
	}

	if cmd != nil {
		t.Error("Should not return a command for scan error")
	}
}

func TestSaveRulesModel_FileSelectedMsg(t *testing.T) {
	model, files, _ := createTestModelWithFiles(t)

	// Process scan complete first
	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Test file selection
	selectedFile := model.markdownFiles[0]
	fileSelectedMsg := filepicker.FileSelectedMsg{File: selectedFile}
	updatedModel, cmd := model.Update(fileSelectedMsg)

	result, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if result.state != StateFileNameInput {
		t.Errorf("Expected state %v, got %v", StateFileNameInput, result.state)
	}

	if result.selectedFile != selectedFile {
		t.Errorf("Expected selected file %v, got %v", selectedFile, result.selectedFile)
	}

	if result.newFileName != selectedFile.Name {
		t.Errorf("Expected new filename %s, got %s", selectedFile.Name, result.newFileName)
	}

	if !result.nameInput.Focused() {
		t.Error("Name input should be focused")
	}

	if cmd == nil {
		t.Error("Should return blink command for text input")
	}
}

func TestSaveRulesModel_SaveFileCompleteMsg(t *testing.T) {
	model := createTestModel(t)

	testPath := "/test/path/file.md"
	msg := SaveFileCompleteMsg{DestPath: testPath}
	updatedModel, cmd := model.Update(msg)

	result, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if result.state != StateSuccess {
		t.Errorf("Expected state %v, got %v", StateSuccess, result.state)
	}

	if result.destinationPath != testPath {
		t.Errorf("Expected destination path %s, got %s", testPath, result.destinationPath)
	}

	if result.err != nil {
		t.Errorf("Error should be cleared, got %v", result.err)
	}

	if cmd != nil {
		t.Error("Should not return a command for save complete")
	}
}

func TestSaveRulesModel_SaveFileErrorMsg(t *testing.T) {
	model := createTestModel(t)

	// Test overwrite error
	testError := errors.New("file already exists")
	msg := SaveFileErrorMsg{Err: testError, IsOverwriteError: true}
	updatedModel, cmd := model.Update(msg)

	result, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if result.state != StateConfirmation {
		t.Errorf("Expected state %v for overwrite error, got %v", StateConfirmation, result.state)
	}

	if result.err != testError {
		t.Errorf("Expected error %v, got %v", testError, result.err)
	}

	if !result.isOverwriteError {
		t.Error("Should be marked as overwrite error")
	}

	if cmd != nil {
		t.Error("Should not return a command for save error")
	}

	// Test non-overwrite error
	model = createTestModel(t)
	msg = SaveFileErrorMsg{Err: testError, IsOverwriteError: false}
	updatedModel, _ = model.Update(msg)

	result, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if result.state != StateError {
		t.Errorf("Expected state %v for non-overwrite error, got %v", StateError, result.state)
	}

	if result.isOverwriteError {
		t.Error("Should not be marked as overwrite error")
	}
}

func TestSaveRulesModel_WindowSizeMsg(t *testing.T) {
	model := createTestModel(t)

	windowMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, cmd := model.Update(windowMsg)

	result, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	// Window size should be propagated to layout
	if result.layout.ContentWidth() <= 0 {
		t.Error("Layout should have updated content width")
	}

	if cmd != nil {
		t.Error("Should not return a command for window size")
	}

	// Test with FilePicker present
	model, files, _ := createTestModelWithFiles(t)
	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ = model.Update(scanMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	windowMsg = tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, cmd = model.Update(windowMsg)

	result, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	// Should propagate to FilePicker
	if result.filePicker == nil {
		t.Error("FilePicker should still exist")
	}

	if cmd != nil {
		t.Error("Should not return a command when propagating to FilePicker")
	}
}

func TestSaveRulesModel_SpinnerTickMsg(t *testing.T) {
	model := createTestModel(t)

	// Test in loading state
	model.state = StateLoading
	tickMsg := model.spinner.Tick()
	updatedModel, cmd := model.Update(tickMsg)

	_, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if cmd == nil {
		t.Error("Should return spinner command in loading state")
	}

	// Test in saving state
	model.state = StateSaving
	updatedModel, cmd = model.Update(tickMsg)

	_, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if cmd == nil {
		t.Error("Should return spinner command in saving state")
	}

	// Test in other states (should not handle)
	model.state = StateFileSelection
	updatedModel, cmd = model.Update(tickMsg)

	if cmd != nil {
		t.Error("Should not return command for spinner tick in non-loading/saving states")
	}
}

// Key Handling Tests

func TestSaveRulesModel_KeyHandling_FileSelection(t *testing.T) {
	model, files, _ := createTestModelWithFiles(t)

	// Get to file selection state
	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	tests := []struct {
		name       string
		key        string
		expectsCmd bool
	}{
		{"quit", "q", true},
		{"escape", "esc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			updatedModel, cmd := model.Update(keyMsg)

			if tt.expectsCmd && cmd == nil {
				t.Errorf("Expected command for key %s", tt.key)
			} else if !tt.expectsCmd && cmd != nil {
				t.Errorf("Did not expect command for key %s", tt.key)
			}

			if updatedModel == nil {
				t.Error("Updated model should not be nil")
			}
		})
	}
}

func TestSaveRulesModel_KeyHandling_FileNameInput(t *testing.T) {
	model, files, _ := createTestModelWithFiles(t)

	// Get to filename input state
	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	fileSelectedMsg := filepicker.FileSelectedMsg{File: model.markdownFiles[0]}
	updatedModel, _ = model.Update(fileSelectedMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Test Enter key (submit)
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(keyMsg)
	result, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if result.state != StateSaving {
		t.Errorf("Expected state %v after Enter, got %v", StateSaving, result.state)
	}

	if result.nameInput.Focused() {
		t.Error("Name input should be blurred after submit")
	}

	if cmd == nil {
		t.Error("Should return save command after Enter")
	}

	// Test Escape key
	model.state = StateFileNameInput
	model.nameInput.Focus()
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd = model.Update(keyMsg)

	if cmd == nil {
		t.Error("Should return navigation command for Escape")
	}

	// Test regular text input - need to set up fresh model for this test
	model = createTestModel(t)
	model.state = StateFileNameInput
	model.nameInput.Focus()
	model.nameInput.SetValue("") // Clear any existing value
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")}
	updatedModel, cmd = model.Update(keyMsg)
	result, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Error("Update should return SaveRulesModel")
	}

	if result.newFileName != "test" {
		t.Errorf("Expected new filename 'test', got %s", result.newFileName)
	}
}

func TestSaveRulesModel_KeyHandling_Confirmation(t *testing.T) {
	model := createTestModel(t)
	model.state = StateConfirmation
	model.isOverwriteError = true
	model.selectedFile = filemanager.FileItem{Name: "test.md", Path: "/test/test.md"}

	tests := []struct {
		name          string
		key           string
		expectedState SaveFileModelState
		expectsCmd    bool
	}{
		{"yes", "y", StateSaving, true},
		{"no", "n", StateFileNameInput, true},
		{"escape", "esc", StateLoading, true}, // Should navigate to main menu
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := model
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			updatedModel, cmd := testModel.Update(keyMsg)
			result, ok := updatedModel.(SaveRulesModel)
			if !ok {
				t.Error("Update should return SaveRulesModel")
			}

			if tt.key != "esc" && result.state != tt.expectedState {
				t.Errorf("Expected state %v after %s, got %v", tt.expectedState, tt.key, result.state)
			}

			if tt.expectsCmd && cmd == nil {
				t.Errorf("Expected command for key %s", tt.key)
			} else if !tt.expectsCmd && cmd != nil {
				t.Errorf("Did not expect command for key %s", tt.key)
			}

			// Verify state changes for specific keys
			if tt.key == "n" {
				if result.err != nil {
					t.Error("Error should be cleared after 'no'")
				}
				if result.isOverwriteError {
					t.Error("Overwrite error flag should be cleared after 'no'")
				}
				if !result.nameInput.Focused() {
					t.Error("Name input should be focused after 'no'")
				}
			}
		})
	}
}

func TestSaveRulesModel_KeyHandling_Error(t *testing.T) {
	model := createTestModel(t)

	tests := []struct {
		name          string
		key           string
		expectedState SaveFileModelState
		setup         func(*SaveRulesModel)
	}{
		{
			name:          "retry from general error",
			key:           "r",
			expectedState: StateLoading,
			setup: func(m *SaveRulesModel) {
				m.state = StateError
				m.err = errors.New("general error")
				m.isOverwriteError = false
			},
		},
		{
			name:          "retry from overwrite error",
			key:           "r",
			expectedState: StateFileNameInput,
			setup: func(m *SaveRulesModel) {
				m.state = StateError
				m.err = errors.New("overwrite error")
				m.isOverwriteError = true
			},
		},
		{
			name:          "escape from error",
			key:           "esc",
			expectedState: StateError, // State doesn't change, but should navigate
			setup: func(m *SaveRulesModel) {
				m.state = StateError
				m.err = errors.New("test error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := model
			tt.setup(&testModel)

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			updatedModel, cmd := testModel.Update(keyMsg)
			result, ok := updatedModel.(SaveRulesModel)
			if !ok {
				t.Error("Update should return SaveRulesModel")
			}

			if tt.key != "esc" && result.state != tt.expectedState {
				t.Errorf("Expected state %v after %s, got %v", tt.expectedState, tt.key, result.state)
			}

			if cmd == nil {
				t.Errorf("Expected command for key %s", tt.key)
			}

			if tt.key == "r" && result.err != nil {
				t.Error("Error should be cleared after retry")
			}
		})
	}
}

func TestSaveRulesModel_KeyHandling_Success(t *testing.T) {
	model := createTestModel(t)
	model.state = StateSuccess
	model.destinationPath = "/test/path/file.md"

	tests := []struct {
		name          string
		key           string
		expectedState SaveFileModelState
		expectsCmd    bool
	}{
		{"main menu", "m", StateSuccess, true}, // State doesn't change, but should navigate
		{"save another", "a", StateFileSelection, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := model
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}

			updatedModel, cmd := testModel.Update(keyMsg)
			result, ok := updatedModel.(SaveRulesModel)
			if !ok {
				t.Error("Update should return SaveRulesModel")
			}

			if tt.key != "m" && result.state != tt.expectedState {
				t.Errorf("Expected state %v after %s, got %v", tt.expectedState, tt.key, result.state)
			}

			if tt.expectsCmd && cmd == nil {
				t.Errorf("Expected command for key %s", tt.key)
			} else if !tt.expectsCmd && cmd != nil {
				t.Errorf("Did not expect command for key %s", tt.key)
			}

			// Verify state reset for "save another"
			if tt.key == "a" {
				if result.selectedFile != (filemanager.FileItem{}) {
					t.Error("Selected file should be reset")
				}
				if result.newFileName != "" {
					t.Error("New filename should be reset")
				}
				if result.destinationPath != "" {
					t.Error("Destination path should be reset")
				}
			}
		})
	}
}

// Helper Function Tests

func TestSaveRulesModel_CommitOrDefaultFilename(t *testing.T) {
	model := createTestModel(t)
	model.selectedFile = filemanager.FileItem{Name: "original.md"}

	// Test with user input
	model.nameInput.SetValue("  custom.md  ")
	model.commitOrDefaultFilename()
	if model.newFileName != "custom.md" {
		t.Errorf("Expected trimmed filename 'custom.md', got %s", model.newFileName)
	}

	// Test with empty input (should default)
	model.nameInput.SetValue("   ")
	model.commitOrDefaultFilename()
	if model.newFileName != "original.md" {
		t.Errorf("Expected default filename 'original.md', got %s", model.newFileName)
	}

	// Test with completely empty input
	model.nameInput.SetValue("")
	model.commitOrDefaultFilename()
	if model.newFileName != "original.md" {
		t.Errorf("Expected default filename 'original.md', got %s", model.newFileName)
	}
}

func TestSaveRulesModel_OptionalNewNamePtr(t *testing.T) {
	model := createTestModel(t)
	model.selectedFile = filemanager.FileItem{Name: "original.md"}

	// Test when filename is unchanged
	model.newFileName = "original.md"
	ptr := model.optionalNewNamePtr()
	if ptr != nil {
		t.Error("Should return nil when filename is unchanged")
	}

	// Test when filename is changed
	model.newFileName = "custom.md"
	ptr = model.optionalNewNamePtr()
	if ptr == nil {
		t.Error("Should return pointer when filename is changed")
	}
	if *ptr != "custom.md" {
		t.Errorf("Expected pointer to 'custom.md', got %s", *ptr)
	}

	// Test when filename is empty (should return nil)
	model.newFileName = ""
	ptr = model.optionalNewNamePtr()
	if ptr != nil {
		t.Error("Should return nil when filename is empty")
	}
}

// Command Tests

func TestSaveRulesModel_ScanForFilesCmd(t *testing.T) {
	model, _, _ := createTestModelWithFiles(t)

	cmd := model.scanForFilesCmd()
	if cmd == nil {
		t.Error("Command should not be nil")
	}

	// Execute the command
	msg := cmd()
	if msg == nil {
		t.Error("Message should not be nil")
	}

	// Should return FileScanCompleteMsg for successful scan
	switch m := msg.(type) {
	case FileScanCompleteMsg:
		if len(m.Files) == 0 {
			t.Error("Should have found test files")
		}
	case FileScanErrorMsg:
		t.Errorf("Did not expect scan error: %v", m.Err)
	default:
		t.Errorf("Unexpected message type: %T", msg)
	}
}

func TestSaveRulesModel_SaveFileCmd(t *testing.T) {
	model, _, _ := createTestModelWithFiles(t)
	model.selectedFile = filemanager.FileItem{Name: "test.md", Path: "/tmp/test.md"}

	// Test with nil FileManager
	model.fileManager = nil
	cmd := model.saveFileCmd("test.md", nil, false)
	if cmd == nil {
		t.Error("Command should not be nil")
	}

	msg := cmd()
	errorMsg, ok := msg.(SaveFileErrorMsg)
	if !ok {
		t.Errorf("Expected SaveFileErrorMsg, got %T", msg)
	}
	if !strings.Contains(errorMsg.Err.Error(), "FileManager not initialized") {
		t.Errorf("Expected FileManager error, got: %v", errorMsg.Err)
	}
}

// FilePicker Absolute Path Regression Tests

func TestSaveRulesModel_FilePickerReceivesAbsolutePaths(t *testing.T) {
	model, files, _ := createTestModelWithFiles(t)

	if len(files) == 0 {
		t.Fatal("Should have found test files")
	}

	// Simulate the scan complete message - note that files from ScanCurrDirectory
	// are relative paths, but SaveRulesModel.Update should convert them to absolute
	scanCompleteMsg := FileScanCompleteMsg{Files: files}

	// Process the message
	updatedModel, _ := model.Update(scanCompleteMsg)
	model, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Verify state changed to file selection
	if model.state != StateFileSelection {
		t.Errorf("Expected state %v, got %v", StateFileSelection, model.state)
	}

	// Verify FilePicker was created
	if model.filePicker == nil {
		t.Fatal("FilePicker should be created after scan complete")
	}

	// Verify all files in markdownFiles have absolute paths
	if len(model.markdownFiles) == 0 {
		t.Fatal("Should have markdown files after scan")
	}

	for i, file := range model.markdownFiles {
		if !filepath.IsAbs(file.Path) {
			t.Errorf("File %d (%s) should have absolute path, got: %s", i, file.Name, file.Path)
		}

		// Verify the file actually exists and can be read
		content, err := os.ReadFile(file.Path)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", file.Path, err)
			continue
		}

		if len(content) == 0 {
			t.Errorf("File %s should not be empty", file.Path)
		}
	}
}

func TestSaveRulesModel_AbsolutePaths(t *testing.T) {
	// Create working directory with test file
	workDir := createTestWorkingDir(t)

	testContent := "# Test Rule\nThis is a test rule."
	testFile := filepath.Join(workDir, "test.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to work directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Failed to change to work directory: %v", err)
	}

	// Create storage dir separate from work dir
	storageDir := createTestStorageDir(t)
	logger := createTestLogger()

	// Create FileManager directly to test scanning
	fm, err := filemanager.NewFileManager(storageDir, logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	// Get files - should already have absolute paths
	files, err := fm.ScanCurrDirectory()
	if err != nil {
		t.Fatalf("ScanCurrDirectory failed: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("Should have found at least one file")
	}

	// Verify files have absolute paths
	for i, file := range files {
		if !filepath.IsAbs(file.Path) {
			t.Errorf("File %d should have absolute path, got: %s", i, file.Path)
		}

		// Verify file can be read directly using the path
		content, err := os.ReadFile(file.Path)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", file.Path, err)
		}

		if string(content) != testContent {
			t.Errorf("Content mismatch: expected %q, got %q", testContent, string(content))
		}
	}
}

// Broken Symlink Tests

func TestSaveRulesModel_BrokenSymlinkHandling(t *testing.T) {
	// Create working directory with test files
	workDir := createTestWorkingDir(t)
	_ = createTestFiles(t, workDir)

	// Change to work directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Failed to change to work directory: %v", err)
	}

	// Create storage directory and a conflicting broken symlink there
	storageDir := createTestStorageDir(t)
	brokenLinkPath := filepath.Join(storageDir, "broken-link.md")
	if err := os.Symlink("nonexistent-target.md", brokenLinkPath); err != nil {
		t.Fatalf("Failed to create broken symlink: %v", err)
	}

	// Create model
	cfg := createTestConfigWithPath(storageDir)
	logger := createTestLogger()
	ctx := helpers.NewUIContext(80, 24, cfg, logger)
	model := NewSaveRulesModel(ctx)

	// Get files and start workflow
	files, err := model.fileManager.ScanCurrDirectory()
	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	// Process scan complete
	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Select a file
	fileSelectedMsg := filepicker.FileSelectedMsg{File: model.markdownFiles[0]}
	updatedModel, _ = model.Update(fileSelectedMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Enter conflicting filename that matches broken symlink
	model.nameInput.SetValue("broken-link.md")
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(keyMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Should transition to saving state
	if model.state != StateSaving {
		t.Errorf("Expected state %v, got %v", StateSaving, model.state)
	}

	// Simulate save error due to broken symlink conflict
	errorMsg := SaveFileErrorMsg{
		Err:              errors.New("destination file already exists: broken-link.md (use overwrite=true to replace)"),
		IsOverwriteError: true,
	}
	updatedModel, _ = model.Update(errorMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Should be in confirmation state
	if model.state != StateConfirmation {
		t.Errorf("Expected state %v, got %v", StateConfirmation, model.state)
	}

	if !model.isOverwriteError {
		t.Error("Should be marked as overwrite error")
	}

	// Confirm overwrite
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	updatedModel, _ = model.Update(keyMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Should transition back to saving state
	if model.state != StateSaving {
		t.Errorf("Expected state %v after confirmation, got %v", StateSaving, model.state)
	}

	// Simulate successful save (overwriting broken symlink)
	successMsg := SaveFileCompleteMsg{DestPath: brokenLinkPath}
	updatedModel, _ = model.Update(successMsg)
	model, ok = updatedModel.(SaveRulesModel)
	if !ok {
		t.Fatal("Update should return SaveRulesModel")
	}

	// Should be in success state
	if model.state != StateSuccess {
		t.Errorf("Expected state %v, got %v", StateSuccess, model.state)
	}

	if model.destinationPath != brokenLinkPath {
		t.Errorf("Expected destination path %s, got %s", brokenLinkPath, model.destinationPath)
	}
}

// Benchmark Tests

func BenchmarkSaveRulesModel_Update(b *testing.B) {
	// Create model outside the benchmark loop
	workDir, err := os.MkdirTemp("", "bench-work-*")
	if err != nil {
		b.Fatalf("Failed to create work dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	storageDir, err := os.MkdirTemp("", "bench-storage-*")
	if err != nil {
		b.Fatalf("Failed to create storage dir: %v", err)
	}
	defer os.RemoveAll(storageDir)

	// Create test files
	for i := 0; i < 100; i++ {
		filename := filepath.Join(workDir, "file"+fmt.Sprintf("%d", i)+".md")
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			b.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(filename, []byte("# Test"), 0644); err != nil {
			b.Fatalf("Failed to create file: %v", err)
		}
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(workDir)

	cfg := createTestConfigWithPath(storageDir)
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, cfg, logger)
	model := NewSaveRulesModel(ctx)

	files, _ := model.fileManager.ScanCurrDirectory()
	scanMsg := FileScanCompleteMsg{Files: files}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(scanMsg)
	}
}

func BenchmarkSaveRulesModel_ScanForFiles(b *testing.B) {
	workDir, err := os.MkdirTemp("", "bench-scan-*")
	if err != nil {
		b.Fatalf("Failed to create work dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	// Create many test files
	for i := 0; i < 1000; i++ {
		filename := filepath.Join(workDir, "dir"+fmt.Sprintf("%d", i%10), "file"+fmt.Sprintf("%d", i)+".md")
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			b.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(filename, []byte("# Test"), 0644); err != nil {
			b.Fatalf("Failed to create file: %v", err)
		}
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(workDir)

	storageDir, _ := os.MkdirTemp("", "bench-storage-*")
	defer os.RemoveAll(storageDir)

	cfg := createTestConfigWithPath(storageDir)
	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(80, 24, cfg, logger)
	model := NewSaveRulesModel(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := model.scanForFilesCmd()
		cmd()
	}
}
