package setupmenu

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rulem/internal/filemanager"
	"rulem/internal/logging"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestSuccessfulSetup tests the entire successful setup workflow
func TestSuccessfulSetup(t *testing.T) {
	// Create a temporary directory for testing valid paths
	tempDir, err := os.MkdirTemp("", "setupmenu-complete-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validTestPath := filepath.Join(tempDir, "complete-test-storage")

	logger, _ := logging.NewTestLogger()

	model := NewSetupModel(logger)
	testmodel := teatest.NewTestModel(t, model)

	// Step 1: Welcome screen
	waitForString(t, testmodel, "Welcome to Rulem")

	// Step 2: Go to storage input
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForString(t, testmodel, "Storage Directory")

	// Step 3: Clear existing text and enter new path
	defaultPath := filemanager.GetDefaultStorageDir()
	for i := 0; i < len(defaultPath); i++ {
		testmodel.Send(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validTestPath)})
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 4: Confirm configuration
	waitForString(t, testmodel, "Confirm Configuration")
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Step 5: Setup complete
	waitForString(t, testmodel, "Setup Complete")

}

// TestCancelledAtWelcome tests cancellation at welcome screen
func TestCancelledAtWelcome(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	model := NewSetupModel(logger)
	testmodel := teatest.NewTestModel(t, model)

	// Step 1: Welcome screen
	waitForString(t, testmodel, "Welcome to Rulem")

	// Step 2: Cancel with Escape
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEscape})
	waitForString(t, testmodel, "Setup Cancelled")

}

// TestCancelledAtStorageInput tests cancellation at storage input
func TestCancelledAtStorageInput(t *testing.T) {
	logger, _ := logging.NewTestLogger()

	model := NewSetupModel(logger)
	testmodel := teatest.NewTestModel(t, model)

	// Step 1: Welcome screen
	waitForString(t, testmodel, "Welcome to Rulem")

	// Step 2: Go to storage input
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForString(t, testmodel, "Storage Directory")

	// Step 3: Cancel with Escape
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEscape})
	waitForString(t, testmodel, "Setup Cancelled")

}

// TestBackAndForthNavigation tests going back and forth between states
func TestBackAndForthNavigation(t *testing.T) {
	// Create temporary directories for testing
	tempDir, err := os.MkdirTemp("", "setupmenu-navigation-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	firstPath := filepath.Join(tempDir, "first-path")
	finalPath := filepath.Join(tempDir, "final-path")

	logger, _ := logging.NewTestLogger()

	model := NewSetupModel(logger)
	testmodel := teatest.NewTestModel(t, model)

	// Step 1: Welcome screen
	waitForString(t, testmodel, "Welcome to Rulem")

	// Step 2: Go to storage input
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForString(t, testmodel, "Storage Directory")

	// Step 3: Enter first path
	defaultPath := filemanager.GetDefaultStorageDir()
	for i := 0; i < len(defaultPath); i++ {
		testmodel.Send(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(firstPath)})
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 4: Go to confirmation, then go back
	waitForString(t, testmodel, "Confirm Configuration")
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}) // Go back

	// Step 5: Should be back at storage input
	waitForString(t, testmodel, "Storage Directory")

	// Step 6: Clear and enter final path
	// Clear the first path
	for i := 0; i < len(firstPath); i++ {
		testmodel.Send(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(finalPath)})
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 7: Confirm final configuration
	waitForString(t, testmodel, "Confirm Configuration")
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Step 8: Setup complete
	waitForString(t, testmodel, "Setup Complete")

}

// Helper function to wait for a specific string in the output
func waitForString(t *testing.T, tm *teatest.TestModel, s string) {
	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool {
			return strings.Contains(string(b), s)
		},
		teatest.WithCheckInterval(time.Millisecond*100),
		teatest.WithDuration(time.Second*3),
	)
}
