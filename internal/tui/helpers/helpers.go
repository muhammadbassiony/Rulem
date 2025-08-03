package helpers

import (
	"rulem/internal/config"
	"rulem/internal/logging"
)

// NavigateToMainMenuMsg is a common message for all submodels to navigate back to main menu
type NavigateToMainMenuMsg struct{}

// UIContext carries environment information needed for creating UI models
type UIContext struct {
	Width  int
	Height int
	Config *config.Config
	Logger *logging.AppLogger
}

// NewUIContext creates a new UI context with the provided parameters
func NewUIContext(width, height int, config *config.Config, logger *logging.AppLogger) UIContext {
	return UIContext{
		Width:  width,
		Height: height,
		Config: config,
		Logger: logger,
	}
}

// HasValidDimensions checks if the context has valid window dimensions
func (ctx UIContext) HasValidDimensions() bool {
	return ctx.Width > 0 && ctx.Height > 0
}
