package importrulesmenu

import (
	"errors"
	"os"
	"path/filepath"
	"rulem/internal/config"
	"rulem/internal/editors"
	"rulem/internal/filemanager"
	"rulem/internal/logging"
	"rulem/internal/tui/components/filepicker"
	"rulem/internal/tui/helpers"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Test utilities and fixtures

func createTestLogger() *logging.AppLogger {
	return logging.NewAppLogger()
}

func createTestStorageDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "rulem-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

func createTestUIContext(t *testing.T) helpers.UIContext {
	storageDir := createTestStorageDir(t)

	// Create and change to a temporary working directory to isolate test files
	workDir, err := os.MkdirTemp("", "rulem-work-*")
	if err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to test working directory
	err = os.Chdir(workDir)
	if err != nil {
		t.Fatalf("Failed to change to work dir: %v", err)
	}

	// Ensure cleanup restores original directory and removes temp directory
	t.Cleanup(func() {
		os.Chdir(originalWd)
		os.RemoveAll(workDir)
	})

	return helpers.UIContext{
		Width:  80,
		Height: 24,
		Config: &config.Config{
			StorageDir: storageDir,
		},
		Logger: createTestLogger(),
	}
}

func createTestFiles(t *testing.T, storageDir string) []filemanager.FileItem {
	// Create test files in storage directory
	files := []struct {
		name    string
		content string
	}{
		{"eslint.md", "# ESLint Rules\nSome content"},
		{"prettier.md", "# Prettier Rules\nSome content"},
		{"typescript.md", "# TypeScript Rules\nSome content"},
	}

	var fileItems []filemanager.FileItem
	for _, file := range files {
		fullPath := filepath.Join(storageDir, file.name)
		err := os.WriteFile(fullPath, []byte(file.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		fileItems = append(fileItems, filemanager.FileItem{
			Name: file.name,
			Path: fullPath,
		})
	}

	return fileItems
}

func createTestModel(t *testing.T) *ImportRulesModel {
	ctx := createTestUIContext(t)
	return NewImportRulesModel(ctx)
}

func createTestModelWithFiles(t *testing.T) (*ImportRulesModel, []filemanager.FileItem) {
	ctx := createTestUIContext(t)
	model := NewImportRulesModel(ctx)
	files := createTestFiles(t, ctx.Config.StorageDir)
	return model, files
}

// Test NewImportRulesModel

func TestNewImportRulesModel(t *testing.T) {
	model := createTestModel(t)

	if model == nil {
		t.Error("Model should not be nil")
	}
	if model.state != StateLoading {
		t.Errorf("Expected state %v, got %v", StateLoading, model.state)
	}
	if model.logger == nil {
		t.Error("Logger should not be nil")
	}
	if model.fileManager == nil {
		t.Error("FileManager should not be nil")
	}
	if model.filePicker != nil {
		t.Error("FilePicker should be nil initially")
	}
	if model.isOverwriteError {
		t.Error("IsOverwriteError should be false initially")
	}
	if model.err != nil {
		t.Error("Error should be nil initially")
	}
	if len(model.ruleFiles) != 0 {
		t.Error("RuleFiles should be empty initially")
	}
	if model.selectedFile != (filemanager.FileItem{}) {
		t.Error("SelectedFile should be empty initially")
	}
	if model.selectedEditor != (editors.EditorRuleConfig{}) {
		t.Error("SelectedEditor should be empty initially")
	}
	if model.selectedImportMode != (CopyMode{}) {
		t.Error("SelectedImportMode should be empty initially")
	}
}

func TestNewImportRulesModel_WithInvalidDimensions(t *testing.T) {
	storageDir := createTestStorageDir(t)
	ctx := helpers.UIContext{
		Width:  0,
		Height: 0,
		Config: &config.Config{StorageDir: storageDir},
		Logger: createTestLogger(),
	}

	model := NewImportRulesModel(ctx)
	if model == nil {
		t.Error("Model should not be nil")
	}
	if model.state != StateLoading {
		t.Errorf("Expected state %v, got %v", StateLoading, model.state)
	}
}

func TestNewImportRulesModel_ImportModeListInitialization(t *testing.T) {
	model := createTestModel(t)

	items := model.importModeList.Items()
	if len(items) != 2 {
		t.Errorf("Expected 2 import mode items, got %d", len(items))
	}

	// Check copy mode
	copyMode, ok := items[0].(CopyMode)
	if !ok {
		t.Error("First item should be CopyMode")
	}
	if copyMode.Title() != "Copy file" {
		t.Errorf("Expected 'Copy file', got '%s'", copyMode.Title())
	}
	if !strings.Contains(copyMode.Description(), "copy") {
		t.Error("Copy mode description should contain 'copy'")
	}
	if copyMode.copyMode != CopyModeOptionCopy {
		t.Errorf("Expected %v, got %v", CopyModeOptionCopy, copyMode.copyMode)
	}

	// Check link mode
	linkMode, ok := items[1].(CopyMode)
	if !ok {
		t.Error("Second item should be CopyMode")
	}
	if linkMode.Title() != "Link file" {
		t.Errorf("Expected 'Link file', got '%s'", linkMode.Title())
	}
	if !strings.Contains(linkMode.Description(), "symbolic link") {
		t.Error("Link mode description should contain 'symbolic link'")
	}
	if linkMode.copyMode != CopyModeOptionLink {
		t.Errorf("Expected %v, got %v", CopyModeOptionLink, linkMode.copyMode)
	}
}

func TestNewImportRulesModel_EditorListInitialization(t *testing.T) {
	model := createTestModel(t)

	items := model.editorList.Items()
	editorConfigs := editors.GetAllEditorRuleConfigs()
	if len(items) != len(editorConfigs) {
		t.Errorf("Expected %d editor items, got %d", len(editorConfigs), len(items))
	}

	// Verify that all editors are included
	for i, item := range items {
		editor, ok := item.(editors.EditorRuleConfig)
		if !ok {
			t.Errorf("Item %d should be EditorRuleConfig", i)
			continue
		}
		if editor != editorConfigs[i] {
			t.Errorf("Editor %d mismatch", i)
		}
	}
}

// Test Init

func TestImportRulesModel_Init_WithNilFileManager(t *testing.T) {
	model := createTestModel(t)
	model.fileManager = nil

	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}

	msg := cmd()
	errorMsg, ok := msg.(ImportFileErrorMsg)
	if !ok {
		t.Errorf("Expected ImportFileErrorMsg, got %T", msg)
	}
	if !strings.Contains(errorMsg.Err.Error(), "FileManager not initialized") {
		t.Error("Error message should mention FileManager not initialized")
	}
	if errorMsg.IsOverwriteError {
		t.Error("Should not be an overwrite error")
	}
}

// Test state machine and message handling

func TestImportRulesModel_FileScanCompleteMsg(t *testing.T) {
	model, files := createTestModelWithFiles(t)

	msg := FileScanCompleteMsg{Files: files}
	updatedModel, cmd := model.Update(msg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	if result.state != StateFileSelection {
		t.Errorf("Expected state %v, got %v", StateFileSelection, result.state)
	}
	if len(result.ruleFiles) != 3 {
		t.Errorf("Expected 3 rule files, got %d", len(result.ruleFiles))
	}
	if result.err != nil {
		t.Error("Error should be nil")
	}
	if result.filePicker == nil {
		t.Error("FilePicker should not be nil")
	}
	if cmd == nil {
		t.Error("Command should not be nil") // FilePicker init command
	}
}

func TestImportRulesModel_FileScanErrorMsg(t *testing.T) {
	model := createTestModel(t)

	testErr := errors.New("scan failed")
	msg := FileScanErrorMsg{Err: testErr}
	updatedModel, cmd := model.Update(msg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	if result.state != StateError {
		t.Errorf("Expected state %v, got %v", StateError, result.state)
	}
	if result.err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, result.err)
	}
	if result.isOverwriteError {
		t.Error("Should not be an overwrite error")
	}
	if cmd != nil {
		t.Error("Command should be nil")
	}
}

func TestImportRulesModel_FileSelectedMsg(t *testing.T) {
	model, files := createTestModelWithFiles(t)
	model.state = StateFileSelection

	msg := filepicker.FileSelectedMsg{File: files[0]}
	updatedModel, cmd := model.Update(msg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	if result.state != StateEditorSelection {
		t.Errorf("Expected state %v, got %v", StateEditorSelection, result.state)
	}
	if result.selectedFile != files[0] {
		t.Error("Selected file should match")
	}
	if cmd != nil {
		t.Error("Command should be nil")
	}
}

func TestImportRulesModel_ImportFileCompleteMsg(t *testing.T) {
	model := createTestModel(t)
	model.state = StateImporting

	destPath := "/test/dest/eslint.md"
	msg := ImportFileCompleteMsg{DestPath: destPath}
	updatedModel, cmd := model.Update(msg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	if result.state != StateSuccess {
		t.Errorf("Expected state %v, got %v", StateSuccess, result.state)
	}
	if result.finalDestPath != destPath {
		t.Errorf("Expected final dest path %s, got %s", destPath, result.finalDestPath)
	}
	if result.err != nil {
		t.Error("Error should be nil")
	}
	if cmd != nil {
		t.Error("Command should be nil")
	}
}

func TestImportRulesModel_ImportFileErrorMsg_WithOverwrite(t *testing.T) {
	model := createTestModel(t)
	model.state = StateImporting

	testErr := errors.New("file already exists")
	msg := ImportFileErrorMsg{Err: testErr, IsOverwriteError: true}
	updatedModel, cmd := model.Update(msg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	if result.state != StateConfirmation {
		t.Errorf("Expected state %v, got %v", StateConfirmation, result.state)
	}
	if result.err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, result.err)
	}
	if !result.isOverwriteError {
		t.Error("Should be an overwrite error")
	}
	if cmd != nil {
		t.Error("Command should be nil")
	}
}

func TestImportRulesModel_ImportFileErrorMsg_WithoutOverwrite(t *testing.T) {
	model := createTestModel(t)
	model.state = StateImporting

	testErr := errors.New("permission denied")
	msg := ImportFileErrorMsg{Err: testErr, IsOverwriteError: false}
	updatedModel, cmd := model.Update(msg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	if result.state != StateError {
		t.Errorf("Expected state %v, got %v", StateError, result.state)
	}
	if result.err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, result.err)
	}
	if result.isOverwriteError {
		t.Error("Should not be an overwrite error")
	}
	if cmd != nil {
		t.Error("Command should be nil")
	}
}

func TestImportRulesModel_WindowSizeMsg(t *testing.T) {
	model := createTestModel(t)

	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, cmd := model.Update(windowMsg)
	_, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	// Layout is a value type, verify window resize worked by checking model is not nil
	if cmd != nil {
		t.Error("Command should be nil unless filePicker exists") // No commands expected unless filePicker exists
	}
}

func TestImportRulesModel_WindowSizeMsg_WithFilePicker(t *testing.T) {
	model, files := createTestModelWithFiles(t)

	// Set up file picker
	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, cmd := model.Update(windowMsg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	if result.filePicker == nil {
		t.Error("FilePicker should not be nil")
	}
	// Command may or may not be present depending on FilePicker implementation
	_ = cmd
}

// Test key handling in different states

func TestImportRulesModel_KeyHandling_FileSelection(t *testing.T) {
	model, files := createTestModelWithFiles(t)

	// Setup file selection state
	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	tests := []struct {
		name        string
		key         string
		expectsCmd  bool
		cmdContains string
	}{
		{"quit key", KeyQuit, true, "NavigateToMainMenuMsg"},
		{"escape key", KeyEscape, true, "NavigateToMainMenuMsg"},
		{"other key", "j", true, ""}, // FilePicker may return commands for navigation keys
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updatedModel, cmd := model.Update(keyMsg)

			if updatedModel == nil {
				t.Error("Updated model should not be nil")
			}
			if tt.expectsCmd && cmd == nil {
				t.Error("Expected a command but got nil")
			}
			if !tt.expectsCmd && cmd != nil {
				t.Error("Did not expect a command but got one")
			}
		})
	}
}

func TestImportRulesModel_KeyHandling_EditorSelection(t *testing.T) {
	model := createTestModel(t)
	model.state = StateEditorSelection

	// Setup editor list with items
	editorConfigs := editors.GetAllEditorRuleConfigs()
	if len(editorConfigs) > 0 {
		items := make([]list.Item, len(editorConfigs))
		for i, editor := range editorConfigs {
			items[i] = editor
		}
		model.editorList.SetItems(items)
		model.editorList.Select(0)
	}

	tests := []struct {
		name          string
		key           string
		expectedState ImportRulesModelState
	}{
		{"quit returns to file selection", KeyQuit, StateFileSelection},
		{"escape returns to file selection", KeyEscape, StateFileSelection},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := *model // Copy model for test
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updatedModel, _ := testModel.Update(keyMsg)
			result, ok := updatedModel.(*ImportRulesModel)
			if !ok {
				t.Error("Update should return ImportRulesModel")
			}

			if result.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, result.state)
			}
		})
	}
}

func TestImportRulesModel_KeyHandling_EditorSelection_Enter(t *testing.T) {
	model := createTestModel(t)
	model.state = StateEditorSelection

	// Setup editor list with items
	editorConfigs := editors.GetAllEditorRuleConfigs()
	if len(editorConfigs) > 0 {
		items := make([]list.Item, len(editorConfigs))
		for i, editor := range editorConfigs {
			items[i] = editor
		}
		model.editorList.SetItems(items)
		model.editorList.Select(0)

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(KeyEnter)}
		updatedModel, cmd := model.Update(keyMsg)
		result, ok := updatedModel.(*ImportRulesModel)
		if !ok {
			t.Error("Update should return ImportRulesModel")
		}

		if result.state != StateImportModeSelection {
			t.Errorf("Expected state %v, got %v", StateImportModeSelection, result.state)
		}
		if result.selectedEditor != editorConfigs[0] {
			t.Error("Selected editor should match")
		}
		if cmd != nil {
			t.Error("Command should be nil")
		}
	}
}

func TestImportRulesModel_KeyHandling_ImportModeSelection(t *testing.T) {
	model := createTestModel(t)
	model.state = StateImportModeSelection

	// Select first import mode
	model.importModeList.Select(0)

	tests := []struct {
		name          string
		key           string
		expectedState ImportRulesModelState
	}{
		{"quit returns to editor selection", KeyQuit, StateEditorSelection},
		{"escape returns to editor selection", KeyEscape, StateEditorSelection},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := *model // Copy model for test
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updatedModel, _ := testModel.Update(keyMsg)
			result, ok := updatedModel.(*ImportRulesModel)
			if !ok {
				t.Error("Update should return ImportRulesModel")
			}

			if result.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, result.state)
			}
		})
	}
}

func TestImportRulesModel_KeyHandling_ImportModeSelection_Enter(t *testing.T) {
	model := createTestModel(t)
	model.state = StateImportModeSelection

	// Select first import mode
	model.importModeList.Select(0)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(KeyEnter)}
	updatedModel, cmd := model.Update(keyMsg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	if result.state != StateConfirmation {
		t.Errorf("Expected state %v, got %v", StateConfirmation, result.state)
	}
	if result.selectedImportMode == (CopyMode{}) {
		t.Error("Selected import mode should not be empty")
	}
	if cmd != nil {
		t.Error("Command should be nil")
	}
}

func TestImportRulesModel_KeyHandling_Confirmation(t *testing.T) {
	model, files := createTestModelWithFiles(t)
	model.state = StateConfirmation
	model.selectedFile = files[0]
	model.selectedEditor = editors.GetAllEditorRuleConfigs()[0]
	model.selectedImportMode = CopyMode{title: "Copy", copyMode: CopyModeOptionCopy}

	tests := []struct {
		name          string
		key           string
		expectedState ImportRulesModelState
		expectsCmd    bool
	}{
		{"quit returns to import mode", KeyQuit, StateImportModeSelection, false},
		{"escape returns to import mode", KeyEscape, StateImportModeSelection, false},
		{"no returns to import mode", KeyNo, StateImportModeSelection, false},
		{"yes starts importing", KeyYes, StateImporting, true},
		{"enter starts importing", KeyEnter, StateImporting, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := *model // Copy model for test
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updatedModel, cmd := testModel.Update(keyMsg)
			result, ok := updatedModel.(*ImportRulesModel)
			if !ok {
				t.Error("Update should return ImportRulesModel")
			}

			if result.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, result.state)
			}
			if tt.expectsCmd && cmd == nil {
				t.Error("Expected a command but got nil")
			}
			if !tt.expectsCmd && cmd != nil {
				t.Error("Did not expect a command but got one")
			}
		})
	}
}

func TestImportRulesModel_KeyHandling_Error(t *testing.T) {
	model := createTestModel(t)
	model.state = StateError
	model.err = errors.New("test error")

	tests := []struct {
		name          string
		key           string
		expectedState ImportRulesModelState
		setup         func(*ImportRulesModel)
	}{
		{
			name:          "retry with overwrite error",
			key:           KeyRetry,
			expectedState: StateConfirmation,
			setup: func(m *ImportRulesModel) {
				m.isOverwriteError = true
			},
		},
		{
			name:          "retry without overwrite error",
			key:           KeyRetry,
			expectedState: StateLoading,
			setup: func(m *ImportRulesModel) {
				m.isOverwriteError = false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := *model // Copy model for test
			if tt.setup != nil {
				tt.setup(&testModel)
			}

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updatedModel, _ := testModel.Update(keyMsg)
			result, ok := updatedModel.(*ImportRulesModel)
			if !ok {
				t.Error("Update should return ImportRulesModel")
			}

			if result.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, result.state)
			}
		})
	}
}

func TestImportRulesModel_KeyHandling_Success(t *testing.T) {
	model := createTestModel(t)
	model.state = StateSuccess
	model.finalDestPath = "/test/dest/file.md"

	tests := []struct {
		name          string
		key           string
		expectedState ImportRulesModelState
		expectsCmd    bool
	}{
		{"menu returns to main menu", KeyMenu, StateSuccess, true}, // State doesn't change but command sent
		{"again resets to file selection", KeyAgain, StateFileSelection, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModel := *model // Copy model for test
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updatedModel, cmd := testModel.Update(keyMsg)
			result, ok := updatedModel.(*ImportRulesModel)
			if !ok {
				t.Error("Update should return ImportRulesModel")
			}

			if result.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, result.state)
			}
			if tt.expectsCmd && cmd == nil {
				t.Error("Expected a command but got nil")
			}
			if !tt.expectsCmd && cmd != nil {
				t.Error("Did not expect a command but got one")
			}

			if tt.key == KeyAgain {
				// Verify state was reset
				if result.selectedFile != (filemanager.FileItem{}) {
					t.Error("Selected file should be reset")
				}
				if result.selectedEditor != (editors.EditorRuleConfig{}) {
					t.Error("Selected editor should be reset")
				}
				if result.selectedImportMode != (CopyMode{}) {
					t.Error("Selected import mode should be reset")
				}
			}
		})
	}
}

// Test filtering

func TestImportRulesModel_FilterMatches_EditorSelection(t *testing.T) {
	model := createTestModel(t)
	model.state = StateEditorSelection

	filterMsg := list.FilterMatchesMsg{}
	updatedModel, cmd := model.Update(filterMsg)

	if updatedModel == nil {
		t.Error("Updated model should not be nil")
	}
	// Command may or may not be present depending on list implementation
	_ = cmd
}

func TestImportRulesModel_FilterMatches_ImportModeSelection(t *testing.T) {
	model := createTestModel(t)
	model.state = StateImportModeSelection

	filterMsg := list.FilterMatchesMsg{}
	updatedModel, cmd := model.Update(filterMsg)

	if updatedModel == nil {
		t.Error("Updated model should not be nil")
	}
	// Command may or may not be present depending on list implementation
	_ = cmd
}

// Test validation functions

func TestImportRulesModel_ValidateEditorSelection(t *testing.T) {
	model := createTestModel(t)

	// Test with no selection
	model.editorList.SetItems([]list.Item{})
	if model.validateEditorSelection() {
		t.Error("Validation should fail with no selection")
	}

	// Test with valid selection
	editorConfigs := editors.GetAllEditorRuleConfigs()
	if len(editorConfigs) > 0 {
		items := make([]list.Item, len(editorConfigs))
		for i, editor := range editorConfigs {
			items[i] = editor
		}
		model.editorList.SetItems(items)
		model.editorList.Select(0)
		if !model.validateEditorSelection() {
			t.Error("Validation should pass with valid selection")
		}
	}
}

func TestImportRulesModel_ValidateImportModeSelection(t *testing.T) {
	model := createTestModel(t)

	// Test with valid selection (import modes are set up by default)
	items := model.importModeList.Items()
	if len(items) > 0 {
		model.importModeList.Select(0)
		if !model.validateImportModeSelection() {
			t.Error("Validation should pass with valid selection")
		}
	}

	// Test with no items
	model.importModeList.SetItems([]list.Item{})
	if model.validateImportModeSelection() {
		t.Error("Validation should fail with no items")
	}
}

// Test reset functionality

func TestImportRulesModel_ResetSelectionState(t *testing.T) {
	model, files := createTestModelWithFiles(t)

	// Set some state
	model.selectedFile = files[0]
	model.selectedEditor = editors.GetAllEditorRuleConfigs()[0]
	model.selectedImportMode = CopyMode{title: "Copy", copyMode: CopyModeOptionCopy}

	// Reset
	model.resetSelectionState()

	// Verify state is reset
	if model.selectedFile != (filemanager.FileItem{}) {
		t.Error("Selected file should be reset")
	}
	if model.selectedEditor != (editors.EditorRuleConfig{}) {
		t.Error("Selected editor should be reset")
	}
	if model.selectedImportMode != (CopyMode{}) {
		t.Error("Selected import mode should be reset")
	}
}

// Test command generation

func TestImportRulesModel_ScanForFilesCmd(t *testing.T) {
	model, _ := createTestModelWithFiles(t)

	cmd := model.scanForFilesCmd()
	if cmd == nil {
		t.Error("Command should not be nil")
	}

	// Execute the command
	msg := cmd()
	if msg == nil {
		t.Error("Message should not be nil")
	}

	// Should return FileScanCompleteMsg since we have real files
	completeMsg, ok := msg.(FileScanCompleteMsg)
	if !ok {
		t.Errorf("Expected FileScanCompleteMsg, got %T", msg)
	}
	if len(completeMsg.Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(completeMsg.Files))
	}
}

func TestImportRulesModel_ScanForFilesCmd_EmptyDirectory(t *testing.T) {
	model := createTestModel(t)

	cmd := model.scanForFilesCmd()
	if cmd == nil {
		t.Error("Command should not be nil")
	}

	// Execute the command
	msg := cmd()
	if msg == nil {
		t.Error("Message should not be nil")
	}

	// Should return FileScanCompleteMsg with empty files
	completeMsg, ok := msg.(FileScanCompleteMsg)
	if !ok {
		t.Errorf("Expected FileScanCompleteMsg, got %T", msg)
	}
	if len(completeMsg.Files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(completeMsg.Files))
	}
}

func TestImportRulesModel_SaveFileCmd_Copy(t *testing.T) {
	model, files := createTestModelWithFiles(t)
	model.selectedFile = files[0]
	model.selectedEditor = editors.GetAllEditorRuleConfigs()[0]
	model.selectedImportMode = CopyMode{copyMode: CopyModeOptionCopy}

	cmd := model.saveFileCmd(false)
	if cmd == nil {
		t.Error("Command should not be nil")
	}

	// Execute the command
	msg := cmd()
	if msg == nil {
		t.Error("Message should not be nil")
	}

	// Check if it's success or error
	switch m := msg.(type) {
	case ImportFileCompleteMsg:
		if m.DestPath == "" {
			t.Error("Destination path should not be empty")
		}
	case ImportFileErrorMsg:
		if m.Err == nil {
			t.Error("Error should not be nil")
		}
	default:
		t.Errorf("Unexpected message type: %T", msg)
	}
}

func TestImportRulesModel_SaveFileCmd_Link(t *testing.T) {
	model, files := createTestModelWithFiles(t)
	model.selectedFile = files[0]
	model.selectedEditor = editors.GetAllEditorRuleConfigs()[0]
	model.selectedImportMode = CopyMode{copyMode: CopyModeOptionLink}

	cmd := model.saveFileCmd(false)
	if cmd == nil {
		t.Error("Command should not be nil")
	}

	// Execute the command
	msg := cmd()
	if msg == nil {
		t.Error("Message should not be nil")
	}

	// Check if it's success or error
	switch m := msg.(type) {
	case ImportFileCompleteMsg:
		if m.DestPath == "" {
			t.Error("Destination path should not be empty")
		}
	case ImportFileErrorMsg:
		if m.Err == nil {
			t.Error("Error should not be nil")
		}
	default:
		t.Errorf("Unexpected message type: %T", msg)
	}
}

func TestImportRulesModel_SaveFileCmd_OverwriteError(t *testing.T) {
	model, files := createTestModelWithFiles(t)
	model.selectedFile = files[0]
	model.selectedEditor = editors.GetAllEditorRuleConfigs()[0]
	model.selectedImportMode = CopyMode{copyMode: CopyModeOptionCopy}

	// Create a file that will conflict
	destPath := model.selectedEditor.GenerateRuleFileFullPath(model.selectedFile.Name)

	// Ensure directory exists
	destDir := filepath.Dir(destPath)
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	err = os.WriteFile(destPath, []byte("existing content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create conflicting file: %v", err)
	}

	cmd := model.saveFileCmd(false) // Don't allow overwrite
	if cmd == nil {
		t.Error("Command should not be nil")
	}

	msg := cmd()
	if msg == nil {
		t.Error("Message should not be nil")
	}

	errorMsg, ok := msg.(ImportFileErrorMsg)
	if !ok {
		t.Errorf("Expected ImportFileErrorMsg, got %T", msg)
	}
	if !errorMsg.IsOverwriteError {
		t.Error("Should be an overwrite error")
	}
	if !strings.Contains(strings.ToLower(errorMsg.Err.Error()), "exists") {
		t.Error("Error message should mention file exists")
	}
}

// Test CopyMode methods

func TestCopyMode_Methods(t *testing.T) {
	copyMode := CopyMode{
		title:       "Test Title",
		description: "Test Description",
		copyMode:    CopyModeOptionCopy,
	}

	if copyMode.Title() != "Test Title" {
		t.Errorf("Expected 'Test Title', got '%s'", copyMode.Title())
	}
	if copyMode.Description() != "Test Description" {
		t.Errorf("Expected 'Test Description', got '%s'", copyMode.Description())
	}
	if copyMode.FilterValue() != "Test Title Test Description" {
		t.Errorf("Expected 'Test Title Test Description', got '%s'", copyMode.FilterValue())
	}
}

// Test edge cases and error conditions

func TestImportRulesModel_KeyHandling_UnknownState(t *testing.T) {
	model := createTestModel(t)
	model.state = ImportRulesModelState(999) // Invalid state

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	updatedModel, cmd := model.Update(keyMsg)

	if updatedModel == nil {
		t.Error("Updated model should not be nil")
	}
	if cmd != nil {
		t.Error("Command should be nil for unknown state")
	}
}

func TestImportRulesModel_KeyHandling_EditorSelection_NoValidation(t *testing.T) {
	model := createTestModel(t)
	model.state = StateEditorSelection
	// Don't set up any editors

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(KeyEnter)}
	updatedModel, cmd := model.Update(keyMsg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	// Should transition to import mode selection (validation passes with default state)
	if result.state != StateImportModeSelection {
		t.Errorf("Expected state %v, got %v", StateImportModeSelection, result.state)
	}
	if cmd != nil {
		t.Error("Command should be nil")
	}
}

func TestImportRulesModel_KeyHandling_ImportModeSelection_NoValidation(t *testing.T) {
	model := createTestModel(t)
	model.state = StateImportModeSelection
	// Clear import mode list
	model.importModeList.SetItems([]list.Item{})

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(KeyEnter)}
	updatedModel, cmd := model.Update(keyMsg)
	result, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}

	// Should remain in same state due to validation failure
	if result.state != StateImportModeSelection {
		t.Errorf("Expected state %v, got %v", StateImportModeSelection, result.state)
	}
	if cmd != nil {
		t.Error("Command should be nil")
	}
}

func TestImportRulesModel_KeyHandling_Filtering(t *testing.T) {
	model := createTestModel(t)

	// Test editor selection filtering
	model.state = StateEditorSelection
	model.editorList.SetFilteringEnabled(true)
	// Simulate filter state

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	updatedModel, cmd := model.Update(keyMsg)

	if updatedModel == nil {
		t.Error("Updated model should not be nil")
	}
	// Command may or may not be present depending on list implementation
	_ = cmd
}

// Integration test - complete workflow

func TestImportRulesModel_CompleteWorkflow(t *testing.T) {
	model, files := createTestModelWithFiles(t)

	// 1. Start with loading state and receive scan complete
	if model.state != StateLoading {
		t.Errorf("Expected initial state %v, got %v", StateLoading, model.state)
	}

	scanMsg := FileScanCompleteMsg{Files: files}
	updatedModel, _ := model.Update(scanMsg)
	model, ok := updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}
	if model.state != StateFileSelection {
		t.Errorf("Expected state %v, got %v", StateFileSelection, model.state)
	}
	if len(model.ruleFiles) != 3 {
		t.Errorf("Expected 3 rule files, got %d", len(model.ruleFiles))
	}

	// 2. Select a file
	fileSelectedMsg := filepicker.FileSelectedMsg{File: files[0]}
	updatedModel, _ = model.Update(fileSelectedMsg)
	model, ok = updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}
	if model.state != StateEditorSelection {
		t.Errorf("Expected state %v, got %v", StateEditorSelection, model.state)
	}
	if model.selectedFile != files[0] {
		t.Error("Selected file should match")
	}

	// 3. Select an editor
	editorConfigs := editors.GetAllEditorRuleConfigs()
	if len(editorConfigs) > 0 {
		items := make([]list.Item, len(editorConfigs))
		for i, editor := range editorConfigs {
			items[i] = editor
		}
		model.editorList.SetItems(items)
		model.editorList.Select(0)

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(KeyEnter)}
		updatedModel, _ = model.Update(keyMsg)
		model, ok = updatedModel.(*ImportRulesModel)
		if !ok {
			t.Error("Update should return ImportRulesModel")
		}
		if model.state != StateImportModeSelection {
			t.Errorf("Expected state %v, got %v", StateImportModeSelection, model.state)
		}
		if model.selectedEditor != editorConfigs[0] {
			t.Error("Selected editor should match")
		}
	}

	// 4. Select import mode
	model.importModeList.Select(0)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(KeyEnter)}
	updatedModel, _ = model.Update(keyMsg)
	model, ok = updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}
	if model.state != StateConfirmation {
		t.Errorf("Expected state %v, got %v", StateConfirmation, model.state)
	}
	if model.selectedImportMode == (CopyMode{}) {
		t.Error("Selected import mode should not be empty")
	}

	// 5. Confirm import
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(KeyYes)}
	updatedModel, cmd := model.Update(keyMsg)
	model, ok = updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}
	if model.state != StateImporting {
		t.Errorf("Expected state %v, got %v", StateImporting, model.state)
	}
	if cmd == nil {
		t.Error("Command should not be nil")
	}

	// 6. Simulate successful import
	destPath := "/dest/eslint.md"
	importCompleteMsg := ImportFileCompleteMsg{DestPath: destPath}
	updatedModel, _ = model.Update(importCompleteMsg)
	model, ok = updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}
	if model.state != StateSuccess {
		t.Errorf("Expected state %v, got %v", StateSuccess, model.state)
	}
	if model.finalDestPath != destPath {
		t.Errorf("Expected final dest path %s, got %s", destPath, model.finalDestPath)
	}

	// 7. Reset for another import
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(KeyAgain)}
	updatedModel, _ = model.Update(keyMsg)
	model, ok = updatedModel.(*ImportRulesModel)
	if !ok {
		t.Error("Update should return ImportRulesModel")
	}
	if model.state != StateFileSelection {
		t.Errorf("Expected state %v, got %v", StateFileSelection, model.state)
	}
	if model.selectedFile != (filemanager.FileItem{}) {
		t.Error("Selected file should be reset")
	}
}

// TestImportRulesModel_FilePickerCanReadFiles tests that the FilePicker receives
// absolute paths and can actually read file contents. This catches the regression
// where ScanCentralRepo returns relative paths but FilePicker needs absolute paths.
func TestImportRulesModel_FilePickerCanReadFiles(t *testing.T) {
	// Create storage directory with test files
	storageDir := createTestStorageDir(t)

	// Create test files with actual content in storage
	testFiles := map[string]string{
		"rule1.md":       "# Rule 1\nThis is the first rule.",
		"rule2.md":       "# Rule 2\nThis is the second rule.",
		"docs/nested.md": "# Nested Rule\nThis is a nested rule file.",
	}

	for relPath, content := range testFiles {
		fullPath := filepath.Join(storageDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", relPath, err)
		}
	}

	// Create model
	ctx := createTestUIContext(t)
	ctx.Config.StorageDir = storageDir
	model := NewImportRulesModel(ctx)

	// Initialize and trigger file scan
	initCmd := model.Init()
	if initCmd == nil {
		t.Fatal("Init command should not be nil")
	}

	// Execute scan command
	scanCmd := model.scanForFilesCmd()
	scanResult := scanCmd()

	scanCompleteMsg, ok := scanResult.(FileScanCompleteMsg)
	if !ok {
		t.Fatalf("Expected FileScanCompleteMsg, got %T", scanResult)
	}

	// Process the scan complete message to set up FilePicker
	updatedModel, _ := model.Update(scanCompleteMsg)
	model, ok = updatedModel.(*ImportRulesModel)
	if !ok {
		t.Fatal("Update should return ImportRulesModel")
	}

	// Verify we have files and they have absolute paths
	if len(model.ruleFiles) == 0 {
		t.Fatal("Should have found rule files")
	}

	// Test that all files in ruleFiles have absolute paths
	for i, file := range model.ruleFiles {
		if !filepath.IsAbs(file.Path) {
			t.Errorf("File %d (%s) should have absolute path, got: %s", i, file.Name, file.Path)
		}

		// Verify the file actually exists and can be read
		content, err := os.ReadFile(file.Path)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", file.Path, err)
			continue
		}

		// Verify content matches what we expect
		expectedContent, exists := testFiles[filepath.Base(file.Path)]
		if !exists {
			// Check if it's a nested file
			for originalPath, expectedContent := range testFiles {
				if filepath.Base(originalPath) == file.Name {
					if string(content) != expectedContent {
						t.Errorf("Content mismatch for file %s: expected %q, got %q", file.Name, expectedContent, string(content))
					}
					break
				}
			}
		} else {
			if string(content) != expectedContent {
				t.Errorf("Content mismatch for file %s: expected %q, got %q", file.Name, expectedContent, string(content))
			}
		}
	}

	// Verify FilePicker was created and can access the files
	if model.filePicker == nil {
		t.Fatal("FilePicker should be created after scan complete")
	}

	// Verify that the paths are within the storage directory
	for _, file := range model.ruleFiles {
		if !strings.HasPrefix(file.Path, storageDir) {
			t.Errorf("File path %s should be within storage directory %s", file.Path, storageDir)
		}
	}
}

// TestImportRulesModel_RelativeToAbsoluteConversion tests the conversion from
// relative paths (from ScanCentralRepo) to absolute paths (for FilePicker)
func TestImportRulesModel_RelativeToAbsoluteConversion(t *testing.T) {
	storageDir := createTestStorageDir(t)

	// Create a test file
	testContent := "# Test Rule\nThis is a test rule."
	testFile := filepath.Join(storageDir, "test.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := createTestUIContext(t)
	ctx.Config.StorageDir = storageDir

	// Create FileManager directly to test the conversion
	fm, err := filemanager.NewFileManager(storageDir, ctx.Logger)
	if err != nil {
		t.Fatalf("Failed to create FileManager: %v", err)
	}

	// Get files with relative paths
	relativeFiles, err := fm.ScanCentralRepo()
	if err != nil {
		t.Fatalf("ScanCentralRepo failed: %v", err)
	}

	if len(relativeFiles) == 0 {
		t.Fatal("Should have found at least one file")
	}

	// Verify files have relative paths
	for _, file := range relativeFiles {
		if filepath.IsAbs(file.Path) {
			t.Errorf("ScanCentralRepo should return relative paths, got absolute: %s", file.Path)
		}
	}

	// Convert to absolute paths
	absoluteFiles := fm.ConvertToAbsolutePaths(relativeFiles)

	// Verify conversion worked
	if len(absoluteFiles) != len(relativeFiles) {
		t.Errorf("Conversion should preserve count: relative=%d, absolute=%d", len(relativeFiles), len(absoluteFiles))
	}

	for i, file := range absoluteFiles {
		if !filepath.IsAbs(file.Path) {
			t.Errorf("Converted file %d should have absolute path, got: %s", i, file.Path)
		}

		// Verify file can be read
		content, err := os.ReadFile(file.Path)
		if err != nil {
			t.Errorf("Failed to read converted file %s: %v", file.Path, err)
		}

		if string(content) != testContent {
			t.Errorf("Content mismatch: expected %q, got %q", testContent, string(content))
		}
	}
}
