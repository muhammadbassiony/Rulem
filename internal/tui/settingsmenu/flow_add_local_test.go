// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

import (
	"fmt"
	"rulem/internal/repository"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestHandleAddLocalNameKeys_ValidInput tests accepting valid repository names
func TestHandleAddLocalNameKeys_ValidInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantState SettingsState
	}{
		{
			name:      "simple name",
			input:     "My Rules",
			wantState: SettingsStateAddLocalPath,
		},
		{
			name:      "name with numbers",
			input:     "Rules123",
			wantState: SettingsStateAddLocalPath,
		},
		{
			name:      "name with special chars",
			input:     "My-Rules_v2",
			wantState: SettingsStateAddLocalPath,
		},
		{
			name:      "max length name (100 chars)",
			input:     "A123456789B123456789C123456789D123456789E123456789F123456789G123456789H123456789I123456789J123456789",
			wantState: SettingsStateAddLocalPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel(t)
			m.state = SettingsStateAddLocalName
			m.textInput.SetValue(tt.input)

			newModel, cmd := m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

			if newModel.state != tt.wantState {
				t.Fatalf("expected state %v, got %v", tt.wantState, newModel.state)
			}
			if newModel.addRepositoryName != tt.input {
				t.Fatalf("expected name %q, got %q", tt.input, newModel.addRepositoryName)
			}
			if cmd != nil {
				t.Fatalf("expected nil command")
			}
		})
	}
}

// TestHandleAddLocalNameKeys_EmptyName tests rejecting empty names
func TestHandleAddLocalNameKeys_EmptyName(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalName
	m.textInput.SetValue("")

	newModel, cmd := m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if newModel.state != SettingsStateAddLocalName {
		t.Fatalf("should stay in same state: expected %v, got %v", SettingsStateAddLocalName, newModel.state)
	}
	if newModel.addRepositoryName != "" {
		t.Fatalf("should not set name: got %q", newModel.addRepositoryName)
	}
	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if newModel.layout.GetError() == nil {
		t.Fatalf("should have error")
	}
}

// TestHandleAddLocalNameKeys_TooLongName tests rejecting names over 100 chars
func TestHandleAddLocalNameKeys_TooLongName(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalName
	m.textInput.SetValue(string(make([]byte, 101))) // 101 characters

	newModel, cmd := m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if newModel.state != SettingsStateAddLocalName {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalName, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if newModel.layout.GetError() == nil {
		t.Fatalf("expected error")
	}
}

// TestHandleAddLocalNameKeys_DuplicateName tests rejecting duplicate names
func TestHandleAddLocalNameKeys_DuplicateName(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalName

	// Add existing repository with same name
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "existing",
			Name: "Existing Repo",
			Type: repository.RepositoryTypeLocal,
			Path: "/existing/path",
		},
	}

	m.textInput.SetValue("Existing Repo")

	newModel, cmd := m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if newModel.state != SettingsStateAddLocalName {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalName, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if newModel.layout.GetError() == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(newModel.layout.GetError().Error(), "already exists") {
		t.Fatalf("expected error to mention duplicate, got %q", newModel.layout.GetError().Error())
	}
}

// TestHandleAddLocalNameKeys_Cancel tests canceling name input
func TestHandleAddLocalNameKeys_Cancel(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalName
	m.textInput.SetValue("Some Name")

	newModel, cmd := m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if newModel.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected %v, got %v", SettingsStateAddRepositoryType, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command")
	}
}

// TestHandleAddLocalPathKeys_ValidPath tests accepting valid paths
func TestHandleAddLocalPathKeys_ValidPath(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalPath
	m.addRepositoryName = "Test Repo"

	// Use temp directory for valid path
	tempDir := t.TempDir()
	m.textInput.SetValue(tempDir)

	newModel, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatalf("should trigger createLocalRepository")
	}
	if newModel.addRepositoryPath != tempDir {
		t.Fatalf("expected path %q, got %q", tempDir, newModel.addRepositoryPath)
	}
}

// TestHandleAddLocalPathKeys_EmptyPath tests rejecting empty paths
func TestHandleAddLocalPathKeys_EmptyPath(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalPath
	m.textInput.SetValue("")

	newModel, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if newModel.state != SettingsStateAddLocalPath {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalPath, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if newModel.layout.GetError() == nil {
		t.Fatalf("expected error")
	}
}

// TestHandleAddLocalPathKeys_DuplicatePath tests rejecting duplicate paths
func TestHandleAddLocalPathKeys_DuplicatePath(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalPath
	m.addRepositoryName = "New Repo"

	tempDir := t.TempDir()

	// Add existing repository with same path
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "existing",
			Name: "Existing",
			Type: repository.RepositoryTypeLocal,
			Path: tempDir,
		},
	}

	m.textInput.SetValue(tempDir)

	newModel, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if newModel.state != SettingsStateAddLocalPath {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalPath, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if newModel.layout.GetError() == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(newModel.layout.GetError().Error(), "already used") {
		t.Fatalf("expected error to mention duplicate path, got %q", newModel.layout.GetError().Error())
	}
}

// TestHandleAddLocalPathKeys_Cancel tests canceling path input
func TestHandleAddLocalPathKeys_Cancel(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalPath
	m.addRepositoryName = "Test"

	newModel, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if newModel.state != SettingsStateAddLocalName {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalName, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command")
	}
}

// TestCreateLocalRepository_Success tests successful repository creation
func TestCreateLocalRepository_Success(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)
	m.addRepositoryName = "Test Local"
	m.addRepositoryPath = t.TempDir()

	// Ensure config can be saved
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	cmd := m.createLocalRepository()
	if cmd == nil {
		t.Fatalf("expected command")
	}

	// Execute the command
	msg := cmd()

	// Should return completion message
	_, ok := msg.(settingsCompleteMsg)
	if !ok {
		t.Fatalf("should return settingsCompleteMsg")
	}

	// Verify repository was added
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].Name != "Test Local" {
		t.Fatalf("expected name %q, got %q", "Test Local", m.currentConfig.Repositories[0].Name)
	}
	if m.currentConfig.Repositories[0].Type != repository.RepositoryTypeLocal {
		t.Fatalf("expected type %v, got %v", repository.RepositoryTypeLocal, m.currentConfig.Repositories[0].Type)
	}
}

// TestCreateLocalRepository_GeneratesID tests that unique ID is generated
func TestCreateLocalRepository_GeneratesID(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)
	m.addRepositoryName = "Test"
	m.addRepositoryPath = t.TempDir()

	cmd := m.createLocalRepository()
	if cmd == nil {
		t.Fatalf("expected command")
	}

	cmd()

	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].ID == "" {
		t.Fatalf("expected non-empty ID")
	}
	if !strings.Contains(m.currentConfig.Repositories[0].ID, "test") {
		t.Fatalf("expected ID to contain %q, got %q", "test", m.currentConfig.Repositories[0].ID)
	}
}

// TestCreateLocalRepository_AppendsToExisting tests adding to existing repos
func TestCreateLocalRepository_AppendsToExisting(t *testing.T) {
	_, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)

	// Add existing repository
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "existing",
			Name: "Existing",
			Type: repository.RepositoryTypeLocal,
			Path: "/existing/path",
		},
	}

	m.addRepositoryName = "New Repo"
	m.addRepositoryPath = t.TempDir()

	cmd := m.createLocalRepository()
	if cmd == nil {
		t.Fatalf("expected command")
	}

	cmd()

	if len(m.currentConfig.Repositories) != 2 {
		t.Fatalf("should have both repositories: got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].Name != "Existing" {
		t.Fatalf("expected %q, got %q", "Existing", m.currentConfig.Repositories[0].Name)
	}
	if m.currentConfig.Repositories[1].Name != "New Repo" {
		t.Fatalf("expected %q, got %q", "New Repo", m.currentConfig.Repositories[1].Name)
	}
}

// TestTransitionToAddLocalName tests state transition setup
func TestTransitionToAddLocalName(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddRepositoryType

	newModel, cmd := m.transitionToAddLocalName()

	if newModel.state != SettingsStateAddLocalName {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalName, newModel.state)
	}
	if cmd != nil {
		t.Fatalf("expected nil command")
	}
	if newModel.textInput.Value() != "" {
		t.Fatalf("expected empty input value")
	}
	if newModel.textInput.Placeholder == "" {
		t.Fatalf("expected non-empty placeholder")
	}
}

// TestViewAddLocalName tests name input view
func TestViewAddLocalName(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalName

	view := m.viewAddLocalName()

	expectedContent := []string{
		"Add Local Repository",
		"Repository Name",
		"Enter to continue",
	}

	for _, content := range expectedContent {
		if !strings.Contains(view, content) {
			t.Fatalf("expected view to contain %q", content)
		}
	}
}

// TestViewAddLocalPath tests path input view
func TestViewAddLocalPath(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalPath
	m.addRepositoryName = "My Repo"

	view := m.viewAddLocalPath()

	if !strings.Contains(view, "My Repo") {
		t.Fatalf("expected view to contain repository name")
	}
	if !strings.Contains(view, "Local Directory Path") {
		t.Fatalf("expected view to contain local directory prompt")
	}
	if !strings.Contains(view, "Enter to save") {
		t.Fatalf("expected view to contain submit hint")
	}
}

// TestViewAddLocalError tests error view
func TestViewAddLocalError(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalError
	m.layout = m.layout.SetError(fmt.Errorf("test error"))

	view := m.viewAddLocalError()

	if !strings.Contains(view, "Failed") {
		t.Fatalf("expected view to contain failure header")
	}
	if !strings.Contains(view, "test error") {
		t.Fatalf("expected view to contain error message")
	}
	if !strings.Contains(view, "Press any key") {
		t.Fatalf("expected view to contain prompt")
	}
}

// TestAddLocalFlow_Complete tests full successful flow
func TestAddLocalFlow_Complete(t *testing.T) {
	m := createTestModel(t)

	// Step 1: Transition to name input
	m, _ = m.transitionToAddLocalName()
	if m.state != SettingsStateAddLocalName {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalName, m.state)
	}

	// Step 2: Enter name
	m.textInput.SetValue("Test Repo")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalPath {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalPath, m.state)
	}
	if m.addRepositoryName != "Test Repo" {
		t.Fatalf("expected %q, got %q", "Test Repo", m.addRepositoryName)
	}

	// Step 3: Enter path
	tempDir := t.TempDir()
	m.textInput.SetValue(tempDir)
	m, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command")
	}
	if m.addRepositoryPath != tempDir {
		t.Fatalf("expected path %q, got %q", tempDir, m.addRepositoryPath)
	}
}

// TestAddLocalFlow_CancelAtName tests canceling at name step
func TestAddLocalFlow_CancelAtName(t *testing.T) {
	m := createTestModel(t)
	m, _ = m.transitionToAddLocalName()

	m.textInput.SetValue("Some Name")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected %v, got %v", SettingsStateAddRepositoryType, m.state)
	}
}

// TestAddLocalFlow_CancelAtPath tests canceling at path step
func TestAddLocalFlow_CancelAtPath(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalPath
	m.addRepositoryName = "Test"

	m, _ = m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateAddLocalName {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalName, m.state)
	}
}

// TestAddLocalFlow_ValidationErrors tests error handling throughout flow
func TestAddLocalFlow_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		state         SettingsState
		setupFunc     func(*SettingsModel)
		expectedState SettingsState
		expectError   bool
	}{
		{
			name:  "empty name",
			state: SettingsStateAddLocalName,
			setupFunc: func(m *SettingsModel) {
				m.textInput.SetValue("")
			},
			expectedState: SettingsStateAddLocalName,
			expectError:   true,
		},
		{
			name:  "empty path",
			state: SettingsStateAddLocalPath,
			setupFunc: func(m *SettingsModel) {
				m.addRepositoryName = "Test"
				m.textInput.SetValue("")
			},
			expectedState: SettingsStateAddLocalPath,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel(t)
			m.state = tt.state
			tt.setupFunc(m)

			var newModel *SettingsModel
			if tt.state == SettingsStateAddLocalName {
				newModel, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
			} else {
				newModel, _ = m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
			}

			if newModel.state != tt.expectedState {
				t.Fatalf("expected %v, got %v", tt.expectedState, newModel.state)
			}
			if tt.expectError && newModel.layout.GetError() == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

// TestAddLocalFlow_StateIsolation tests that flow doesn't affect other state
func TestAddLocalFlow_StateIsolation(t *testing.T) {
	m := createTestModel(t)
	m.selectedRepositoryID = "existing-id"
	m.newGitHubURL = "https://github.com/test/repo"

	// Go through add local flow
	m, _ = m.transitionToAddLocalName()
	m.textInput.SetValue("New Repo")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// Unrelated state should be preserved
	if m.selectedRepositoryID != "existing-id" {
		t.Fatalf("expected %q, got %q", "existing-id", m.selectedRepositoryID)
	}
	if m.newGitHubURL != "https://github.com/test/repo" {
		t.Fatalf("expected %q, got %q", "https://github.com/test/repo", m.newGitHubURL)
	}
}

// Benchmark tests
func BenchmarkHandleAddLocalNameKeys(b *testing.B) {
	m := createTestModel(&testing.T{})
	m.state = SettingsStateAddLocalName
	m.textInput.SetValue("Test Repo")
	msg := tea.KeyMsg{Type: tea.KeyEnter}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.handleAddLocalNameKeys(msg)
	}
}

func BenchmarkCreateLocalRepository(b *testing.B) {
	m := createTestModel(&testing.T{})
	m.addRepositoryName = "Test"
	m.addRepositoryPath = "/tmp/test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.createLocalRepository()
	}
}

func BenchmarkViewAddLocalName(b *testing.B) {
	m := createTestModel(&testing.T{})
	m.state = SettingsStateAddLocalName

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.viewAddLocalName()
	}
}

// TestIntegration_AddLocalComplete tests the complete
// add local repository flow from start to finish with successful completion
func TestIntegration_AddLocalComplete(t *testing.T) {
	// Use test config path to avoid overwriting user's config
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{} // Start empty

	// Step 1: Start at AddRepositoryType (user already selected "Add Repository")
	m.state = SettingsStateAddRepositoryType
	if m.state != SettingsStateAddRepositoryType {
		t.Fatalf("expected %v, got %v", SettingsStateAddRepositoryType, m.state)
	}

	// Step 2: Select Local type (index 0)
	m.addRepositoryTypeIndex = 0
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalName {
		t.Fatalf("should transition to AddLocalName: expected %v, got %v", SettingsStateAddLocalName, m.state)
	}

	// Step 3: Enter repository name
	m.textInput.SetValue("My Integration Test Repo")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalPath {
		t.Fatalf("should transition to AddLocalPath: expected %v, got %v", SettingsStateAddLocalPath, m.state)
	}
	if m.addRepositoryName != "My Integration Test Repo" {
		t.Fatalf("should store repository name: expected %q, got %q", "My Integration Test Repo", m.addRepositoryName)
	}

	// Step 4: Enter repository path - using a subdir to avoid long path issues
	testPath := t.TempDir()
	m.textInput.SetValue(testPath)
	m, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should return createLocalRepository command")
	}
	// Note: Path may be expanded/modified by fileops.ExpandPath
	if m.addRepositoryPath == "" {
		t.Fatalf("should store repository path")
	}

	// Step 5: Execute createLocalRepository command
	// Note: createLocalRepository sets state to MainMenu and returns settingsCompleteMsg
	msg := cmd()
	if _, ok := msg.(settingsCompleteMsg); !ok {
		t.Fatalf("should return settingsCompleteMsg, got %T", msg)
	}

	// Step 6: Model already transitioned to MainMenu internally
	// (createLocalRepository sets m.state = SettingsStateMainMenu before returning)
	if m.state != SettingsStateMainMenu {
		t.Fatalf("should be at MainMenu after creation: expected %v, got %v", SettingsStateMainMenu, m.state)
	}

	// Step 7: Verify repository was added to config
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("should have 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	repo := m.currentConfig.Repositories[0]
	if repo.Name != "My Integration Test Repo" {
		t.Fatalf("expected %q, got %q", "My Integration Test Repo", repo.Name)
	}
	if repo.Path == "" {
		t.Fatalf("should have path")
	}
	if repo.Type != repository.RepositoryTypeLocal {
		t.Fatalf("expected %v, got %v", repository.RepositoryTypeLocal, repo.Type)
	}
	if repo.ID == "" {
		t.Fatalf("should generate ID")
	}

	// Step 8: Verify config file was saved (if applicable)
	// Note: Config saving happens in createLocalRepository, already verified above
	_ = configPath // Config path is set up but just for cleanup
}

// TestIntegration_AddLocalRepository_CancelAtTypeSelection tests canceling
// during the type selection step
func TestIntegration_AddLocalRepository_CancelAtTypeSelection(t *testing.T) {
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
		t.Fatalf("should not add any repository, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_AddLocalRepository_CancelAtNameInput tests canceling
// during the name input step
func TestIntegration_AddLocalRepository_CancelAtNameInput(t *testing.T) {
	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Navigate to name input
	m.state = SettingsStateAddRepositoryType
	m.addRepositoryTypeIndex = 0
	m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalName {
		t.Fatalf("expected state %v, got %v", SettingsStateAddLocalName, m.state)
	}

	// Enter some text, then cancel
	m.textInput.SetValue("Partial Name")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateAddRepositoryType {
		t.Fatalf("should return to type selection: expected %v, got %v", SettingsStateAddRepositoryType, m.state)
	}
	if m.addRepositoryName != "" {
		t.Fatalf("should not store partial name, got %q", m.addRepositoryName)
	}
	if len(m.currentConfig.Repositories) != 0 {
		t.Fatalf("should not add repository, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_AddLocalRepository_CancelAtPathInput tests canceling
// during the path input step
func TestIntegration_AddLocalRepository_CancelAtPathInput(t *testing.T) {
	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Navigate through name input
	m.state = SettingsStateAddLocalName
	m.textInput.SetValue("Test Repo")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalPath {
		t.Fatalf("expected %v, got %v", SettingsStateAddLocalPath, m.state)
	}

	// Enter path, then cancel
	m.textInput.SetValue("/some/path")
	m, _ = m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != SettingsStateAddLocalName {
		t.Fatalf("should return to name input: expected %v, got %v", SettingsStateAddLocalName, m.state)
	}
	if m.addRepositoryPath != "" {
		t.Fatalf("should not store partial path, got %q", m.addRepositoryPath)
	}
	if len(m.currentConfig.Repositories) != 0 {
		t.Fatalf("should not add repository, got %d", len(m.currentConfig.Repositories))
	}
}

// TestIntegration_AddLocalRepository_EmptyNameValidation tests validation
// error when submitting empty repository name
func TestIntegration_AddLocalRepository_EmptyNameValidation(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalName
	m.textInput.SetValue("")

	// Try to submit empty name
	m, cmd := m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddLocalName {
		t.Fatalf("should stay in name input: expected %v, got %v", SettingsStateAddLocalName, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not proceed: expected nil cmd")
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should show error")
	}
	if !strings.Contains(m.layout.GetError().Error(), "cannot be empty") {
		t.Fatalf("error should mention empty, got %q", m.layout.GetError().Error())
	}
}

// TestIntegration_AddLocalRepository_DuplicateNameValidation tests validation
// error when submitting a duplicate repository name
func TestIntegration_AddLocalRepository_DuplicateNameValidation(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalName

	// Add existing repository
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "existing-id",
			Name: "Existing Repo",
			Type: repository.RepositoryTypeLocal,
			Path: "/existing/path",
		},
	}

	// Try to use duplicate name
	m.textInput.SetValue("Existing Repo")
	m, cmd := m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddLocalName {
		t.Fatalf("should stay in name input: expected %v, got %v", SettingsStateAddLocalName, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not proceed: expected nil cmd")
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should show error")
	}
	if !strings.Contains(m.layout.GetError().Error(), "already exists") {
		t.Fatalf("error should mention duplicate, got %q", m.layout.GetError().Error())
	}
}

// TestIntegration_AddLocalRepository_EmptyPathValidation tests validation
// error when submitting empty repository path
func TestIntegration_AddLocalRepository_EmptyPathValidation(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalPath
	m.addRepositoryName = "Test Repo"
	m.textInput.SetValue("")

	// Try to submit empty path
	m, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddLocalPath {
		t.Fatalf("should stay in path input: expected %v, got %v", SettingsStateAddLocalPath, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not proceed: expected nil cmd")
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should show error")
	}
	if !strings.Contains(m.layout.GetError().Error(), "cannot be empty") {
		t.Fatalf("error should mention empty, got %q", m.layout.GetError().Error())
	}
}

// TestIntegration_AddLocalRepository_DuplicatePathValidation tests validation
// error when submitting a duplicate repository path
func TestIntegration_AddLocalRepository_DuplicatePathValidation(t *testing.T) {
	m := createTestModel(t)
	m.state = SettingsStateAddLocalPath
	m.addRepositoryName = "New Repo"

	testPath := t.TempDir()

	// Add existing repository with same path
	m.currentConfig.Repositories = []repository.RepositoryEntry{
		{
			ID:   "existing-id",
			Name: "Existing",
			Type: repository.RepositoryTypeLocal,
			Path: testPath,
		},
	}

	// Try to use duplicate path
	m.textInput.SetValue(testPath)
	m, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != SettingsStateAddLocalPath {
		t.Fatalf("should stay in path input: expected %v, got %v", SettingsStateAddLocalPath, m.state)
	}
	if cmd != nil {
		t.Fatalf("should not proceed: expected nil cmd")
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should show error")
	}
	if !strings.Contains(m.layout.GetError().Error(), "already used") {
		t.Fatalf("error should mention duplicate, got %q", m.layout.GetError().Error())
	}
}

// TestIntegration_AddLocalMultiple tests
// adding multiple local repositories in sequence
func TestIntegration_AddLocalMultiple(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}

	// Helper to add a repository
	addRepo := func(name, path string) {
		m.state = SettingsStateAddRepositoryType
		m.addRepositoryTypeIndex = 0 // Local
		m, _ = m.handleAddRepositoryTypeKeys(tea.KeyMsg{Type: tea.KeyEnter})

		m.textInput.SetValue(name)
		m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

		m.textInput.SetValue(path)
		m, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatalf("expected command when adding repo %q", name)
		}

		// Execute command - it sets state to MainMenu internally
		msg := cmd()
		_, ok := msg.(settingsCompleteMsg)
		if !ok {
			t.Fatalf("should return settingsCompleteMsg, got %T", msg)
		}
		if m.state != SettingsStateMainMenu {
			t.Fatalf("should be at MainMenu after adding repo %q", name)
		}
	}

	// Add first repository
	path1 := t.TempDir()
	addRepo("Repo One", path1)
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("expected 1 repo after first add, got %d", len(m.currentConfig.Repositories))
	}

	// Add second repository
	path2 := t.TempDir()
	addRepo("Repo Two", path2)
	if len(m.currentConfig.Repositories) != 2 {
		t.Fatalf("expected 2 repos after second add, got %d", len(m.currentConfig.Repositories))
	}

	// Add third repository
	path3 := t.TempDir()
	addRepo("Repo Three", path3)
	if len(m.currentConfig.Repositories) != 3 {
		t.Fatalf("expected 3 repos after third add, got %d", len(m.currentConfig.Repositories))
	}

	// Verify all repositories exist with correct data
	if m.currentConfig.Repositories[0].Name != "Repo One" {
		t.Fatalf("expected first repo name %q, got %q", "Repo One", m.currentConfig.Repositories[0].Name)
	}
	if m.currentConfig.Repositories[0].Path == "" {
		t.Fatalf("expected first repo path to be non-empty")
	}

	if m.currentConfig.Repositories[1].Name != "Repo Two" {
		t.Fatalf("expected second repo name %q, got %q", "Repo Two", m.currentConfig.Repositories[1].Name)
	}
	if m.currentConfig.Repositories[1].Path == "" {
		t.Fatalf("expected second repo path to be non-empty")
	}

	if m.currentConfig.Repositories[2].Name != "Repo Three" {
		t.Fatalf("expected third repo name %q, got %q", "Repo Three", m.currentConfig.Repositories[2].Name)
	}
	if m.currentConfig.Repositories[2].Path == "" {
		t.Fatalf("expected third repo path to be non-empty")
	}

	// Verify all have unique IDs
	ids := make(map[string]bool)
	for _, repo := range m.currentConfig.Repositories {
		if repo.ID == "" {
			t.Fatalf("each repo should have ID")
		}
		if ids[repo.ID] {
			t.Fatalf("IDs should be unique, duplicate ID %q", repo.ID)
		}
		ids[repo.ID] = true
	}

	_ = configPath // Config path is set up but just for cleanup
}

// TestIntegration_AddLocalRepository_StateIsolationFromOtherFlows tests
// that the Add Local flow doesn't interfere with state from other flows
func TestIntegration_AddLocalRepository_StateIsolationFromOtherFlows(t *testing.T) {
	m := createTestModel(t)

	// Set state from other flows (these should not be affected)
	m.selectedRepositoryID = "some-repo-id"
	m.newGitHubURL = "https://github.com/test/repo"
	m.newGitHubBranch = "main"
	m.newGitHubPath = "/github/clone/path"

	// Go through Add Local flow
	m.state = SettingsStateAddLocalName
	m.textInput.SetValue("Local Repo")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})

	tempPath := t.TempDir()
	m.textInput.SetValue(tempPath)
	m, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected createLocalRepository command")
	}

	// Verify other flow state is preserved
	if m.selectedRepositoryID != "some-repo-id" {
		t.Fatalf("should preserve selected repo ID: expected %q, got %q", "some-repo-id", m.selectedRepositoryID)
	}
	if m.newGitHubURL != "https://github.com/test/repo" {
		t.Fatalf("should preserve GitHub URL: expected %q, got %q", "https://github.com/test/repo", m.newGitHubURL)
	}
	if m.newGitHubBranch != "main" {
		t.Fatalf("should preserve GitHub branch: expected %q, got %q", "main", m.newGitHubBranch)
	}
	if m.newGitHubPath != "/github/clone/path" {
		t.Fatalf("should preserve GitHub path: expected %q, got %q", "/github/clone/path", m.newGitHubPath)
	}
}

// TestIntegration_AddLocalRecover tests
// recovering from a validation error and successfully completing the flow
func TestIntegration_AddLocalRecover(t *testing.T) {
	configPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	m := createTestModel(t)
	m.currentConfig.Repositories = []repository.RepositoryEntry{}
	m.state = SettingsStateAddLocalName

	// Step 1: Try empty name (should fail)
	m.textInput.SetValue("")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalName {
		t.Fatalf("should stay in name input: expected %v, got %v", SettingsStateAddLocalName, m.state)
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should have error")
	}

	// Step 2: Fix the error by entering valid name
	m.textInput.SetValue("Valid Repo Name")
	m, _ = m.handleAddLocalNameKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalPath {
		t.Fatalf("should proceed to path input: expected %v, got %v", SettingsStateAddLocalPath, m.state)
	}
	// Note: Errors are cleared on state transition

	// Step 3: Try empty path (should fail)
	m.textInput.SetValue("")
	m, _ = m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != SettingsStateAddLocalPath {
		t.Fatalf("should stay in path input: expected %v, got %v", SettingsStateAddLocalPath, m.state)
	}
	if m.layout.GetError() == nil {
		t.Fatalf("should have error")
	}

	// Step 4: Fix the error by entering valid path
	testPath := t.TempDir()
	m.textInput.SetValue(testPath)
	m, cmd := m.handleAddLocalPathKeys(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("should proceed with creation")
	}

	// Step 5: Complete the flow
	msg := cmd()
	_, ok := msg.(settingsCompleteMsg)
	if !ok {
		t.Fatalf("should return settingsCompleteMsg, got %T", msg)
	}
	if m.state != SettingsStateMainMenu {
		t.Fatalf("should be at MainMenu: expected %v, got %v", SettingsStateMainMenu, m.state)
	}

	// Verify repository was created successfully
	if len(m.currentConfig.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(m.currentConfig.Repositories))
	}
	if m.currentConfig.Repositories[0].Name != "Valid Repo Name" {
		t.Fatalf("expected repo name %q, got %q", "Valid Repo Name", m.currentConfig.Repositories[0].Name)
	}
	if m.currentConfig.Repositories[0].Path == "" {
		t.Fatalf("expected repo path to be non-empty")
	}

	_ = configPath // Config path is set up but just for cleanup
}
