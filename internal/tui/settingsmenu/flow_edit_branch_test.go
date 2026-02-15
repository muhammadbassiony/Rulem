// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"errors"
	"testing"

	"rulem/internal/repository"

	tea "github.com/charmbracelet/bubbletea"
)

// TestIntegration_EditBranchComplete tests the complete
// edit branch flow from start to finish with successful completion (clean repo)
func TestIntegration_EditBranchComplete(t *testing.T) {
	t.Skip("Requires real git repository - updateGitHubBranch validates branch on remote")
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	// Set up existing GitHub repository
	testPath := t.TempDir()
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

	// Step 1: Select repository and navigate to edit branch
	m.selectedRepositoryID = "github-repo-id"
	m.state = SettingsStateUpdateGitHubBranch
	m.textInput.SetValue("develop")

	// Step 2: Submit new branch (triggers dirty check)
	m, cmd := m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should return dirty check command")
	}

	// Step 3: Simulate clean dirty check response
	cleanMsg := editBranchDirtyStateMsg{isDirty: false, err: nil}

	// Step 4: Handle dirty check result (should transition to confirm)
	model, _ := m.Update(cleanMsg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditBranchConfirm {
		t.Fatalf("expected state %v, got %v", SettingsStateEditBranchConfirm, m.state)
	}

	// Step 5: Confirm the change
	m, cmd = m.handleEditBranchConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
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

	// Step 8: Verify branch was updated
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].Branch == nil || *m.currentConfig.Repositories[0].Branch != "develop" {
		var got string
		if m.currentConfig.Repositories[0].Branch != nil {
			got = *m.currentConfig.Repositories[0].Branch
		}
		t.Fatalf("expected branch %q, got %q", "develop", got)
	}
	if m.currentConfig.Repositories[0].ID != "github-repo-id" {
		t.Fatalf("expected repo ID %q, got %q", "github-repo-id", m.currentConfig.Repositories[0].ID)
	}

	_ = configPath
}

// TestIntegration_EditBranchDirtyRepository tests editing branch
// when repository has uncommitted changes
func TestIntegration_EditBranchDirtyRepository(t *testing.T) {
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
	m.state = SettingsStateUpdateGitHubBranch
	m.textInput.SetValue("feature-branch")

	// Submit new branch
	m, cmd := m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected a command after submitting new branch")
	}

	// Simulate dirty repository response
	dirtyMsg := editBranchDirtyStateMsg{isDirty: true, err: nil}

	// Handle dirty check result (should transition to error and return error msg command)
	model, cmd := m.Update(dirtyMsg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditBranchError {
		t.Fatalf("expected state %v, got %v", SettingsStateEditBranchError, m.state)
	}
	if cmd == nil {
		t.Fatalf("expected an error message command")
	}

	// Execute the error message command
	errMsg := cmd()

	// Handle the error message to set the error in layout
	model, _ = m.Update(errMsg)
	m = model.(*SettingsModel)
	if m.layout.GetError() == nil {
		t.Fatalf("expected layout to contain an error")
	}

	// Verify branch was NOT updated
	if m.currentConfig.Repositories[0].Branch == nil || *m.currentConfig.Repositories[0].Branch != "main" {
		var got string
		if m.currentConfig.Repositories[0].Branch != nil {
			got = *m.currentConfig.Repositories[0].Branch
		}
		t.Fatalf("expected branch to remain %q, got %q", "main", got)
	}
}

// TestIntegration_EditBranchCancel tests canceling the edit branch flow
func TestIntegration_EditBranchCancel(t *testing.T) {
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
	m.state = SettingsStateUpdateGitHubBranch
	m.textInput.SetValue("feature-branch")

	// Cancel with Esc
	m, _ = m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateRepositoryActions {
		t.Fatalf("expected state %v, got %v", SettingsStateRepositoryActions, m.state)
	}
	if m.currentConfig.Repositories[0].Branch == nil || *m.currentConfig.Repositories[0].Branch != "main" {
		t.Fatalf("expected branch to remain %q", "main")
	}
}

// TestIntegration_EditBranchCancelAtConfirmation tests canceling
// at the confirmation step
func TestIntegration_EditBranchCancelAtConfirmation(t *testing.T) {
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
	m.state = SettingsStateUpdateGitHubBranch
	m.textInput.SetValue("develop")

	// Submit and pass dirty check
	m, cmd := m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected a command after submit")
	}

	// Simulate clean repository response
	cleanMsg := editBranchDirtyStateMsg{isDirty: false, err: nil}
	model, _ := m.Update(cleanMsg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditBranchConfirm {
		t.Fatalf("expected state %v, got %v", SettingsStateEditBranchConfirm, m.state)
	}

	// Cancel at confirmation
	m, _ = m.handleEditBranchConfirmKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateRepositoryActions {
		t.Fatalf("expected state %v, got %v", SettingsStateRepositoryActions, m.state)
	}
	if m.currentConfig.Repositories[0].Branch == nil || *m.currentConfig.Repositories[0].Branch != "main" {
		t.Fatalf("expected branch to remain %q", "main")
	}
}

// TestIntegration_EditBranchEmptyValue tests setting branch to empty (default)
func TestIntegration_EditBranchEmptyValue(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	testPath := t.TempDir()
	testURL := "https://github.com/test/repo"
	testBranch := "develop"
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
	m.state = SettingsStateUpdateGitHubBranch
	m.textInput.SetValue("") // Empty = use default branch

	// Submit empty branch
	m, cmd := m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected a command after submit")
	}

	// Simulate clean repository response
	cleanMsg := editBranchDirtyStateMsg{isDirty: false, err: nil}
	model, _ := m.Update(cleanMsg)
	m = model.(*SettingsModel)

	// Confirm
	m, cmd = m.handleEditBranchConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command on confirm")
	}
	completeMsg := cmd()
	model, _ = m.Update(completeMsg)
	m = model.(*SettingsModel)

	// Verify branch is nil (default)
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].Branch != nil {
		t.Fatalf("expected branch to be nil for default, got %v", *m.currentConfig.Repositories[0].Branch)
	}

	_ = configPath
}

// TestIntegration_EditBranchInvalidName tests validation error for invalid branch names
func TestIntegration_EditBranchInvalidName(t *testing.T) {
	testCases := []struct {
		name       string
		branchName string
	}{
		{"starts with slash", "/invalid"},
		{"ends with slash", "invalid/"},
		{"contains double dot", "invalid..branch"},
		{"contains tilde", "invalid~branch"},
		{"contains caret", "invalid^branch"},
		{"contains colon", "invalid:branch"},
		{"contains space", "invalid branch"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
			m.state = SettingsStateUpdateGitHubBranch
			m.textInput.SetValue(tc.branchName)

			// Submit invalid branch
			m, cmd := m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
			if cmd == nil {
				t.Fatalf("expected error command for invalid branch")
			}

			// Execute error command
			msg := cmd()
			if _, ok := msg.(editBranchErrorMsg); !ok {
				t.Fatalf("expected editBranchErrorMsg, got %T", msg)
			}

			// State should not change yet (error command not handled)
			if m.state != SettingsStateUpdateGitHubBranch {
				t.Fatalf("expected state %v, got %v", SettingsStateUpdateGitHubBranch, m.state)
			}
		})
	}
}

// TestIntegration_EditBranchDirtyCheckError tests handling of dirty check error
func TestIntegration_EditBranchDirtyCheckError(t *testing.T) {
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
	m.state = SettingsStateUpdateGitHubBranch
	m.textInput.SetValue("develop")

	// Submit branch
	m, cmd := m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected a command after submit")
	}

	// Simulate dirty check error
	errorMsg := editBranchDirtyStateMsg{isDirty: false, err: errors.New("simulated error")}

	// Handle dirty check error (should transition to error and return error msg command)
	model, cmd := m.Update(errorMsg)
	m = model.(*SettingsModel)
	if m.state != SettingsStateEditBranchError {
		t.Fatalf("expected state %v, got %v", SettingsStateEditBranchError, m.state)
	}
	if cmd == nil {
		t.Fatalf("expected an error message command")
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

// TestIntegration_EditBranchMultipleRepositories tests editing branch
// with multiple repositories (should only affect selected one)
func TestIntegration_EditBranchMultipleRepositories(t *testing.T) {
	t.Skip("Requires real git repository - updateGitHubBranch validates branch on remote")
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	path3 := t.TempDir()
	url1 := "https://github.com/test/repo1"
	url2 := "https://github.com/test/repo2"
	url3 := "https://github.com/test/repo3"
	branch1 := "main"
	branch2 := "develop"
	branch3 := "feature"

	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{ID: "repo-1", Name: "Repo 1", Type: repository.RepositoryTypeGitHub, Path: path1, RemoteURL: &url1, Branch: &branch1},
		{ID: "repo-2", Name: "Repo 2", Type: repository.RepositoryTypeGitHub, Path: path2, RemoteURL: &url2, Branch: &branch2},
		{ID: "repo-3", Name: "Repo 3", Type: repository.RepositoryTypeGitHub, Path: path3, RemoteURL: &url3, Branch: &branch3},
	}

	// Edit the middle repository
	m.selectedRepositoryID = "repo-2"
	m.state = SettingsStateUpdateGitHubBranch
	m.textInput.SetValue("release")

	// Submit and pass dirty check
	m, cmd := m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected a command after submit")
	}

	// Simulate clean repository response
	cleanMsg := editBranchDirtyStateMsg{isDirty: false, err: nil}
	model, _ := m.Update(cleanMsg)
	m = model.(*SettingsModel)

	// Confirm
	m, cmd = m.handleEditBranchConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	completeMsg := cmd()
	model, _ = m.Update(completeMsg)
	m = model.(*SettingsModel)

	// Verify only the selected repo was updated
	if m.currentConfig.Repositories[0].Branch == nil || *m.currentConfig.Repositories[0].Branch != "main" {
		t.Fatalf("expected first branch %q", "main")
	}
	if m.currentConfig.Repositories[1].Branch == nil || *m.currentConfig.Repositories[1].Branch != "release" {
		t.Fatalf("expected second branch %q", "release")
	}
	if m.currentConfig.Repositories[2].Branch == nil || *m.currentConfig.Repositories[2].Branch != "feature" {
		t.Fatalf("expected third branch %q", "feature")
	}

	_ = configPath
}

// TestIntegration_EditBranchPreservesOtherFields tests that editing
// branch doesn't affect other repository fields
func TestIntegration_EditBranchPreservesOtherFields(t *testing.T) {
	t.Skip("Requires real git repository - updateGitHubBranch validates branch on remote")
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	testPath := t.TempDir()
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
	m.state = SettingsStateUpdateGitHubBranch
	m.textInput.SetValue("new-branch")

	// Submit and pass dirty check
	m, cmd := m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected a command after submit")
	}

	// Simulate clean repository response
	cleanMsg := editBranchDirtyStateMsg{isDirty: false, err: nil}
	model, _ := m.Update(cleanMsg)
	m = model.(*SettingsModel)

	// Confirm
	m, cmd = m.handleEditBranchConfirmKeys(tea.KeyMsg{Type: tea.KeyEnter})
	completeMsg := cmd()
	model, _ = m.Update(completeMsg)
	m = model.(*SettingsModel)

	// Verify all other fields are preserved
	repo := m.currentConfig.Repositories[0]
	if repo.Branch == nil || *repo.Branch != "new-branch" {
		t.Fatalf("expected branch %q, got %v", "new-branch", repo.Branch)
	}
	if repo.ID != "github-repo" {
		t.Fatalf("expected ID %q, got %q", "github-repo", repo.ID)
	}
	if repo.Type != repository.RepositoryTypeGitHub {
		t.Fatalf("expected Type %v, got %v", repository.RepositoryTypeGitHub, repo.Type)
	}
	if repo.Path != testPath {
		t.Fatalf("expected Path %q, got %q", testPath, repo.Path)
	}
	if repo.RemoteURL == nil || *repo.RemoteURL != testURL {
		t.Fatalf("expected RemoteURL %q, got %v", testURL, repo.RemoteURL)
	}
	if repo.Name != "Original Name" {
		t.Fatalf("expected Name %q, got %q", "Original Name", repo.Name)
	}
	if repo.CreatedAt != createdAt {
		t.Fatalf("expected CreatedAt %d, got %d", createdAt, repo.CreatedAt)
	}

	_ = configPath
}

// TestIntegration_EditBranchStateIsolation tests that edit branch flow
// doesn't interfere with other state
func TestIntegration_EditBranchStateIsolation(t *testing.T) {
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

	// Set state from other flows
	m.addRepositoryName = "Some Other Repo"
	m.newGitHubURL = "https://github.com/other/repo"
	m.newGitHubPath = "/some/path"

	// Edit branch
	m.selectedRepositoryID = "github-repo"
	m.state = SettingsStateUpdateGitHubBranch
	m.textInput.SetValue("feature")

	m, _ = m.handleUpdateGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify other flow state is preserved
	// Note: newGitHubBranch is reused by edit branch flow, so we don't check it
	if m.addRepositoryName != "Some Other Repo" {
		t.Fatalf("expected addRepositoryName preserved, got %q", m.addRepositoryName)
	}
	if m.newGitHubURL != "https://github.com/other/repo" {
		t.Fatalf("expected newGitHubURL preserved, got %q", m.newGitHubURL)
	}
	if m.newGitHubPath != "/some/path" {
		t.Fatalf("expected newGitHubPath preserved, got %q", m.newGitHubPath)
	}
}
