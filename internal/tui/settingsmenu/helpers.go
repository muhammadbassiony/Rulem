package settingsmenu

import (
	"rulem/internal/repository"
	"rulem/internal/tui/helpers/repolist"

	"github.com/charmbracelet/bubbles/list"
)

// SettingsActionListItem implements list.Item for action items in settings menu.
// These are special items like "Add Repository" or "Update GitHub PAT" that appear
// alongside repository entries in the main settings menu.
type SettingsActionListItem struct {
	// Action is the ChangeOption type for this action
	Action ChangeOption

	// DisplayTitle is the main display text
	DisplayTitle string

	// DisplayDescription is the description text
	DisplayDescription string
}

// Title returns the action title
func (i SettingsActionListItem) Title() string {
	return i.DisplayTitle
}

// Description returns the action description
func (i SettingsActionListItem) Description() string {
	return i.DisplayDescription
}

// FilterValue returns the search string for filtering
func (i SettingsActionListItem) FilterValue() string {
	return i.DisplayTitle + " " + i.DisplayDescription
}

// BuildSettingsMainMenuItems builds the main settings menu list items.
// This includes all repositories plus action items for adding new repositories
// and updating GitHub PAT.
//
// Parameters:
//   - prepared: Slice of prepared repositories
//
// Returns:
//   - []list.Item: Complete list including repositories and action items
func BuildSettingsMainMenuItems(prepared []repository.PreparedRepository) []list.Item {
	// Start with repository items
	items := repolist.BuildRepositoryListItems(prepared)

	// Add action items
	items = append(items, SettingsActionListItem{
		Action:             ChangeOptionAddNewRepository,
		DisplayTitle:       "➕ Add New Repository",
		DisplayDescription: "Add a local or GitHub repository",
	})

	items = append(items, SettingsActionListItem{
		Action:             ChangeOptionGitHubPAT,
		DisplayTitle:       "🔑 Update GitHub PAT",
		DisplayDescription: "Update Personal Access Token for GitHub repositories",
	})

	return items
}

// GetSelectedSettingsAction returns the selected SettingsActionListItem from a list.
// Returns nil if nothing is selected or selection is not a SettingsActionListItem.
//
// Parameters:
//   - l: The list.Model to get selection from
//
// Returns:
//   - *SettingsActionListItem: The selected action item or nil
func GetSelectedSettingsAction(l list.Model) *SettingsActionListItem {
	_, selected := repolist.GetSelectedRepository(l)
	if selected == nil {
		return nil
	}
	if item, ok := selected.(SettingsActionListItem); ok {
		return &item
	}
	return nil
}

// IsSettingsActionItem checks if the selected item is an action item
func IsSettingsActionItem(l list.Model) bool {
	return GetSelectedSettingsAction(l) != nil
}
