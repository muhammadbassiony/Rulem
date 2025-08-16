package setupmenu

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rulem/internal/logging"
	"rulem/internal/tui/helpers"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestSuccessfulSetup tests the entire successful setup workflow
func TestSuccessfulSetup(t *testing.T) {
	// Set up test config path to prevent overwriting real config
	testConfigPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	// Create temporary storage directory for testing
	tempStorageDir := CreateTempStorageDir(t, "setupmenu-complete-test-*")
	validTestPath := filepath.Join(tempStorageDir, "complete-test-storage")

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(100, 30, nil, logger)

	model := NewSetupModel(ctx)
	testmodel := teatest.NewTestModel(t, model)

	// Step 1: Welcome screen
	waitForString(t, testmodel, "Welcome to Rulem")

	// Step 2: Go to storage input
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForString(t, testmodel, "Storage Directory")

	// Step 3: Clear existing text and enter new path
	testmodel.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line (Unix shortcut)
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validTestPath)})
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 4: Confirm configuration
	waitForString(t, testmodel, "Confirm Configuration")
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Step 5: Setup complete
	waitForString(t, testmodel, "Setup Complete")

	// Verify that config was created in test location, not real location
	if !fileExists(testConfigPath) {
		t.Error("Test config file should have been created")
	}

	// Verify that storage directory was created
	if !fileExists(validTestPath) {
		t.Error("Storage directory should have been created")
	}
}

// TestCancelledAtWelcome tests cancellation at welcome screen
func TestCancelledAtWelcome(t *testing.T) {
	// Set up test config path to prevent overwriting real config
	testConfigPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(100, 30, nil, logger)

	model := NewSetupModel(ctx)
	testmodel := teatest.NewTestModel(t, model)

	// Step 1: Welcome screen
	waitForString(t, testmodel, "Welcome to Rulem")

	// Step 2: Cancel with Escape
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEscape})
	waitForString(t, testmodel, "Setup Cancelled")

	// Verify no config was created
	if fileExists(testConfigPath) {
		t.Error("Test config file should not have been created when cancelled")
	}
}

// TestCancelledAtStorageInput tests cancellation at storage input
func TestCancelledAtStorageInput(t *testing.T) {
	// Set up test config path to prevent overwriting real config
	testConfigPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(100, 30, nil, logger)

	model := NewSetupModel(ctx)
	testmodel := teatest.NewTestModel(t, model)

	// Step 1: Welcome screen
	waitForString(t, testmodel, "Welcome to Rulem")

	// Step 2: Go to storage input
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForString(t, testmodel, "Storage Directory")

	// Step 3: Cancel with Escape
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEscape})
	waitForString(t, testmodel, "Setup Cancelled")

	// Verify no config was created
	if fileExists(testConfigPath) {
		t.Error("Test config file should not have been created when cancelled")
	}
}

// TestBackAndForthNavigation tests going back and forth between states
func TestBackAndForthNavigation(t *testing.T) {
	// Set up test config path to prevent overwriting real config
	testConfigPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	// Create temporary directories for testing
	tempStorageDir := CreateTempStorageDir(t, "setupmenu-navigation-test-*")
	firstPath := filepath.Join(tempStorageDir, "first-path")
	finalPath := filepath.Join(tempStorageDir, "final-path")

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(100, 30, nil, logger)

	model := NewSetupModel(ctx)
	testmodel := teatest.NewTestModel(t, model)

	// Step 1: Welcome screen
	waitForString(t, testmodel, "Welcome to Rulem")

	// Step 2: Go to storage input
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForString(t, testmodel, "Storage Directory")

	// Step 3: Enter first path
	testmodel.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line (Unix shortcut)
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(firstPath)})
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 4: Go to confirmation, then go back
	waitForString(t, testmodel, "Confirm Configuration")
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}) // Go back

	// Step 5: Should be back at storage input
	waitForString(t, testmodel, "Storage Directory")

	// Step 6: Clear and enter final path
	testmodel.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line (Unix shortcut)
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(finalPath)})
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 7: Confirm final configuration
	waitForString(t, testmodel, "Confirm Configuration")
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Step 8: Setup complete
	waitForString(t, testmodel, "Setup Complete")

	// Verify that config was created with final path
	if !fileExists(testConfigPath) {
		t.Error("Test config file should have been created")
	}

	// Verify that final storage directory was created
	if !fileExists(finalPath) {
		t.Error("Final storage directory should have been created")
	}
}

// TestInvalidStoragePath tests handling of invalid storage paths
func TestInvalidStoragePath(t *testing.T) {
	// Set up test config path to prevent overwriting real config
	testConfigPath, cleanup := SetTestConfigPath(t)
	defer cleanup()

	logger, _ := logging.NewTestLogger()
	ctx := helpers.NewUIContext(100, 30, nil, logger)

	model := NewSetupModel(ctx)
	testmodel := teatest.NewTestModel(t, model)

	// Step 1: Welcome screen
	waitForString(t, testmodel, "Welcome to Rulem")

	// Step 2: Go to storage input
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForString(t, testmodel, "Storage Directory")

	// Step 3: Enter invalid path (empty string)
	testmodel.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 4: Should show error message
	waitForString(t, testmodel, "Error: storage directory cannot be empty")

	// Step 5: Enter valid path to recover
	tempStorageDir := CreateTempStorageDir(t, "setupmenu-recovery-test-*")
	validPath := filepath.Join(tempStorageDir, "valid-path")
	testmodel.Send(tea.KeyMsg{Type: tea.KeyCtrlU}) // Clear line
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(validPath)})
	testmodel.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Step 6: Should proceed to confirmation
	waitForString(t, testmodel, "Confirm Configuration")
	testmodel.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	// Step 7: Setup complete
	waitForString(t, testmodel, "Setup Complete")

	// Verify config was created
	if !fileExists(testConfigPath) {
		t.Error("Test config file should have been created")
	}
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

// Helper function to check if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
