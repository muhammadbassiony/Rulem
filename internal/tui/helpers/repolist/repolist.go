// Package repolist provides shared repository list components for TUI menus.
//
// This package contains reusable list item types and builders for displaying
// and selecting repositories in various Bubble Tea TUI contexts. It is used by:
//   - saverulesmodel: Repository selection when saving rules to multiple repos
//   - settingsmenu: Repository list display and management
//
// The package provides:
//   - RepositoryListItem: A list.Item implementation for repository entries
//   - ActionListItem: A list.Item implementation for action items (add, delete)
//   - BuildRepositoryList: Helper to create a configured list.Model from repositories
//
// Example usage:
//
//	items := repolist.BuildRepositoryListItems(prepared)
//	repoList := repolist.BuildRepositoryList(items, width, height)
package repolist

import (
	"fmt"
	"rulem/internal/repository"

	"github.com/charmbracelet/bubbles/list"
)

// RepositoryListItem implements list.Item for displaying repositories
// in Bubble Tea list components. It wraps a PreparedRepository with
// display-friendly methods.
type RepositoryListItem struct {
	// ID is the unique repository identifier (e.g., "personal-rules-1728756432")
	ID string

	// Name is the user-provided display name (e.g., "Personal Rules")
	Name string

	// Type is the repository type ("local" or "github")
	Type string

	// Path is the local filesystem path to the repository
	Path string
}

// Title returns the repository name for display in the list.
// This is the primary text shown for each list item.
func (i RepositoryListItem) Title() string {
	return i.Name
}

// Description returns the repository type and path for display.
// Shows an icon based on repository type followed by the path.
func (i RepositoryListItem) Description() string {
	icon := "📁" // local
	if i.Type == "github" {
		icon = "🔗" // github
	}
	return fmt.Sprintf("%s %s • %s", icon, i.Type, i.Path)
}

// FilterValue returns the combined search string for filtering.
// Includes name, type, and path for comprehensive search.
func (i RepositoryListItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s", i.Name, i.Type, i.Path)
}

// BuildRepositoryListItems converts PreparedRepository slice to RepositoryListItem slice.
// This is used to populate the repository selection list.
//
// Parameters:
//   - prepared: Slice of prepared repositories from PrepareAllRepositories
//
// Returns:
//   - []list.Item: Slice of RepositoryListItem as list.Item interface
func BuildRepositoryListItems(prepared []repository.PreparedRepository) []list.Item {
	items := make([]list.Item, len(prepared))
	for i, prep := range prepared {
		items[i] = RepositoryListItem{
			ID:   prep.ID(),
			Name: prep.Name(),
			Type: string(prep.Type()),
			Path: prep.LocalPath,
		}
	}
	return items
}

// BuildRepositoryList creates a configured list.Model for repository selection.
// The list is set up with sensible defaults for repository selection UX.
//
// Parameters:
//   - items: Slice of list.Item (typically from BuildRepositoryListItems)
//   - width: Available width for the list
//   - height: Available height for the list
//
// Returns:
//   - list.Model: Configured Bubble Tea list ready for use
func BuildRepositoryList(items []list.Item, width, height int) list.Model {
	repoList := list.New(items, list.NewDefaultDelegate(), width, height)
	repoList.Title = "" // Use layout for titles
	repoList.SetShowTitle(false)
	repoList.SetShowStatusBar(false)
	repoList.SetFilteringEnabled(true)
	repoList.SetShowHelp(false) // Use layout for help
	return repoList
}

// GetSelectedRepository returns the selected RepositoryListItem from a list along with
// the raw list.Item. This permits callers to inspect non-repository selections (e.g., action items).
// Returns (nil, nil) if nothing is selected.
//
// Parameters:
//   - l: The list.Model to get the current selection from
//
// Returns:
//   - *RepositoryListItem: The selected repository item or nil
//   - list.Item: The raw selected item for further inspection
func GetSelectedRepository(l list.Model) (*RepositoryListItem, list.Item) {
	selected := l.SelectedItem()
	if selected == nil {
		return nil, nil
	}
	if item, ok := selected.(RepositoryListItem); ok {
		return &item, selected
	}
	return nil, selected
}
