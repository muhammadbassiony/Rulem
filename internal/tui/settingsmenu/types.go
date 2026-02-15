// Package settingsmenu provides the settings modification flow for the rulem TUI application.
package settingsmenu

// State Definitions
// Architecture: Mutually Exclusive States
// Each flow has its own dedicated states to prevent state pollution and ensure
// clear separation of concerns. States are grouped by flow for better organization.

// SettingsState represents the current state of the settings flow using mutually exclusive states.
// Each flow (Add Local, Add GitHub, Delete, Edit, etc.) has its own set of states.
type SettingsState int

const (
	// Shared States (2)
	// These states are used across all flows

	// SettingsStateMainMenu displays the list of repositories and options to add new ones
	SettingsStateMainMenu SettingsState = iota
	// SettingsStateComplete indicates successful completion of any operation
	SettingsStateComplete

	// Add Local Repository Flow
	// Flow: AddRepositoryType → AddLocalName → AddLocalPath → [AddLocalError | Complete]

	// SettingsStateAddRepositoryType prompts user to choose between Local and GitHub repository
	SettingsStateAddRepositoryType // TODO, why part of this flow?
	// SettingsStateAddLocalName prompts for the local repository name
	SettingsStateAddLocalName
	// SettingsStateAddLocalPath prompts for the local repository directory path
	SettingsStateAddLocalPath
	// SettingsStateAddLocalError displays error during local repository creation
	SettingsStateAddLocalError

	// Add GitHub Repository Flow (5 states)
	// Flow: AddRepositoryType → AddGitHubName → AddGitHubURL → AddGitHubBranch → AddGitHubPath → [AddGitHubError | Complete]

	// SettingsStateAddGitHubName prompts for the GitHub repository name
	SettingsStateAddGitHubName
	// SettingsStateAddGitHubURL prompts for the GitHub repository URL
	SettingsStateAddGitHubURL
	// SettingsStateAddGitHubBranch prompts for the GitHub branch to track
	SettingsStateAddGitHubBranch
	// SettingsStateAddGitHubPath prompts for the local clone directory path
	SettingsStateAddGitHubPath
	// SettingsStateAddGitHubPAT prompts for GitHub PAT when none exists (optional state)
	// This is an optional flow state - only entered when PAT is missing during Add GitHub flow
	SettingsStateAddGitHubPAT
	// SettingsStateAddGitHubError displays error during GitHub repository creation
	SettingsStateAddGitHubError

	// Delete Repository Flow (3 states)
	// Flow: RepositoryActions → ConfirmDelete → [DeleteError | Complete]

	// SettingsStateRepositoryActions shows available actions for the selected repository
	SettingsStateRepositoryActions
	// SettingsStateConfirmDelete prompts for confirmation before deleting a repository
	SettingsStateConfirmDelete
	// SettingsStateDeleteError displays error during repository deletion
	SettingsStateDeleteError

	// Edit Branch Flow (3 states)
	// Flow: UpdateGitHubBranch → EditBranchConfirm → [EditBranchError | Complete]

	// SettingsStateUpdateGitHubBranch prompts for new GitHub branch name
	SettingsStateUpdateGitHubBranch
	// SettingsStateEditBranchConfirm displays confirmation for branch change
	SettingsStateEditBranchConfirm
	// SettingsStateEditBranchError displays error during branch update
	SettingsStateEditBranchError

	// Edit Clone Path Flow (3 states)
	// Flow: UpdateGitHubPath → EditClonePathConfirm → [EditClonePathError | Complete]
	// Also used for local repository path editing

	// SettingsStateUpdateGitHubPath prompts for new clone directory path
	SettingsStateUpdateGitHubPath
	// SettingsStateEditClonePathConfirm displays confirmation for path change
	SettingsStateEditClonePathConfirm
	// SettingsStateEditClonePathError displays error during path update
	SettingsStateEditClonePathError

	// Edit Repository Name Flow (3 states)
	// Flow: UpdateRepoName → EditNameConfirm → [EditNameError | Complete]

	// SettingsStateUpdateRepoName prompts for new repository name
	SettingsStateUpdateRepoName
	// SettingsStateEditNameConfirm displays confirmation for name change
	SettingsStateEditNameConfirm
	// SettingsStateEditNameError displays error during name update
	SettingsStateEditNameError

	// Manual Refresh Flow (3 states)
	// Flow: ManualRefresh → RefreshInProgress → [RefreshError | Complete]

	// SettingsStateManualRefresh prompts for confirmation before refreshing from GitHub
	SettingsStateManualRefresh
	// SettingsStateRefreshInProgress shows progress indicator during refresh operation
	SettingsStateRefreshInProgress
	// SettingsStateRefreshError displays error during manual refresh
	SettingsStateRefreshError

	// Update PAT Flow (3 states)
	// Flow: UpdateGitHubPAT → UpdatePATConfirm → [UpdatePATError | Complete]

	// SettingsStateUpdateGitHubPAT prompts for new GitHub Personal Access Token
	SettingsStateUpdateGitHubPAT
	// SettingsStateUpdatePATConfirm displays confirmation for PAT update
	SettingsStateUpdatePATConfirm
	// SettingsStateUpdatePATError displays error during PAT update
	SettingsStateUpdatePATError
)

// String returns a human-readable name for the state, useful for debugging and logging.
func (s SettingsState) String() string {
	switch s {
	// Shared states
	case SettingsStateMainMenu:
		return "MainMenu"
	case SettingsStateComplete:
		return "Complete"

	// Add Local Repository flow
	case SettingsStateAddRepositoryType:
		return "AddRepositoryType"
	case SettingsStateAddLocalName:
		return "AddLocalName"
	case SettingsStateAddLocalPath:
		return "AddLocalPath"
	case SettingsStateAddLocalError:
		return "AddLocalError"

	// Add GitHub Repository flow
	case SettingsStateAddGitHubName:
		return "AddGitHubName"
	case SettingsStateAddGitHubURL:
		return "AddGitHubURL"
	case SettingsStateAddGitHubBranch:
		return "AddGitHubBranch"
	case SettingsStateAddGitHubPath:
		return "AddGitHubPath"
	case SettingsStateAddGitHubPAT:
		return "AddGitHubPAT"
	case SettingsStateAddGitHubError:
		return "AddGitHubError"

	// Delete Repository flow
	case SettingsStateRepositoryActions:
		return "RepositoryActions"
	case SettingsStateConfirmDelete:
		return "ConfirmDelete"
	case SettingsStateDeleteError:
		return "DeleteError"

	// Edit Branch flow
	case SettingsStateUpdateGitHubBranch:
		return "UpdateGitHubBranch"
	case SettingsStateEditBranchConfirm:
		return "EditBranchConfirm"
	case SettingsStateEditBranchError:
		return "EditBranchError"

	// Edit Clone Path flow
	case SettingsStateUpdateGitHubPath:
		return "UpdateGitHubPath"
	case SettingsStateEditClonePathConfirm:
		return "EditClonePathConfirm"
	case SettingsStateEditClonePathError:
		return "EditClonePathError"

	// Edit Repository Name flow
	case SettingsStateUpdateRepoName:
		return "UpdateRepoName"
	case SettingsStateEditNameConfirm:
		return "EditNameConfirm"
	case SettingsStateEditNameError:
		return "EditNameError"

	// Manual Refresh flow
	case SettingsStateManualRefresh:
		return "ManualRefresh"
	case SettingsStateRefreshInProgress:
		return "RefreshInProgress"
	case SettingsStateRefreshError:
		return "RefreshError"

	// Update PAT flow
	case SettingsStateUpdateGitHubPAT:
		return "UpdateGitHubPAT"
	case SettingsStateUpdatePATConfirm:
		return "UpdatePATConfirm"
	case SettingsStateUpdatePATError:
		return "UpdatePATError"

	default:
		return "Unknown"
	}
}

// Message Types

// settingsCompleteMsg signals successful completion of any settings operation.
// Transitions model to SettingsStateComplete and triggers config reload.
type settingsCompleteMsg struct{}

// refreshInitiateMsg signals the start of a manual refresh operation.
// Sent when user confirms manual refresh from GitHub.
type refreshInitiateMsg struct{}

// refreshInProgressMsg indicates refresh operation is currently running.
// Used to update UI during async refresh operations.
type refreshInProgressMsg struct{}

// refreshCompleteMsg signals completion of manual refresh operation.
// Contains the outcome of the refresh attempt.
type refreshCompleteMsg struct {
	success bool
	err     error
}

// Flow-Specific Dirty State Check Messages
// Each flow that performs destructive GitHub operations has its own dirty state message.
// This eliminates complex conditional logic in the Update handler and maintains
// the mutually exclusive states architecture.

// editBranchDirtyStateMsg reports dirty state check result for branch editing flow.
// If isDirty=true, transitions to SettingsStateEditBranchError.
// If isDirty=false, proceeds to SettingsStateEditBranchConfirm.
type editBranchDirtyStateMsg struct {
	isDirty bool  // true if repository has uncommitted changes
	err     error // error from dirty state check, if any
}

// refreshDirtyStateMsg reports dirty state check result for manual refresh flow.
// If isDirty=true, transitions to SettingsStateRefreshError.
// If isDirty=false, proceeds with triggerRefresh().
type refreshDirtyStateMsg struct {
	isDirty bool  // true if repository has uncommitted changes
	err     error // error from dirty state check, if any
}

// deleteDirtyStateMsg reports dirty state check result for delete flow.
// If isDirty=true, transitions to SettingsStateDeleteError.
// If isDirty=false, proceeds with deleteRepository().
type deleteDirtyStateMsg struct {
	isDirty bool  // true if repository has uncommitted changes
	err     error // error from dirty state check, if any
}

// editClonePathDirtyStateMsg reports dirty state check result for clone path editing flow.
// If isDirty=true, transitions to SettingsStateEditClonePathError.
// If isDirty=false, proceeds to SettingsStateEditClonePathConfirm.
type editClonePathDirtyStateMsg struct {
	isDirty bool  // true if repository has uncommitted changes
	err     error // error from dirty state check, if any
}

// Flow-Specific Error Messages
// Each flow has its own error message type for better error handling and state management

// addLocalErrorMsg signals an error during local repository creation.
// Transitions to SettingsStateAddLocalError.
type addLocalErrorMsg struct{ err error }

// addGitHubErrorMsg signals an error during GitHub repository creation.
// Transitions to SettingsStateAddGitHubError.
type addGitHubErrorMsg struct{ err error }

// deleteErrorMsg signals an error during repository deletion.
// Transitions to SettingsStateDeleteError.
type deleteErrorMsg struct{ err error }

// editBranchErrorMsg signals an error during branch update.
// Transitions to SettingsStateEditBranchError.
type editBranchErrorMsg struct{ err error }

// editClonePathErrorMsg signals an error during clone path update.
// Transitions to SettingsStateEditClonePathError.
type editClonePathErrorMsg struct{ err error }

// editNameErrorMsg signals an error during repository name update.
// Transitions to SettingsStateEditNameError.
type editNameErrorMsg struct{ err error }

// updatePATErrorMsg signals an error during PAT update.
// Transitions to SettingsStateUpdatePATError.
type updatePATErrorMsg struct{ err error }

// addGitHubPATNeededMsg signals that PAT is required to complete GitHub repository creation.
// This is an optional flow message - only sent when PAT is missing during Add GitHub flow.
// Transitions to SettingsStateAddGitHubPAT to allow inline PAT entry.
type addGitHubPATNeededMsg struct{}

// Enums

// ChangeOption represents the type of change or action available for a repository.
// Used to build context menus and determine available operations.
type ChangeOption int

const (
	// ChangeOptionManualRefresh triggers a manual sync from GitHub
	ChangeOptionManualRefresh ChangeOption = iota
	// ChangeOptionGitHubBranch allows editing the tracked GitHub branch
	ChangeOptionGitHubBranch
	// ChangeOptionGitHubPath allows editing the local clone directory path
	ChangeOptionGitHubPath
	// ChangeOptionChangeRepoName allows editing the repository name
	ChangeOptionChangeRepoName
	// ChangeOptionDelete removes the repository from configuration
	ChangeOptionDelete
	// ChangeOptionAddNewRepository starts the add repository flow (type selection)
	ChangeOptionAddNewRepository
	// ChangeOptionGitHubPAT updates or removes the GitHub Personal Access Token (global, not per-repo)
	ChangeOptionGitHubPAT
	// ChangeOptionBack returns to the previous menu
	ChangeOptionBack
)

// ChangeOptionInfo contains display information for a change option.
// Used to render menu items with title, description, and associated action.
type ChangeOptionInfo struct {
	Title       string       // Display title shown in menu
	Description string       // Detailed description of the action
	Option      ChangeOption // Associated action to perform
}
