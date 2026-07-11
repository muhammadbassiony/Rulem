package repository

import "time"

// Network operation timeouts bound the go-git operations that contact a remote
// so a hung connection can never freeze the caller (for example the TUI's
// spinner, which would otherwise spin forever). They are applied at the call
// boundaries via context.WithTimeout and flow through the package's network
// path as a context.Context.
const (
	// CloneTimeout bounds initial clone paths, which transfer the most data.
	// PrepareRepository / PrepareAllRepositories may clone, so their boundaries
	// use this larger budget.
	CloneTimeout = 120 * time.Second

	// FetchTimeout bounds fetch/sync paths against already-cloned repositories
	// (FetchUpdates, manual refresh, post-branch-change sync).
	FetchTimeout = 60 * time.Second

	// ValidationTimeout bounds lightweight checks such as ls-remote and branch
	// validation.
	ValidationTimeout = 30 * time.Second
)
