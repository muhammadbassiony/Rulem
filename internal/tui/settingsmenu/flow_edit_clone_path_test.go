// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"errors"
	"strings"
	"testing"

	"rulem/internal/repository"

	tea "github.com/charmbracelet/bubbletea"
)

// TestIntegration_EditClonePathComplete tests the complete
// edit clone path flow from start to finish with successful completion (clean repo)
func TestIntegration_EditClonePathComplete(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	// Set up existing GitHub repository
	testPath := t.TempDir()
	newPath := t.TempDir()
	testURL := "https://github.com/test/repo"
	testBranch := "main"
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "github-repo-id",
			Name:      "Test GitHub Repo",
			Type:      repository.RepositoryTypeGitHub,
			Path:      testPath,
			RemoteURL: &testURL,
			Branch:    &testBranch,
		},
	}

	// Step 1: Select repository and navigate to edit clone path
	m.selectedRepositoryID = "github-repo-id"
	m.state = SettingsStateUpdateGitHubPath
	m.textInput.SetValue(newPath)

	// Step 2: Submit new path (triggers dirty check)
	m, cmd := m.handleUpdateGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should return dirty check command")
	}

	// Step 3: Simulate clean repository response
	cleanMsg := editClonePathDirtyStateMsg{isDirty: false, err: nil}

	// Step 4: Handle dirty check result (should transition to confirm)
	model, _ := m.Update(cleanMsg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditClonePathConfirm {
		t.Fatalf("expected state %v, got %v", SettingsStateEditClonePathConfirm, m.state)
	}

	// Step 5: Confirm the change
	m, cmd = m.handleEditClonePathConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should return saveChanges command")
	}

	// Step 6: Execute saveChanges command
	completeMsg := cmd()
	if _, ok := completeMsg.(settingsCompleteMsg); !ok {
		t.Fatalf("should return settingsCompleteMsg, got %T", completeMsg)
	}

	// Step 7: Handle the complete message
	model, _ = m.Update(completeMsg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateComplete {
		t.Fatalf("expected state %v, got %v", SettingsStateComplete, m.state)
	}

	// Step 8: Verify path was updated
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].Path == "" {
		t.Fatalf("expected repository path to be non-empty")
	}
	if m.currentConfig.Repositories[0].Path == testPath {
		t.Fatalf("expected path to change from %q", testPath)
	}
	if m.currentConfig.Repositories[0].ID != "github-repo-id" {
		t.Fatalf("expected ID %q, got %q", "github-repo-id", m.currentConfig.Repositories[0].ID)
	}

	_ = configPath
}

// TestIntegration_EditClonePathDirtyRepository tests editing clone path
// when repository has uncommitted changes
func TestIntegration_EditClonePathDirtyRepository(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	newPath := t.TempDir()
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
	m.state = SettingsStateUpdateGitHubPath
	m.textInput.SetValue(newPath)

	// Submit new path
	m, cmd := m.handleUpdateGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected a command after submitting new path")
	}

	// Simulate dirty repository response
	dirtyMsg := editClonePathDirtyStateMsg{isDirty: true, err: nil}

	// Handle dirty check result (should transition to error and return error msg command)
	model, cmd := m.Update(dirtyMsg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditClonePathError {
		t.Fatalf("expected state %v, got %v", SettingsStateEditClonePathError, m.state)
	}
	if cmd == nil {
		t.Fatalf("expected error message command")
	}

	// Execute the error message command
	errMsg := cmd()

	// Handle the error message to set the error in layout
	model, _ = m.Update(errMsg)
	m = model.(*SettingsModel)
	if m.layout.GetError() == nil {
		t.Fatalf("expected layout to contain an error")
	}

	// Verify path was NOT updated
	if m.currentConfig.Repositories[0].Path != testPath {
		t.Fatalf("expected path to remain unchanged, was %q", m.currentConfig.Repositories[0].Path)
	}
}

// TestIntegration_EditClonePathCancel tests canceling the edit clone path flow
func TestIntegration_EditClonePathCancel(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	newPath := t.TempDir()
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
	m.state = SettingsStateUpdateGitHubPath
	m.textInput.SetValue(newPath)

	// Cancel with Esc
	m, _ = m.handleUpdateGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateRepositoryActions {
		t.Fatalf("expected state %v, got %v", SettingsStateRepositoryActions, m.state)
	}
	if m.currentConfig.Repositories[0].Path != testPath {
		t.Fatalf("expected path to remain %q, got %q", testPath, m.currentConfig.Repositories[0].Path)
	}
}

// TestIntegration_EditClonePathCancelAtConfirmation tests canceling
// at the confirmation step
func TestIntegration_EditClonePathCancelAtConfirmation(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	newPath := t.TempDir()
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
	m.state = SettingsStateUpdateGitHubPath
	m.textInput.SetValue(newPath)

	// Submit and pass dirty check
	m, cmd := m.handleUpdateGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}

	// Simulate clean repository response
	cleanMsg := editClonePathDirtyStateMsg{isDirty: false, err: nil}
	model, _ := m.Update(cleanMsg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditClonePathConfirm {
		t.Fatalf("expected state %v, got %v", SettingsStateEditClonePathConfirm, m.state)
	}

	// Cancel at confirmation
	m, _ = m.handleEditClonePathConfirmKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateRepositoryActions {
		t.Fatalf("expected state %v, got %v", SettingsStateRepositoryActions, m.state)
	}
	if m.currentConfig.Repositories[0].Path != testPath {
		t.Fatalf("expected path to remain %q, got %q", testPath, m.currentConfig.Repositories[0].Path)
	}
}

// TestIntegration_EditClonePathDuplicatePath tests validation error for duplicate paths
func TestIntegration_EditClonePathDuplicatePath(t *testing.T) {
	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	testURL1 := "https://github.com/test/repo1"
	testURL2 := "https://github.com/test/repo2"
	testBranch := "main"

	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "repo-1",
			Name:      "Repo 1",
			Type:      repository.RepositoryTypeGitHub,
			Path:      path1,
			RemoteURL: &testURL1,
			Branch:    &testBranch,
		},
		{
			ID:        "repo-2",
			Name:      "Repo 2",
			Type:      repository.RepositoryTypeGitHub,
			Path:      path2,
			RemoteURL: &testURL2,
			Branch:    &testBranch,
		},
	}

	// Try to change repo-2's path to same as repo-1
	m.selectedRepositoryID = "repo-2"
	m.state = SettingsStateUpdateGitHubPath
	m.textInput.SetValue(path1)

	m, cmd := m.handleUpdateGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected error command")
	}

	// Execute error command
	msg := cmd()
	errMsg, ok := msg.(editClonePathErrorMsg)
	if !ok {
		t.Fatalf("expected editClonePathErrorMsg, got %T", msg)
	}
	if errMsg.err == nil {
		t.Fatalf("expected error in editClonePathErrorMsg")
	}
	if !strings.Contains(errMsg.err.Error(), "already used") {
		t.Fatalf("expected error to mention already used, got %q", errMsg.err.Error())
	}
}

// TestIntegration_EditClonePathDirtyCheckError tests handling of dirty check error
func TestIntegration_EditClonePathDirtyCheckError(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	newPath := t.TempDir()
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
	m.state = SettingsStateUpdateGitHubPath
	m.textInput.SetValue(newPath)

	// Submit path
	m, cmd := m.handleUpdateGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command after submit")
	}

	// Simulate dirty check error
	errorMsg := editClonePathDirtyStateMsg{isDirty: false, err: errors.New("simulated error")}

	// Handle dirty check error (should transition to error and return error msg command)
	model, cmd := m.Update(errorMsg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditClonePathError {
		t.Fatalf("expected state %v, got %v", SettingsStateEditClonePathError, m.state)
	}
	if cmd == nil {
		t.Fatalf("expected error message command")
	}

	// Execute the error message command
	errMsg := cmd()

	// Handle the error message to set the error in layout
	model, _ = m.Update(errMsg)
	m = model.(*SettingsModel)
	if m.layout.GetError() == nil {
		t.Fatalf("expected layout to contain an error")
	}
}

// TestIntegration_EditClonePathMultipleRepositories tests editing clone path
// with multiple repositories (should only affect selected one)
func TestIntegration_EditClonePathMultipleRepositories(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	path3 := t.TempDir()
	newPath2 := t.TempDir()
	url1 := "https://github.com/test/repo1"
	url2 := "https://github.com/test/repo2"
	url3 := "https://github.com/test/repo3"
	branch := "main"

	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{ID: "repo-1", Name: "Repo 1", Type: repository.RepositoryTypeGitHub, Path: path1, RemoteURL: &url1, Branch: &branch},
		{ID: "repo-2", Name: "Repo 2", Type: repository.RepositoryTypeGitHub, Path: path2, RemoteURL: &url2, Branch: &branch},
		{ID: "repo-3", Name: "Repo 3", Type: repository.RepositoryTypeGitHub, Path: path3, RemoteURL: &url3, Branch: &branch},
	}

	// Edit the middle repository
	m.selectedRepositoryID = "repo-2"
	m.state = SettingsStateUpdateGitHubPath
	m.textInput.SetValue(newPath2)

	// Submit and pass dirty check
	m, cmd := m.handleUpdateGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command after submit")
	}

	// Simulate clean repository response
	cleanMsg := editClonePathDirtyStateMsg{isDirty: false, err: nil}
	model, _ := m.Update(cleanMsg)
	m = model.(*SettingsModel)

	// Confirm
	m, cmd = m.handleEditClonePathConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	completeMsg := cmd()
	model, _ = m.Update(completeMsg)
	m = model.(*SettingsModel)

	// Verify only the selected repo was updated
	if m.currentConfig.Repositories[0].Path != path1 {
		t.Fatalf("expected first repo path %q, got %q", path1, m.currentConfig.Repositories[0].Path)
	}
	if m.currentConfig.Repositories[1].Path == path2 {
		t.Fatalf("expected second repo path to change from %q", path2)
	}
	if m.currentConfig.Repositories[1].Path == "" {
		t.Fatalf("expected second repo path to be non-empty")
	}
	if m.currentConfig.Repositories[2].Path != path3 {
		t.Fatalf("expected third repo path %q, got %q", path3, m.currentConfig.Repositories[2].Path)
	}

	_ = configPath
}

// TestIntegration_EditClonePathPreservesOtherFields tests that editing clone path
// doesn't affect other repository fields
func TestIntegration_EditClonePathPreservesOtherFields(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	testPath := t.TempDir()
	newPath := t.TempDir()
	testURL := "https://github.com/test/repo"
	testBranch := "main"
	createdAt := int64(1234567890)

	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "github-repo",
			Name:      "Original Name",
			Type:      repository.RepositoryTypeGitHub,
			Path:      testPath,
			RemoteURL: &testURL,
			Branch:    &testBranch,
			CreatedAt: createdAt,
		},
	}

	m.selectedRepositoryID = "github-repo"
	m.state = SettingsStateUpdateGitHubPath
	m.textInput.SetValue(newPath)

	// Submit and pass dirty check
	m, cmd := m.handleUpdateGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command after submit")
	}

	// Simulate clean repository response
	cleanMsg := editClonePathDirtyStateMsg{isDirty: false, err: nil}
	model, _ := m.Update(cleanMsg)
	m = model.(*SettingsModel)

	// Confirm
	m, cmd = m.handleEditClonePathConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	completeMsg := cmd()
	model, _ = m.Update(completeMsg)
	m = model.(*SettingsModel)

	// Verify all other fields are preserved
	repo := m.currentConfig.Repositories[0]
	if repo.Path == testPath {
		t.Fatalf("expected path to have changed from %q", testPath)
	}
	if repo.ID != "github-repo" {
		t.Fatalf("expected ID %q, got %q", "github-repo", repo.ID)
	}
	if repo.Type != repository.RepositoryTypeGitHub {
		t.Fatalf("expected Type %v, got %v", repository.RepositoryTypeGitHub, repo.Type)
	}
	if repo.RemoteURL == nil || *repo.RemoteURL != testURL {
		t.Fatalf("expected RemoteURL %q, got %v", testURL, repo.RemoteURL)
	}
	if repo.Branch == nil || *repo.Branch != testBranch {
		t.Fatalf("expected Branch %q, got %v", testBranch, repo.Branch)
	}
	if repo.Name != "Original Name" {
		t.Fatalf("expected Name %q, got %q", "Original Name", repo.Name)
	}
	if repo.CreatedAt != createdAt {
		t.Fatalf("expected CreatedAt %d, got %d", createdAt, repo.CreatedAt)
	}

	_ = configPath
}

// TestIntegration_EditClonePathStateIsolation tests that edit clone path flow
// doesn't interfere with other state
func TestIntegration_EditClonePathStateIsolation(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	newPath := t.TempDir()
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

	// Set state from other flows
	m.addRepositoryName = "Some Other Repo"
	m.newGitHubURL = "https://github.com/other/repo"
	m.newGitHubBranch = "develop"

	// Edit clone path
	m.selectedRepositoryID = "github-repo"
	m.state = SettingsStateUpdateGitHubPath
	m.textInput.SetValue(newPath)

	m, _ = m.handleUpdateGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify other flow state is preserved
	// Note: newGitHubPath is reused by edit clone path flow, so we don't check it
	if m.addRepositoryName != "Some Other Repo" {
		t.Fatalf("expected addRepositoryName preserved, got %q", m.addRepositoryName)
	}
	if m.newGitHubURL != "https://github.com/other/repo" {
		t.Fatalf("expected newGitHubURL preserved, got %q", m.newGitHubURL)
	}
	if m.newGitHubBranch != "develop" {
		t.Fatalf("expected newGitHubBranch preserved, got %q", m.newGitHubBranch)
	}
}
