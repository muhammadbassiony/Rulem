package repository

import (
	"rulem/internal/logging"
)

// CentralRepositoryConfig represents the configuration for the central rules repository.
// It supports both local directories and remote Git repositories.
type CentralRepositoryConfig struct {
	Path         string  `yaml:"path"`
	RemoteURL    *string `yaml:"remote_url,omitempty"`
	Branch       *string `yaml:"branch,omitempty"`
	LastSyncTime *int64  `yaml:"last_sync_time,omitempty"`
}

// IsRemote returns true if this configuration represents a remote Git repository.
func (c CentralRepositoryConfig) IsRemote() bool {
	return c.RemoteURL != nil && *c.RemoteURL != ""
}

// PrepareRepository creates the appropriate source and prepares it for use.
// This function examines the config to determine whether to use LocalSource or GitSource,
// then prepares the source and returns the local path and sync information.
//
// Parameters:
//   - cfg: Repository configuration
//   - logger: Logger for structured logging during preparation
//
// Returns:
//   - string: Absolute path to the prepared local repository
//   - SyncInfo: Information about sync operations for UI display
//   - error: Preparation errors (validation, network, authentication, etc.)
//
// Usage:
//
//	localPath, syncInfo, err := repository.PrepareRepository(cfg.Central, logger)
//	if err != nil {
//	    return fmt.Errorf("repository preparation failed: %w", err)
//	}
//	if syncInfo.Message != "" {
//	    logger.Info("Repository status", "message", syncInfo.Message)
//	}
//	fm := filemanager.NewFileManager(localPath, logger)
func PrepareRepository(cfg CentralRepositoryConfig, logger *logging.AppLogger) (string, SyncInfo, error) {
	if logger != nil {
		if cfg.IsRemote() {
			logger.Info("Preparing Git repository source", "remoteURL", *cfg.RemoteURL, "path", cfg.Path)
		} else {
			logger.Info("Preparing local repository source", "path", cfg.Path)
		}
	}

	var source Source
	if !cfg.IsRemote() {
		// Local repository mode - use the configured path directly
		source = NewLocalSource(cfg.Path)
	} else {
		// Git repository mode - will be implemented in Phase 3.6
		// For now, return LocalSource as fallback (this will be replaced with GitSource)
		// TODO: Implement GitSource in Phase 3.6
		//
		// Future implementation:
		// source = NewGitSource(*cfg.RemoteURL, cfg.Branch, cfg.Path)

		// Temporary fallback to LocalSource for Git configs
		// This ensures the system works with current LocalSource implementation
		source = NewLocalSource(cfg.Path)
	}

	return source.Prepare(logger)
}
