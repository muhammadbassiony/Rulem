// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"rulem/internal/repository"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestIntegration_EditNameComplete tests the complete
// edit name flow from start to finish with successful completion
func TestIntegration_EditNameComplete(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	// Set up existing repository
	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "test-repo-id",
			Name: "Original Name",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	// Step 1: Select repository and navigate to edit name
	m.selectedRepositoryID = "test-repo-id"
	m.state = SettingsStateUpdateRepoName
	m.textInput.SetValue("Updated Repository Name")

	// Step 2: Submit new name (should transition to confirmation)
	m, cmd := m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("should not return command yet")
	}
	if m.state != SettingsStateEditNameConfirm {
		t.Errorf("should be at confirmation state, got %v", m.state)
	}

	// Step 3: Confirm the change
	m, cmd = m.handleEditNameConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("should return saveChanges command")
	}

	// Step 4: Execute saveChanges command
	msg := cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatal("should return settingsCompleteMsg")
	}

	// Step 5: Handle the complete message
	model, _ := m.Update(msg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateComplete {
		t.Errorf("should be at Complete state, got %v", m.state)
	}
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].Name != "Updated Repository Name" {
		t.Errorf("expected name 'Updated Repository Name', got %q", m.currentConfig.Repositories[0].Name)
	}
	if m.currentConfig.Repositories[0].ID != "test-repo-id" {
		t.Errorf("ID should not change, got %q", m.currentConfig.Repositories[0].ID)
	}
}

// TestIntegration_EditNameLocalRepository tests editing name for local repository
func TestIntegration_EditNameLocalRepository(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "local-repo",
			Name: "Local Repo",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	m.selectedRepositoryID = "local-repo"
	m.state = SettingsStateUpdateRepoName
	m.textInput.SetValue("Renamed Local Repo")

	m, _ = m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateEditNameConfirm {
		t.Errorf("expected EditNameConfirm state, got %v", m.state)
	}

	m, cmd := m.handleEditNameConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected save command")
	}

	msg := cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatal("expected settingsCompleteMsg")
	}

	model, _ := m.Update(msg)
	m = model.(*SettingsModel)

	if m.currentConfig.Repositories[0].Name != "Renamed Local Repo" {
		t.Errorf("expected name 'Renamed Local Repo', got %q", m.currentConfig.Repositories[0].Name)
	}
	if m.currentConfig.Repositories[0].Type != repository.RepositoryTypeLocal {
		t.Errorf("expected type Local, got %v", m.currentConfig.Repositories[0].Type)
	}
}

// TestIntegration_EditNameGitHubRepository tests editing name for GitHub repository
func TestIntegration_EditNameGitHubRepository(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	testPath := t.TempDir()
	testURL := "https://github.com/test/repo"
	testBranch := "main"
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "github-repo",
			Name:      "GitHub Repo",
			Type:      repository.RepositoryTypeGitHub,
			Path:      testPath,
			RemoteURL: &testURL,
			Branch:    &testBranch,
		},
	}

	m.selectedRepositoryID = "github-repo"
	m.state = SettingsStateUpdateRepoName
	m.textInput.SetValue("Renamed GitHub Repo")

	m, _ = m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateEditNameConfirm {
		t.Errorf("expected EditNameConfirm state, got %v", m.state)
	}

	m, cmd := m.handleEditNameConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected save command")
	}

	msg := cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatal("expected settingsCompleteMsg")
	}

	model, _ := m.Update(msg)
	m = model.(*SettingsModel)

	if m.currentConfig.Repositories[0].Name != "Renamed GitHub Repo" {
		t.Errorf("expected name 'Renamed GitHub Repo', got %q", m.currentConfig.Repositories[0].Name)
	}
	if m.currentConfig.Repositories[0].Type != repository.RepositoryTypeGitHub {
		t.Errorf("expected type GitHub, got %v", m.currentConfig.Repositories[0].Type)
	}
	if *m.currentConfig.Repositories[0].RemoteURL != testURL {
		t.Errorf("URL should not change, got %q", *m.currentConfig.Repositories[0].RemoteURL)
	}
}

// TestIntegration_EditNameCancel tests canceling the edit name flow
func TestIntegration_EditNameCancel(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "test-repo",
			Name: "Original Name",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	m.selectedRepositoryID = "test-repo"
	m.state = SettingsStateUpdateRepoName
	m.textInput.SetValue("New Name")

	// Cancel with Esc
	m, _ = m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateRepositoryActions {
		t.Errorf("should return to repository actions, got %v", m.state)
	}
	if m.currentConfig.Repositories[0].Name != "Original Name" {
		t.Errorf("name should not change, got %q", m.currentConfig.Repositories[0].Name)
	}
}

// TestIntegration_EditNameEmptyValidation tests validation error for empty name
func TestIntegration_EditNameEmptyValidation(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "test-repo",
			Name: "Original Name",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	m.selectedRepositoryID = "test-repo"
	m.state = SettingsStateUpdateRepoName
	m.textInput.SetValue("")

	m, cmd := m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateUpdateRepoName {
		t.Errorf("should stay in edit state, got %v", m.state)
	}
	if cmd == nil {
		t.Fatal("should return error command")
	}

	// Execute the error command
	msg := cmd()
	if _, ok := msg.(editNameErrorMsg); !ok {
		t.Fatalf("should return editNameErrorMsg, got %T", msg)
	}

	// Handle the error message
	model, _ := m.Update(msg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditNameError {
		t.Errorf("should be in error state, got %v", m.state)
	}
}

// TestIntegration_EditNameDuplicateValidation tests validation error for duplicate name
func TestIntegration_EditNameDuplicateValidation(t *testing.T) {
	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "repo-1",
			Name: "Repository One",
			Type: repository.RepositoryTypeLocal,
			Path: path1,
		},
		{
			ID:   "repo-2",
			Name: "Repository Two",
			Type: repository.RepositoryTypeLocal,
			Path: path2,
		},
	}

	// Try to rename repo-2 to same name as repo-1
	m.selectedRepositoryID = "repo-2"
	m.state = SettingsStateUpdateRepoName
	m.textInput.SetValue("Repository One")

	m, cmd := m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateUpdateRepoName {
		t.Errorf("should stay in edit state, got %v", m.state)
	}
	if cmd == nil {
		t.Fatal("should return error command")
	}

	// Execute the error command
	msg := cmd()
	errMsg, ok := msg.(editNameErrorMsg)
	if !ok {
		t.Fatalf("should return editNameErrorMsg, got %T", msg)
	}
	if !strings.Contains(errMsg.err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got %q", errMsg.err.Error())
	}
}

// TestIntegration_EditNameTooLongValidation tests validation error for name > 100 chars
func TestIntegration_EditNameTooLongValidation(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "test-repo",
			Name: "Original",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	m.selectedRepositoryID = "test-repo"
	m.state = SettingsStateUpdateRepoName

	// Create a name that's too long
	longName := string(make([]byte, 101))
	for i := range longName {
		longName = longName[:i] + "a" + longName[i+1:]
	}
	m.textInput.SetValue(longName)

	m, cmd := m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateUpdateRepoName {
		t.Errorf("should stay in edit state, got %v", m.state)
	}
	if cmd == nil {
		t.Fatal("should return error command")
	}

	// Execute the error command
	msg := cmd()
	if _, ok := msg.(editNameErrorMsg); !ok {
		t.Fatalf("should return editNameErrorMsg, got %T", msg)
	}
}

// TestIntegration_EditNameMultipleRepositories tests editing name with multiple repos
func TestIntegration_EditNameMultipleRepositories(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	path3 := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{ID: "repo-1", Name: "Repo One", Type: repository.RepositoryTypeLocal, Path: path1},
		{ID: "repo-2", Name: "Repo Two", Type: repository.RepositoryTypeLocal, Path: path2},
		{ID: "repo-3", Name: "Repo Three", Type: repository.RepositoryTypeLocal, Path: path3},
	}

	// Edit the middle repository
	m.selectedRepositoryID = "repo-2"
	m.state = SettingsStateUpdateRepoName
	m.textInput.SetValue("Updated Repo Two")

	m, _ = m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateEditNameConfirm {
		t.Errorf("expected EditNameConfirm state, got %v", m.state)
	}

	m, cmd := m.handleEditNameConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected save command")
	}

	msg := cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatal("expected settingsCompleteMsg")
	}

	model, _ := m.Update(msg)
	m = model.(*SettingsModel)

	// Verify only the selected repo was updated
	if m.currentConfig.Repositories[0].Name != "Repo One" {
		t.Errorf("repo 0 should be unchanged, got %q", m.currentConfig.Repositories[0].Name)
	}
	if m.currentConfig.Repositories[1].Name != "Updated Repo Two" {
		t.Errorf("repo 1 should be updated, got %q", m.currentConfig.Repositories[1].Name)
	}
	if m.currentConfig.Repositories[2].Name != "Repo Three" {
		t.Errorf("repo 2 should be unchanged, got %q", m.currentConfig.Repositories[2].Name)
	}
}

// TestIntegration_EditNamePreservesOtherFields tests that editing name
// doesn't affect other repository fields
func TestIntegration_EditNamePreservesOtherFields(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	testPath := t.TempDir()
	testURL := "https://github.com/test/repo"
	testBranch := "develop"
	createdAt := int64(1234567890)

	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "github-repo",
			Name:      "Original",
			Type:      repository.RepositoryTypeGitHub,
			Path:      testPath,
			RemoteURL: &testURL,
			Branch:    &testBranch,
			CreatedAt: createdAt,
		},
	}

	m.selectedRepositoryID = "github-repo"
	m.state = SettingsStateUpdateRepoName
	m.textInput.SetValue("New Name")

	m, _ = m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateEditNameConfirm {
		t.Errorf("expected EditNameConfirm state, got %v", m.state)
	}

	m, cmd := m.handleEditNameConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected save command")
	}

	msg := cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatal("expected settingsCompleteMsg")
	}

	model, _ := m.Update(msg)
	m = model.(*SettingsModel)

	// Verify all other fields are preserved
	repo := m.currentConfig.Repositories[0]
	if repo.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", repo.Name)
	}
	if repo.ID != "github-repo" {
		t.Errorf("expected ID 'github-repo', got %q", repo.ID)
	}
	if repo.Type != repository.RepositoryTypeGitHub {
		t.Errorf("expected type GitHub, got %v", repo.Type)
	}
	if repo.Path != testPath {
		t.Errorf("expected path %q, got %q", testPath, repo.Path)
	}
	if *repo.RemoteURL != testURL {
		t.Errorf("expected URL %q, got %q", testURL, *repo.RemoteURL)
	}
	if *repo.Branch != testBranch {
		t.Errorf("expected branch %q, got %q", testBranch, *repo.Branch)
	}
	if repo.CreatedAt != createdAt {
		t.Errorf("expected createdAt %d, got %d", createdAt, repo.CreatedAt)
	}
}

// TestIntegration_EditNameRecoverFromValidationError tests recovering from
// validation error and successfully completing
func TestIntegration_EditNameRecoverFromValidationError(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "test-repo",
			Name: "Original",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	m.selectedRepositoryID = "test-repo"
	m.state = SettingsStateUpdateRepoName

	// Step 1: Try empty name (should fail)
	m.textInput.SetValue("")
	m, cmd := m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateUpdateRepoName {
		t.Errorf("should stay in edit state, got %v", m.state)
	}
	if cmd == nil {
		t.Fatal("should return error command")
	}

	// Handle error message
	msg := cmd()
	if _, ok := msg.(editNameErrorMsg); !ok {
		t.Fatalf("expected editNameErrorMsg, got %T", msg)
	}
	model, _ := m.Update(msg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditNameError {
		t.Errorf("should be in error state, got %v", m.state)
	}

	// Step 2: Dismiss error and return to edit state
	m.state = SettingsStateUpdateRepoName // Simulate returning to edit after dismissing error

	// Step 3: Fix with valid name
	m.textInput.SetValue("Valid New Name")
	m, _ = m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateEditNameConfirm {
		t.Errorf("should be at confirmation state, got %v", m.state)
	}

	m, cmd = m.handleEditNameConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected save command")
	}

	msg = cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatalf("expected settingsCompleteMsg, got %T", msg)
	}

	model, _ = m.Update(msg)
	m = model.(*SettingsModel)

	if m.currentConfig.Repositories[0].Name != "Valid New Name" {
		t.Errorf("expected name 'Valid New Name', got %q", m.currentConfig.Repositories[0].Name)
	}
}

// TestIntegration_EditNameStateIsolation tests that edit name flow
// doesn't interfere with other state
func TestIntegration_EditNameStateIsolation(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "test-repo",
			Name: "Original",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	// Set state from other flows
	m.newGitHubURL = "https://github.com/test/repo"
	m.newGitHubBranch = "main"
	m.newGitHubPath = "/some/path"

	// Edit repository name
	m.selectedRepositoryID = "test-repo"
	m.state = SettingsStateUpdateRepoName
	m.textInput.SetValue("New Name")

	m, _ = m.handleUpdateRepoNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify other flow state is preserved
	if m.newGitHubURL != "https://github.com/test/repo" {
		t.Errorf("newGitHubURL should be preserved, got %q", m.newGitHubURL)
	}
	if m.newGitHubBranch != "main" {
		t.Errorf("newGitHubBranch should be preserved, got %q", m.newGitHubBranch)
	}
	if m.newGitHubPath != "/some/path" {
		t.Errorf("newGitHubPath should be preserved, got %q", m.newGitHubPath)
	}
}
