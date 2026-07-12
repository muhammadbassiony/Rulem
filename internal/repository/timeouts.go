package repository

import "time"

// Network operation timeouts bound the go-git operations that contact a remote
// so a hung connection can never freeze the caller (for example the TUI's
// spinner, which would otherwise spin forever). They are applied inside this
// package, at each network operation, by deriving a context.WithTimeout from
// the caller-supplied context. Callers therefore pass a plain context (usually
// context.Background()) and never need to know these values; they may cancel
// that context to abort the work early, and the cancellation still propagates
// through the internally derived deadline.
const (
	// cloneTimeout bounds initial clone paths, which transfer the most data.
	cloneTimeout = 120 * time.Second

	// fetchTimeout bounds fetch/sync paths against already-cloned repositories
	// (FetchUpdates, manual refresh, post-branch-change sync).
	fetchTimeout = 60 * time.Second

	// validationTimeout bounds lightweight remote checks such as the ls-remote
	// used to validate a PAT and repository access.
	validationTimeout = 10 * time.Second
)
