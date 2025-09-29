package tui

import (
	"testing"

	"rulem/internal/config"
	"rulem/internal/logging"
	"rulem/internal/repository"
)

func createTestConfigWithPath(path string) *config.Config {
	return &config.Config{Central: repository.CentralRepositoryConfig{Path: path}}
}

func TestNewMainModel(t *testing.T) {
	cfg := createTestConfigWithPath("/test/path")
	logger, _ := logging.NewTestLogger()

	model := NewMainModel(cfg, logger)

	if model.config != cfg {
		t.Error("Config not properly set")
	}

	if model.logger != logger {
		t.Error("Logger not properly set")
	}

	if model.state != StateMenu {
		t.Errorf("Expected initial state to be StateMenu, got %v", model.state)
	}

	if model.prevState != StateMenu {
		t.Errorf("Expected initial prevState to be StateMenu, got %v", model.prevState)
	}
}

func TestMainModelInit(t *testing.T) {
	cfg := createTestConfigWithPath("/test/path")
	logger, _ := logging.NewTestLogger()

	model := NewMainModel(cfg, logger)
	cmd := model.Init()

	// Init should not return a command for the main model
	if cmd != nil {
		t.Error("Init should not return a command")
	}
}

func TestGetUIContext(t *testing.T) {
	cfg := createTestConfigWithPath("/test/path")
	logger, _ := logging.NewTestLogger()

	model := NewMainModel(cfg, logger)
	model.windowWidth = 100
	model.windowHeight = 50

	ctx := model.GetUIContext()

	if ctx.Width != 100 {
		t.Errorf("Expected width 100, got %d", ctx.Width)
	}

	if ctx.Height != 50 {
		t.Errorf("Expected height 50, got %d", ctx.Height)
	}

	if ctx.Config != cfg {
		t.Error("Config not properly set in context")
	}

	if ctx.Logger != logger {
		t.Error("Logger not properly set in context")
	}
}

func TestHasValidDimensions(t *testing.T) {
	cfg := createTestConfigWithPath("/test/path")
	logger, _ := logging.NewTestLogger()

	model := NewMainModel(cfg, logger)

	// Test invalid dimensions
	model.windowWidth = 0
	model.windowHeight = 0
	if model.hasValidDimensions() {
		t.Error("Should return false for zero dimensions")
	}

	model.windowWidth = -1
	model.windowHeight = 50
	if model.hasValidDimensions() {
		t.Error("Should return false for negative width")
	}

	model.windowWidth = 50
	model.windowHeight = -1
	if model.hasValidDimensions() {
		t.Error("Should return false for negative height")
	}

	// Test valid dimensions
	model.windowWidth = 80
	model.windowHeight = 24
	if !model.hasValidDimensions() {
		t.Error("Should return true for valid dimensions")
	}
}

func TestModelReinitialization(t *testing.T) {
	cfg := createTestConfigWithPath("/test/path")
	logger, _ := logging.NewTestLogger()

	model := NewMainModel(cfg, logger)
	model.windowWidth = 80
	model.windowHeight = 24

	// Test that models are always fresh (not cached)
	settingsModel1 := model.getOrInitializeModel(StateSettings)
	settingsModel2 := model.getOrInitializeModel(StateSettings)

	// Should be different instances since we don't cache
	if settingsModel1 == settingsModel2 {
		t.Error("Models should be fresh instances, not cached")
	}
}

func TestReturnToMenu(t *testing.T) {
	cfg := createTestConfigWithPath("/test/path")
	logger, _ := logging.NewTestLogger()

	model := NewMainModel(cfg, logger)

	// Set up some state that should be cleared
	model.state = StateSettings
	model.err = &testError{"test error"}
	model.comingSoonFeature = "test feature"

	// Return to menu
	updatedModel := model.returnToMenu()
	mainModel := updatedModel.(*MainModel)

	if mainModel.state != StateMenu {
		t.Errorf("Expected state StateMenu, got %v", mainModel.state)
	}

	if mainModel.activeModel != nil {
		t.Error("Active model should be nil after returning to menu")
	}

	if mainModel.err != nil {
		t.Error("Error should be nil after returning to menu")
	}

	if mainModel.comingSoonFeature != "" {
		t.Error("Coming soon feature should be empty after returning to menu")
	}
}

func TestGetOrInitializeModel(t *testing.T) {
	cfg := createTestConfigWithPath("/test/path")
	logger, _ := logging.NewTestLogger()

	model := NewMainModel(cfg, logger)
	model.windowWidth = 80
	model.windowHeight = 24

	// Test settings model initialization
	settingsModel := model.getOrInitializeModel(StateSettings)
	if settingsModel == nil {
		t.Error("Settings model should be initialized")
	}

	// Test that fresh model is returned on second call (no caching)
	settingsModel2 := model.getOrInitializeModel(StateSettings)
	if settingsModel == settingsModel2 {
		t.Error("Should return fresh settings model instance, not cached")
	}

	// Test save rules model initialization
	saveRulesModel := model.getOrInitializeModel(StateSaveRules)
	if saveRulesModel == nil {
		t.Error("Save rules model should be initialized")
	}

	// Test implemented model
	importCopyModel := model.getOrInitializeModel(StateImportCopy)
	if importCopyModel == nil {
		t.Error("Import copy model should be created")
	}
}

func TestFreshModelGeneration(t *testing.T) {
	cfg := createTestConfigWithPath("/test/path")
	logger, _ := logging.NewTestLogger()

	model := NewMainModel(cfg, logger)
	model.windowWidth = 80
	model.windowHeight = 24

	// Test that each call creates a fresh model
	settingsModel := model.getOrInitializeModel(StateSettings)
	if settingsModel == nil {
		t.Error("Settings model should be created")
	}

	saveRulesModel := model.getOrInitializeModel(StateSaveRules)
	if saveRulesModel == nil {
		t.Error("Save rules model should be created")
	}

	// Test implemented model
	importCopyModel := model.getOrInitializeModel(StateImportCopy)
	if importCopyModel == nil {
		t.Error("Import copy model should be created")
	}
}

func TestViewMethods(t *testing.T) {
	cfg := createTestConfigWithPath("/test/path")
	logger, _ := logging.NewTestLogger()

	model := NewMainModel(cfg, logger)

	// Test menu view
	model.state = StateMenu
	view := model.View()
	if view == "" {
		t.Error("Menu view should not be empty")
	}

	// Test error view
	model.state = StateError
	model.err = &testError{"test error"}
	view = model.View()
	if view == "" {
		t.Error("Error view should not be empty")
	}

	// Test coming soon view
	model.state = StateComingSoon
	model.comingSoonFeature = "test feature"
	view = model.View()
	if view == "" {
		t.Error("Coming soon view should not be empty")
	}

	// Test quitting view
	model.state = StateQuitting
	view = model.View()
	if view == "" {
		t.Error("Quitting view should not be empty")
	}
}

// Helper test error type
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
