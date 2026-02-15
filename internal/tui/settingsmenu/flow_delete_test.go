// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"rulem/internal/repository"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestIntegration_DeleteLocalRepositoryComplete tests the complete
// delete flow for a local repository (no dirty check)
func TestIntegration_DeleteLocalRepositoryComplete(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	// Set up multiple repositories (need at least 2 to delete one)
	path1 := t.TempDir()
	path2 := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "local-repo-1",
			Name: "Local Repo 1",
			Type: repository.RepositoryTypeLocal,
			Path: path1,
		},
		{
			ID:   "local-repo-2",
			Name: "Local Repo 2",
			Type: repository.RepositoryTypeLocal,
			Path: path2,
		},
	}

	// Step 1: Select repository for deletion
	m.selectedRepositoryID = "local-repo-1"
	m.state = SettingsStateConfirmDelete

	// Step 2: Confirm deletion
	m, cmd := m.handleConfirmDeleteKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("should return delete command")
	}

	// Step 3: Execute delete command
	msg := cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatalf("should return settingsCompleteMsg, got %T", msg)
	}

	// Step 4: Verify repository was deleted
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("should have 1 repository left, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].ID != "local-repo-2" {
		t.Fatalf("wrong repository deleted, expected %q got %q", "local-repo-2", m.currentConfig.Repositories[0].ID)
	}
	if m.state != SettingsStateMainMenu {
		t.Fatalf("expected state %v, got %v", SettingsStateMainMenu, m.state)
	}
	if m.selectedRepositoryID != "" {
		t.Fatalf("expected selectedRepositoryID to be cleared, got %q", m.selectedRepositoryID)
	}

	_ = configPath
}

// TestIntegration_DeleteGitHubRepositoryComplete tests the complete
// delete flow for a GitHub repository (no dirty check in current implementation)
func TestIntegration_DeleteGitHubRepositoryComplete(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	// Set up multiple repositories
	path1 := t.TempDir()
	path2 := t.TempDir()
	url1 := "https://github.com/test/repo1"
	url2 := "https://github.com/test/repo2"
	branch := "main"

	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "github-repo-1",
			Name:      "GitHub Repo 1",
			Type:      repository.RepositoryTypeGitHub,
			Path:      path1,
			RemoteURL: &url1,
			Branch:    &branch,
		},
		{
			ID:        "github-repo-2",
			Name:      "GitHub Repo 2",
			Type:      repository.RepositoryTypeGitHub,
			Path:      path2,
			RemoteURL: &url2,
			Branch:    &branch,
		},
	}

	// Step 1: Select repository for deletion
	m.selectedRepositoryID = "github-repo-1"
	m.state = SettingsStateConfirmDelete

	// Step 2: Confirm deletion
	m, cmd := m.handleConfirmDeleteKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("should return delete command")
	}

	// Step 3: Execute delete command
	msg := cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatalf("should return settingsCompleteMsg, got %T", msg)
	}

	// Step 4: Verify repository was deleted
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("should have 1 repository left, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].ID != "github-repo-2" {
		t.Fatalf("wrong repository deleted, expected %q got %q", "github-repo-2", m.currentConfig.Repositories[0].ID)
	}
	if m.state != SettingsStateMainMenu {
		t.Fatalf("expected state %v, got %v", SettingsStateMainMenu, m.state)
	}

	_ = configPath
}

// TestIntegration_DeleteRepositoryCancel tests canceling the deletion
func TestIntegration_DeleteRepositoryCancel(t *testing.T) {
	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "repo-1",
			Name: "Repo 1",
			Type: repository.RepositoryTypeLocal,
			Path: path1,
		},
		{
			ID:   "repo-2",
			Name: "Repo 2",
			Type: repository.RepositoryTypeLocal,
			Path: path2,
		},
	}

	m.selectedRepositoryID = "repo-1"
	m.state = SettingsStateConfirmDelete

	// Cancel with 'n'
	m, _ = m.handleConfirmDeleteKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if m.state != SettingsStateRepositoryActions {
		t.Fatalf("expected state %v, got %v", SettingsStateRepositoryActions, m.state)
	}
	if len(m.currentConfig.Repositories) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_DeleteRepositoryCancelWithEsc tests canceling deletion with Esc
func TestIntegration_DeleteRepositoryCancelWithEsc(t *testing.T) {
	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "repo-1",
			Name: "Repo 1",
			Type: repository.RepositoryTypeLocal,
			Path: path1,
		},
		{
			ID:   "repo-2",
			Name: "Repo 2",
			Type: repository.RepositoryTypeLocal,
			Path: path2,
		},
	}

	m.selectedRepositoryID = "repo-1"
	m.state = SettingsStateConfirmDelete

	// Cancel with Esc
	m, _ = m.handleConfirmDeleteKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateRepositoryActions {
		t.Fatalf("expected state %v, got %v", SettingsStateRepositoryActions, m.state)
	}
	if len(m.currentConfig.Repositories) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_DeleteLastRepositoryError tests error when trying to delete last repository
func TestIntegration_DeleteLastRepositoryError(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	// Set up only one repository
	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "only-repo",
			Name: "Only Repo",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	m.selectedRepositoryID = "only-repo"
	m.state = SettingsStateConfirmDelete

	// Try to confirm deletion
	m, cmd := m.handleConfirmDeleteKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("should return command")
	}

	// Execute delete command (should return error)
	msg := cmd()
	errMsg, ok := msg.(deleteErrorMsg)
	if !ok {
		t.Fatalf("expected deleteErrorMsg, got %T", msg)
	}
	if errMsg.err == nil {
		t.Fatalf("expected an error in deleteErrorMsg")
	}
	if !strings.Contains(errMsg.err.Error(), "last repository") {
		t.Fatalf("expected error to mention last repository, got %q", errMsg.err.Error())
	}

	// Verify repository was NOT deleted
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("should still have 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].ID != "only-repo" {
		t.Fatalf("expected repo ID %q, got %q", "only-repo", m.currentConfig.Repositories[0].ID)
	}

	_ = configPath
}

// TestIntegration_DeleteRepositoryNotFound tests error when repository ID not found
func TestIntegration_DeleteRepositoryNotFound(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "repo-1",
			Name: "Repo 1",
			Type: repository.RepositoryTypeLocal,
			Path: path1,
		},
		{
			ID:   "repo-2",
			Name: "Repo 2",
			Type: repository.RepositoryTypeLocal,
			Path: path2,
		},
	}

	// Select a non-existent repository
	m.selectedRepositoryID = "non-existent-repo"
	m.state = SettingsStateConfirmDelete

	// Try to delete
	m, cmd := m.handleConfirmDeleteKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected a command for deletion attempt")
	}

	// Execute delete command (should return error)
	msg := cmd()
	errMsg, ok := msg.(deleteErrorMsg)
	if !ok {
		t.Fatalf("expected deleteErrorMsg, got %T", msg)
	}
	if errMsg.err == nil {
		t.Fatalf("expected an error in deleteErrorMsg")
	}
	if !strings.Contains(errMsg.err.Error(), "not found") {
		t.Fatalf("expected error to mention not found, got %q", errMsg.err.Error())
	}

	// Verify no repository was deleted
	if len(m.currentConfig.Repositories) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(m.currentConfig.Repositories))
	}

	_ = configPath
}

// TestIntegration_DeleteMiddleRepository tests deleting a repository from the middle of the list
func TestIntegration_DeleteMiddleRepository(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	path3 := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{ID: "repo-1", Name: "Repo 1", Type: repository.RepositoryTypeLocal, Path: path1},
		{ID: "repo-2", Name: "Repo 2", Type: repository.RepositoryTypeLocal, Path: path2},
		{ID: "repo-3", Name: "Repo 3", Type: repository.RepositoryTypeLocal, Path: path3},
	}

	// Delete the middle repository
	m.selectedRepositoryID = "repo-2"
	m.state = SettingsStateConfirmDelete

	m, cmd := m.handleConfirmDeleteKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected a command from confirm delete")
	}
	msg := cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatalf("expected settingsCompleteMsg, got %T", msg)
	}

	// Verify only the middle repository was deleted
	if len(m.currentConfig.Repositories) != 2 {
		t.Fatalf("expected 2 repositories after deletion, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].ID != "repo-1" {
		t.Fatalf("expected first repo id %q, got %q", "repo-1", m.currentConfig.Repositories[0].ID)
	}
	if m.currentConfig.Repositories[1].ID != "repo-3" {
		t.Fatalf("expected second repo id %q, got %q", "repo-3", m.currentConfig.Repositories[1].ID)
	}

	_ = configPath
}

// TestIntegration_DeleteErrorDismissal tests dismissing a delete error
func TestIntegration_DeleteErrorDismissal(t *testing.T) {
	m := createTestModel(t)

	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "only-repo",
			Name: "Only Repo",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	// Set up error state
	m.state = SettingsStateDeleteError
	m.selectedRepositoryID = "only-repo"

	// Dismiss error with any key
	m, _ = m.handleDeleteErrorKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateRepositoryActions {
		t.Fatalf("expected state %v, got %v", SettingsStateRepositoryActions, m.state)
	}
}

// TestIntegration_DeleteStateIsolation tests that delete flow
// doesn't interfere with other state
func TestIntegration_DeleteStateIsolation(t *testing.T) {
	m := createTestModel(t)

	path1 := t.TempDir()
	path2 := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "repo-1",
			Name: "Repo 1",
			Type: repository.RepositoryTypeLocal,
			Path: path1,
		},
		{
			ID:   "repo-2",
			Name: "Repo 2",
			Type: repository.RepositoryTypeLocal,
			Path: path2,
		},
	}

	// Set state from other flows
	m.addRepositoryName = "Some Other Repo"
	m.newGitHubURL = "https://github.com/other/repo"
	m.newGitHubBranch = "develop"
	m.newGitHubPath = "/some/path"

	// Navigate to delete confirmation
	m.selectedRepositoryID = "repo-1"
	m.state = SettingsStateConfirmDelete

	// Cancel deletion
	m, _ = m.handleConfirmDeleteKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Verify other flow state is preserved
	if m.addRepositoryName != "Some Other Repo" {
		t.Fatalf("expected addRepositoryName preserved, got %q", m.addRepositoryName)
	}
	if m.newGitHubURL != "https://github.com/other/repo" {
		t.Fatalf("expected newGitHubURL preserved, got %q", m.newGitHubURL)
	}
	if m.newGitHubBranch != "develop" {
		t.Fatalf("expected newGitHubBranch preserved, got %q", m.newGitHubBranch)
	}
	if m.newGitHubPath != "/some/path" {
		t.Fatalf("expected newGitHubPath preserved, got %q", m.newGitHubPath)
	}
}
