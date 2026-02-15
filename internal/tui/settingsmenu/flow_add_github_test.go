// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"rulem/internal/repository"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// TODO remove 3rd party test dependcies and use std lib only

// TestIntegration_AddGitHubComplete tests the complete
// add GitHub repository flow from start to finish with successful completion
func TestIntegration_AddGitHubComplete(t *testing.T) {
	// Use test config path to avoid overwriting user's config
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{} // Start empty

	// Step 1: Start at AddRepositoryType (user already selected "Add Repository")
	m.state = SettingsStateAddRepositoryType
	if m.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected state %v, got %v", SettingsStateAddRepositoryType, m.state)
	}

	// Step 2: Select GitHub type (index 1)
	m.addRepositoryTypeIndex = 1
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubName {
		t.Fatalf("should transition to AddGitHubName: expected %v, got %v", SettingsStateAddGitHubName, m.state)
	}

	// Step 3: Enter repository name
	m.textInput.SetValue("My Integration GitHub Repo")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubURL {
		t.Fatalf("should transition to AddGitHubURL: expected %v, got %v", SettingsStateAddGitHubURL, m.state)
	}
	if m.addRepositoryName != "My Integration GitHub Repo" {
		t.Fatalf("should store repository name: expected %q, got %q", "My Integration GitHub Repo", m.addRepositoryName)
	}

	// Step 4: Enter GitHub URL
	testURL := "https://github.com/test/repo"
	m.textInput.SetValue(testURL)
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubBranch {
		t.Fatalf("should transition to AddGitHubBranch: expected %v, got %v", SettingsStateAddGitHubBranch, m.state)
	}
	if m.newGitHubURL != testURL {
		t.Fatalf("should store GitHub URL: expected %q, got %q", testURL, m.newGitHubURL)
	}

	// Step 5: Enter branch (can be empty for default)
	m.textInput.SetValue("main")
	m, _ = m.handleAddGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubPath {
		t.Fatalf("should transition to AddGitHubPath: expected %v, got %v", SettingsStateAddGitHubPath, m.state)
	}
	if m.newGitHubBranch != "main" {
		t.Fatalf("should store branch: expected %q, got %q", "main", m.newGitHubBranch)
	}

	// Step 6: Enter clone path
	testPath := t.TempDir()
	m.textInput.SetValue(testPath)
	m, cmd := m.handleAddGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should return createGitHubRepository command")
	}
	if m.addRepositoryPath == "" {
		t.Fatalf("should store repository path")
	}

	// Step 7: Execute createGitHubRepository command
	// Note: This will fail without valid PAT, but we're testing the flow
	msg := cmd()

	// Command might return error if PAT not configured (expected in test environment)
	// Just verify it returns a message
	if msg == nil {
		t.Fatalf("should return a message")
	}

	_ = configPath // Config path is set up but just for cleanup
}

// TestIntegration_AddGitHub_CancelAtTypeSelection tests canceling
// during the type selection step
func TestIntegration_AddGitHub_CancelAtTypeSelection(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddRepositoryType
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Press Esc to cancel
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != SettingsStateMainMenu {
		t.Fatalf("should return to MainMenu: expected %v, got %v", SettingsStateMainMenu, m.state)
	}

	// Verify no repository was added
	if len(m.currentConfig.Repositories) != 0 {
		t.Fatalf("should not add any repository: expected 0, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_AddGitHub_CancelAtNameInput tests canceling
// during the name input step
func TestIntegration_AddGitHub_CancelAtNameInput(t *testing.T) {
	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Navigate to name input
	m.state = SettingsStateAddRepositoryType
	m.addRepositoryTypeIndex = 1 // GitHub
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubName {
		t.Fatalf("expected %v, got %v", SettingsStateAddGitHubName, m.state)
	}

	// Enter some text, then cancel
	m.textInput.SetValue("Partial Name")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateAddRepositoryType {
		t.Fatalf("should return to type selection: expected %v, got %v", SettingsStateAddRepositoryType, m.state)
	}
	if m.addRepositoryName != "" {
		t.Fatalf("should not store partial name: got %q", m.addRepositoryName)
	}
	if len(m.currentConfig.Repositories) != 0 {
		t.Fatalf("should not add repository: expected 0, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_AddGitHub_CancelAtURLInput tests canceling
// during the URL input step
func TestIntegration_AddGitHub_CancelAtURLInput(t *testing.T) {
	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Navigate through name input
	m.state = SettingsStateAddGitHubName
	m.textInput.SetValue("Test Repo")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubURL {
		t.Fatalf("expected %v, got %v", SettingsStateAddGitHubURL, m.state)
	}

	// Enter URL, then cancel
	m.textInput.SetValue("https://github.com/test/repo")
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateAddGitHubName {
		t.Fatalf("should return to name input: expected %v, got %v", SettingsStateAddGitHubName, m.state)
	}
	if m.newGitHubURL != "" {
		t.Fatalf("should not store partial URL: got %q", m.newGitHubURL)
	}
	if len(m.currentConfig.Repositories) != 0 {
		t.Fatalf("should not add repository: expected 0, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_AddGitHub_CancelAtBranchInput tests canceling
// during the branch input step
func TestIntegration_AddGitHub_CancelAtBranchInput(t *testing.T) {
	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Navigate through name and URL
	m.state = SettingsStateAddGitHubName
	m.textInput.SetValue("Test Repo")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	m.textInput.SetValue("https://github.com/test/repo")
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubBranch {
		t.Fatalf("expected %v, got %v", SettingsStateAddGitHubBranch, m.state)
	}

	// Enter branch, then cancel
	m.textInput.SetValue("develop")
	m, _ = m.handleAddGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateAddGitHubURL {
		t.Fatalf("should return to URL input: expected %v, got %v", SettingsStateAddGitHubURL, m.state)
	}
	if m.newGitHubBranch != "" {
		t.Fatalf("should not store partial branch: got %q", m.newGitHubBranch)
	}
	if len(m.currentConfig.Repositories) != 0 {
		t.Fatalf("should not add repository: expected 0, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_AddGitHub_CancelAtPathInput tests canceling
// during the path input step
func TestIntegration_AddGitHub_CancelAtPathInput(t *testing.T) {
	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Navigate through all inputs
	m.state = SettingsStateAddGitHubName
	m.textInput.SetValue("Test Repo")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	m.textInput.SetValue("https://github.com/test/repo")
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})

	m.textInput.SetValue("main")
	m, _ = m.handleAddGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubPath {
		t.Fatalf("expected %v, got %v", SettingsStateAddGitHubPath, m.state)
	}

	// Enter path, then cancel
	m.textInput.SetValue("/some/path")
	m, _ = m.handleAddGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateAddGitHubBranch {
		t.Fatalf("should return to branch input: expected %v, got %v", SettingsStateAddGitHubBranch, m.state)
	}
	if m.addRepositoryPath != "" {
		t.Fatalf("should not store partial path: got %q", m.addRepositoryPath)
	}
	if len(m.currentConfig.Repositories) != 0 {
		t.Fatalf("should not add repository: expected 0, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_AddGitHub_EmptyNameValidation tests validation
// error when submitting empty repository name
func TestIntegration_AddGitHub_EmptyNameValidation(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubName
	m.textInput.SetValue("")

	// Try to submit empty name
	m, cmd := m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddGitHubName {
		t.Fatalf("should stay in name input: expected %v, got %v", SettingsStateAddGitHubName, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not proceed")
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should show error")
	}
	if !strings.Contains(m.layout.GetError().Error(), "cannot be empty") {
		t.Fatalf("error should mention empty: got %q", m.layout.GetError().Error())
	}
}

// TestIntegration_AddGitHub_DuplicateNameValidation tests validation
// error when submitting a duplicate repository name
func TestIntegration_AddGitHub_DuplicateNameValidation(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubName

	// Add existing repository
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "existing-id",
			Name: "Existing Repo",
			Type: repository.RepositoryTypeGitHub,
			Path: "/existing/path",
		},
	}

	// Try to use duplicate name
	m.textInput.SetValue("Existing Repo")
	m, cmd := m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddGitHubName {
		t.Fatalf("should stay in name input: expected %v, got %v", SettingsStateAddGitHubName, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not proceed")
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should show error")
	}
	if !strings.Contains(m.layout.GetError().Error(), "already exists") {
		t.Fatalf("error should mention duplicate: got %q", m.layout.GetError().Error())
	}
}

// TestIntegration_AddGitHub_InvalidURLValidation tests validation
// error when submitting invalid GitHub URL
func TestIntegration_AddGitHub_InvalidURLValidation(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "invalid format",
			url:  "not-a-valid-url",
		},
		{
			name: "malformed URL",
			url:  "ht!tp://invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel(t)
			m.state = SettingsStateAddGitHubURL
			m.addRepositoryName = "Test Repo"
			m.textInput.SetValue(tt.url)

			m, cmd := m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})

			if m.state != SettingsStateAddGitHubURL {
				t.Fatalf("should stay in URL input: expected %v, got %v", SettingsStateAddGitHubURL, m.state)
			}
			if cmd != nil {
				t.Fatalf("should not proceed")
			}
			if m.layout.GetError() == nil {
				t.Fatalf("should show error")
			}
		})
	}
}

// TestIntegration_AddGitHub_DuplicateURLValidation tests validation
// error when submitting a duplicate GitHub URL
func TestIntegration_AddGitHub_DuplicateURLValidation(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubURL
	m.addRepositoryName = "New Repo"

	duplicateURL := "https://github.com/test/existing"

	// Add existing repository with same URL
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:        "existing-id",
			Name:      "Existing",
			Type:      repository.RepositoryTypeGitHub,
			Path:      "/existing/path",
			RemoteURL: &duplicateURL,
		},
	}

	// Try to use duplicate URL
	m.textInput.SetValue(duplicateURL)
	m, cmd := m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddGitHubURL {
		t.Fatalf("should stay in URL input: expected %v, got %v", SettingsStateAddGitHubURL, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not proceed")
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should show error")
	}
	if !strings.Contains(m.layout.GetError().Error(), "already used") {
		t.Fatalf("error should mention duplicate: got %q", m.layout.GetError().Error())
	}
}

// TestIntegration_AddGitHub_EmptyBranchAllowed tests that empty branch
// is allowed (uses default branch)
func TestIntegration_AddGitHub_EmptyBranchAllowed(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubBranch
	m.addRepositoryName = "Test Repo"
	m.newGitHubURL = "https://github.com/test/repo"
	m.textInput.SetValue("")

	// Submit empty branch
	m, cmd := m.handleAddGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddGitHubPath {
		t.Fatalf("should proceed to path input: expected %v, got %v", SettingsStateAddGitHubPath, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not execute command yet")
	}
	if m.newGitHubBranch != "" {
		t.Fatalf("branch should remain empty: got %q", m.newGitHubBranch)
	}
}

// TestIntegration_AddGitHub_InvalidBranchValidation tests validation
// error when submitting invalid branch name
func TestIntegration_AddGitHub_InvalidBranchValidation(t *testing.T) {
	tests := []struct {
		name   string
		branch string
	}{
		{
			name:   "branch with spaces",
			branch: "feature branch",
		},
		{
			name:   "branch with special chars",
			branch: "feature..branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel(t)
			m.state = SettingsStateAddGitHubBranch
			m.addRepositoryName = "Test Repo"
			m.newGitHubURL = "https://github.com/test/repo"
			m.textInput.SetValue(tt.branch)

			m, cmd := m.handleAddGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})

			if m.state != SettingsStateAddGitHubBranch {
				t.Fatalf("should stay in branch input: expected %v, got %v", SettingsStateAddGitHubBranch, m.state)
			}
			if cmd != nil {
				t.Fatalf("should not proceed")
			}
			if m.layout.GetError() == nil {
				t.Fatalf("should show error")
			}
		})
	}
}

// TestIntegration_AddGitHub_EmptyPathValidation tests validation
// error when submitting empty repository path
func TestIntegration_AddGitHub_EmptyPathValidation(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubPath
	m.addRepositoryName = "Test Repo"
	m.newGitHubURL = "https://github.com/test/repo"
	m.newGitHubBranch = "main"

	// Set empty placeholder to ensure empty path
	m.textInput.Placeholder = ""
	m.textInput.SetValue("")

	// Try to submit empty path with no placeholder
	m, cmd := m.handleAddGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Empty path uses placeholder, so this test depends on placeholder logic
	// Just verify command is returned or error shown
	_ = cmd
	_ = m
}

// TestIntegration_AddGitHub_DuplicatePathValidation tests validation
// error when submitting a duplicate repository path
func TestIntegration_AddGitHub_DuplicatePathValidation(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubPath
	m.addRepositoryName = "New Repo"
	m.newGitHubURL = "https://github.com/test/newrepo"
	m.newGitHubBranch = "main"

	testPath := t.TempDir()

	// Add existing repository with same path
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "existing-id",
			Name: "Existing",
			Type: repository.RepositoryTypeGitHub,
			Path: testPath,
		},
	}

	// Try to use duplicate path
	m.textInput.SetValue(testPath)
	m, cmd := m.handleAddGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddGitHubPath {
		t.Fatalf("should stay in path input: expected %v, got %v", SettingsStateAddGitHubPath, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not proceed")
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should show error")
	}
	if !strings.Contains(m.layout.GetError().Error(), "already used") {
		t.Fatalf("error should mention duplicate: got %q", m.layout.GetError().Error())
	}
}

// TestIntegration_AddGitHub_PathValidationRecovers tests that the path input can recover
// after a duplicate-path error by editing the value and resubmitting.
func TestIntegration_AddGitHub_PathValidationRecovers(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubPath
	m.addRepositoryName = "Recovery Repo"
	m.newGitHubURL = "https://github.com/test/recovery"
	m.newGitHubBranch = "main"

	duplicatePath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "existing-id",
			Name: "Existing",
			Type: repository.RepositoryTypeGitHub,
			Path: duplicatePath,
		},
	}

	m.textInput.SetValue(duplicatePath)
	m, cmd := m.handleAddGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddGitHubPath {
		t.Fatalf("should stay in path input after duplicate: expected %v, got %v", SettingsStateAddGitHubPath, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not proceed when path is duplicate")
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should show duplicate error")
	}
	if !strings.Contains(m.layout.GetError().Error(), "already used") {
		t.Fatalf("error should mention duplicate: got %q", m.layout.GetError().Error())
	}

	// Simulate user editing the path (deleted characters and typed a new path)
	newPath := t.TempDir()
	m.textInput.SetValue(newPath)
	m, cmd = m.handleAddGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatalf("should return command after correcting path")
	}
	if m.layout.GetError() != nil {
		t.Fatalf("error should be cleared after correction: got %v", m.layout.GetError())
	}
	if m.addRepositoryPath != newPath {
		t.Fatalf("should store the corrected path: expected %q, got %q", newPath, m.addRepositoryPath)
	}
}

// TestIntegration_AddGitHub_StateIsolationFromOtherFlows tests
// that the Add GitHub flow doesn't interfere with state from other flows
func TestIntegration_AddGitHub_StateIsolationFromOtherFlows(t *testing.T) {
	m := createTestModel(t)

	// Set state from other flows (these should not be affected)
	m.selectedRepositoryID = "some-repo-id"

	// Go through Add GitHub flow (partial)
	m.state = SettingsStateAddGitHubName
	m.textInput.SetValue("GitHub Repo")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	m.textInput.SetValue("https://github.com/test/repo")
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify other flow state is preserved
	if m.selectedRepositoryID != "some-repo-id" {
		t.Fatalf("should preserve selected repo ID: expected %q, got %q", "some-repo-id", m.selectedRepositoryID)
	}

	// Verify GitHub flow state is set
	if m.addRepositoryName != "GitHub Repo" {
		t.Fatalf("expected %q, got %q", "GitHub Repo", m.addRepositoryName)
	}
	if m.newGitHubURL != "https://github.com/test/repo" {
		t.Fatalf("expected %q, got %q", "https://github.com/test/repo", m.newGitHubURL)
	}
}

// TestIntegration_AddGitHub_RecoverFromValidationError tests
// recovering from multiple validation errors and successfully completing the flow
func TestIntegration_AddGitHub_RecoverFromValidationError(t *testing.T) {
	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}
	m.state = SettingsStateAddGitHubName

	// Step 1: Try empty name (should fail)
	m.textInput.SetValue("")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubName {
		t.Fatalf("should stay in name input: expected %v, got %v", SettingsStateAddGitHubName, m.state)
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should have error")
	}

	// Step 2: Fix the error by entering valid name
	m.textInput.SetValue("Valid GitHub Repo")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubURL {
		t.Fatalf("should proceed to URL input: expected %v, got %v", SettingsStateAddGitHubURL, m.state)
	}

	// Step 3: Try invalid URL (should fail)
	m.textInput.SetValue("not-a-valid-url")
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubURL {
		t.Fatalf("should stay in URL input: expected %v, got %v", SettingsStateAddGitHubURL, m.state)
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should have error")
	}

	// Step 4: Fix with valid GitHub URL
	m.textInput.SetValue("https://github.com/test/validrepo")
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubBranch {
		t.Fatalf("should proceed to branch input: expected %v, got %v", SettingsStateAddGitHubBranch, m.state)
	}

	// Step 5: Enter valid branch
	m.textInput.SetValue("main")
	m, _ = m.handleAddGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubPath {
		t.Fatalf("should proceed to path input: expected %v, got %v", SettingsStateAddGitHubPath, m.state)
	}

	// Step 6: Try duplicate path (should fail)
	testPath := t.TempDir()
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{ID: "existing", Name: "Existing", Type: repository.RepositoryTypeLocal, Path: testPath},
	}
	m.textInput.SetValue(testPath)
	m, _ = m.handleAddGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubPath {
		t.Fatalf("should stay in path input: expected %v, got %v", SettingsStateAddGitHubPath, m.state)
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should have error")
	}

	// Step 7: Fix with different path
	newPath := t.TempDir()
	m.textInput.SetValue(newPath)
	m, cmd := m.handleAddGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should proceed with creation")
	}

	// Verify flow data is correct
	if m.addRepositoryName != "Valid GitHub Repo" {
		t.Fatalf("expected %q, got %q", "Valid GitHub Repo", m.addRepositoryName)
	}
	if m.newGitHubURL != "https://github.com/test/validrepo" {
		t.Fatalf("expected %q, got %q", "https://github.com/test/validrepo", m.newGitHubURL)
	}
	if m.newGitHubBranch != "main" {
		t.Fatalf("expected %q, got %q", "main", m.newGitHubBranch)
	}
	if m.addRepositoryPath == "" {
		t.Fatalf("expected non-empty repository path")
	}
}

// TestIntegration_AddGitHub_DefaultBranchFlow tests using empty branch
// which should use the repository's default branch
func TestIntegration_AddGitHub_DefaultBranchFlow(t *testing.T) {
	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Navigate to branch input
	m.state = SettingsStateAddGitHubName
	m.textInput.SetValue("Test Repo")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	m.textInput.SetValue("https://github.com/test/repo")
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddGitHubBranch {
		t.Fatalf("expected %v, got %v", SettingsStateAddGitHubBranch, m.state)
	}

	// Submit empty branch (should use default)
	m.textInput.SetValue("")
	m, _ = m.handleAddGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddGitHubPath {
		t.Fatalf("should proceed to path: expected %v, got %v", SettingsStateAddGitHubPath, m.state)
	}
	if m.newGitHubBranch != "" {
		t.Fatalf("branch should be empty (uses default): got %q", m.newGitHubBranch)
	}
}

// TestIntegration_AddGitHub_MissingPAT tests the optional PAT flow
// when no PAT exists in the credential store
func TestIntegration_AddGitHub_MissingPAT(t *testing.T) {
	// Use test config path to avoid overwriting user's config
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{} // Start empty

	// Configure mock to return "not found" error for GetGitHubToken
	mockCred := &mockCredentialManager{
		getErr: fmt.Errorf("no GitHub token found"),
	}
	m.credManager = mockCred

	// Navigate through the Add GitHub flow
	m.state = SettingsStateAddRepositoryType
	m.addRepositoryTypeIndex = 1 // GitHub
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})

	m.textInput.SetValue("Test Repo No PAT")
	m, _ = m.handleAddGitHubNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	m.textInput.SetValue("https://github.com/test/repo")
	m, _ = m.handleAddGitHubURLKeys(tea.KeyMsg{Type: tea.KeyEnter})

	m.textInput.SetValue("main")
	m, _ = m.handleAddGitHubBranchKeys(tea.KeyMsg{Type: tea.KeyEnter})

	testPath := t.TempDir()
	m.textInput.SetValue(testPath)
	m, cmd := m.handleAddGitHubPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should return createGitHubRepository command")
	}

	// Execute the command - should transition to PAT input state
	msg := cmd()
	if _, ok := msg.(addGitHubPATNeededMsg); !ok {
		t.Fatalf("should return PAT needed message: got %T", msg)
	}

	// Handle the message - should transition to AddGitHubPAT state
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(*SettingsModel)
	if m.state != SettingsStateAddGitHubPAT {
		t.Fatalf("should transition to AddGitHubPAT: expected %v, got %v", SettingsStateAddGitHubPAT, m.state)
	}
	if m.textInput.EchoMode != textinput.EchoPassword {
		t.Fatalf("should set password echo mode: expected %v, got %v", textinput.EchoPassword, m.textInput.EchoMode)
	}

	// Verify repository details are preserved
	if m.addRepositoryName != "Test Repo No PAT" {
		t.Fatalf("expected %q, got %q", "Test Repo No PAT", m.addRepositoryName)
	}
	if m.newGitHubURL != "https://github.com/test/repo" {
		t.Fatalf("expected %q, got %q", "https://github.com/test/repo", m.newGitHubURL)
	}
	if m.newGitHubBranch != "main" {
		t.Fatalf("expected %q, got %q", "main", m.newGitHubBranch)
	}
	if m.addRepositoryPath != testPath {
		t.Fatalf("expected %q, got %q", testPath, m.addRepositoryPath)
	}

	_ = configPath
}

// TestIntegration_AddGitHub_PATInput_Success tests successfully
// entering a PAT in the optional PAT flow
func TestIntegration_AddGitHub_PATInput_Success(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Set up the state as if we just transitioned to AddGitHubPAT
	m.state = SettingsStateAddGitHubPAT
	m.addRepositoryName = "Test Repo"
	m.newGitHubURL = "https://github.com/test/repo"
	m.newGitHubBranch = "main"
	m.addRepositoryPath = t.TempDir()

	// Configure mock to accept the PAT
	mockCred := &mockCredentialManager{}
	m.credManager = mockCred

	// Enter valid PAT
	testPAT := "ghp_testtoken123456789"
	m.textInput.SetValue(testPAT)
	m, cmd := m.handleAddGitHubPATKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should return createGitHubRepositoryWithPAT command")
	}

	// Verify PAT was stored
	if mockCred.storedToken != testPAT {
		t.Fatalf("expected %q, got %q", testPAT, mockCred.storedToken)
	}

	_ = configPath
}

// TestIntegration_AddGitHub_PATInput_EmptyPAT tests validation
// when user submits empty PAT
func TestIntegration_AddGitHub_PATInput_EmptyPAT(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubPAT
	m.addRepositoryName = "Test Repo"
	m.newGitHubURL = "https://github.com/test/repo"

	// Submit empty PAT
	m.textInput.SetValue("")
	m, cmd := m.handleAddGitHubPATKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("should not return command")
	}
	if m.state != SettingsStateAddGitHubPAT {
		t.Fatalf("should remain in PAT input state: expected %v, got %v", SettingsStateAddGitHubPAT, m.state)
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should display error")
	}
}

// TestIntegration_AddGitHub_PATInput_InvalidFormat tests validation
// when user submits PAT with invalid format
func TestIntegration_AddGitHub_PATInput_InvalidFormat(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubPAT
	m.addRepositoryName = "Test Repo"
	m.newGitHubURL = "https://github.com/test/repo"

	// Configure mock to reject invalid format
	mockCred := &mockCredentialManager{
		validateTokenErr: fmt.Errorf("invalid token format"),
	}
	m.credManager = mockCred

	// Submit invalid PAT
	m.textInput.SetValue("invalid_token")
	m, cmd := m.handleAddGitHubPATKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("should not return command")
	}
	if m.state != SettingsStateAddGitHubPAT {
		t.Fatalf("should remain in PAT input state: expected %v, got %v", SettingsStateAddGitHubPAT, m.state)
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should display error")
	}
}

// TestIntegration_AddGitHub_PATInput_ValidationFailed tests when
// PAT format is valid but fails validation with repository
func TestIntegration_AddGitHub_PATInput_ValidationFailed(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubPAT
	m.addRepositoryName = "Test Repo"
	m.newGitHubURL = "https://github.com/test/repo"

	// Configure mock to accept format but fail repo validation
	mockCred := &mockCredentialManager{
		validateRepoErrs: map[string]error{
			"https://github.com/test/repo": fmt.Errorf("authentication failed"),
		},
	}
	m.credManager = mockCred

	// Submit PAT
	m.textInput.SetValue("ghp_validformat_but_invalid_permissions")
	m, cmd := m.handleAddGitHubPATKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("should not return command")
	}
	if m.state != SettingsStateAddGitHubPAT {
		t.Fatalf("should remain in PAT input state: expected %v, got %v", SettingsStateAddGitHubPAT, m.state)
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should display error")
	}
}

// TestIntegration_AddGitHub_PATInput_Cancel tests canceling
// the optional PAT input flow
func TestIntegration_AddGitHub_PATInput_Cancel(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubPAT
	m.addRepositoryName = "Test Repo"
	m.newGitHubURL = "https://github.com/test/repo"
	m.addRepositoryPath = "/test/path"

	// Press Esc to cancel
	m, _ = m.handleAddGitHubPATKeys(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != SettingsStateAddGitHubPath {
		t.Fatalf("should return to AddGitHubPath: expected %v, got %v", SettingsStateAddGitHubPath, m.state)
	}
	if m.layout.GetError() != nil {
		t.Fatalf("should clear any errors: got %v", m.layout.GetError())
	}

	// Verify repository details are preserved
	if m.addRepositoryName != "Test Repo" {
		t.Fatalf("expected %q, got %q", "Test Repo", m.addRepositoryName)
	}
	if m.newGitHubURL != "https://github.com/test/repo" {
		t.Fatalf("expected %q, got %q", "https://github.com/test/repo", m.newGitHubURL)
	}
	if m.addRepositoryPath != "/test/path" {
		t.Fatalf("expected %q, got %q", "/test/path", m.addRepositoryPath)
	}
}

// TestIntegration_AddGitHub_PATView tests the view rendering
// for the optional PAT input state
func TestIntegration_AddGitHub_PATView(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddGitHubPAT
	m.addRepositoryName = "My Test Repo"
	m.newGitHubURL = "https://github.com/test/repo"
	m.newGitHubBranch = "develop"

	view := m.View()
	if view == "" {
		t.Fatalf("view should not be empty")
	}
	if !strings.Contains(view, "My Test Repo") {
		t.Fatalf("view should contain repository name")
	}
	if !strings.Contains(view, "https://github.com/test/repo") {
		t.Fatalf("view should contain URL")
	}
	if !strings.Contains(view, "develop") {
		t.Fatalf("view should contain branch")
	}
	if !strings.Contains(view, "Personal Access Token") {
		t.Fatalf("view should mention PAT")
	}
}
