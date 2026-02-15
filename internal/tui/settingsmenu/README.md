# Settings Menu Flows

## Settings menu flows at a glance

This package implements every Settings Menu journey that the TUI exposes:

- **Main menu + repository actions** render the curated list of prepared repositories plus action items (`BuildSettingsMainMenuItems`). Selecting a repository goes to `SettingsStateRepositoryActions`, which then offers context-aware operations: manual refresh, change branch, change clone path, change the repository name, and delete (delete only allowed when more than one repo exists).
- **Add repository** starts in `SettingsStateAddRepositoryType`. Local additions prompt for name and path, while GitHub additions walk through `name → URL → branch (optional) → local clone path → optional PAT`. PAT entry only appears when the credential store is empty and is handled by `createGitHubRepositoryWithPAT`. When all validation passes, `createLocalRepository` or `createGitHubRepository` persist the new entry and enqueue the background prepare flow.
- **Edit flows** are per-field: branch and clone path edits require `checkDirtyState` before confirmation and then call `updateGitHubBranch`/`updateGitHubPath`, whereas name edits (for both types) simply validate uniqueness and save via `updateRepositoryName`. Each flow has matching confirm and error states as described below.
- **Manual refresh and delete** first prompt the user, run flow-specific dirty checks (delete only for GitHub repos), then invoke `triggerRefresh` or `deleteRepository`. Errors are presented in dedicated error states before returning to the main menu.
- **Update GitHub PAT** is a global flow triggered by the PAT action item. It stores the token that every GitHub repository shares and also powers the optional PAT entrance in the add flow.

## Architectural decision: mutually exclusive states

Only `SettingsStateMainMenu` and `SettingsStateComplete` are shared between flows. Every other flow owns its own set of states for input prompts, confirmations, and error screens (e.g., `SettingsStateAddLocalPath`, `SettingsStateEditBranchConfirm`, `SettingsStateEditBranchError`). This keeps `Update()` short, eliminates `switch` statements based on prior state, and makes each flow independently testable.

Dirty-state checks inject flow-specific messages (`editBranchDirtyStateMsg`, `refreshDirtyStateMsg`, `deleteDirtyStateMsg`, `editClonePathDirtyStateMsg`) so that each flow reacts without inspecting global state. Any destructive GitHub operation relies on this pattern before saving, refreshing, or deleting.

## Flow highlights with handlers

### Main menu + actions

- `handleMainMenuKeys` navigates the list, `viewMainMenu` renders repositories plus action items, and `loadCurrentConfig` refreshes the list after persistence.
- Action items branch to Add repository (`SettingsStateAddRepositoryType`) or Update PAT (`SettingsStateUpdateGitHubPAT`).
- Selecting a repository stores `m.selectedRepositoryID`, populates `m.changeType`, and switches to `SettingsStateRepositoryActions`.

### Add repository

- `handleAddRepositoryTypeKeys` routes to `handleAddLocalNameKeys` or `handleAddGitHubNameKeys`.
- Local flow: name validation (`ValidateRepositoryName`) in `handleAddLocalNameKeys`, path validation in `handleAddLocalPathKeys`/`validateAndProceedLocalPath`, then `createLocalRepository` persists the new entry.
- GitHub flow: sequential handlers for name, URL, branch, path, and optionally PAT. `createGitHubRepository` checks for an existing PAT; if missing it yields `addGitHubPATNeededMsg` so `handleAddGitHubPATKeys` can collect and validate the token before calling `createGitHubRepositoryWithPAT`.

### Edit flows

- Branch edits: `handleUpdateGitHubBranchKeys`, `checkDirtyState(editBranchDirtyStateMsg factory)`, `handleEditBranchConfirmKeys`, `updateGitHubBranch` (which validates remote branch existence and triggers `source.FetchUpdates`).
- Clone path edits: `handleUpdateGitHubPathKeys`, the same dirty check pattern, `handleEditClonePathConfirmKeys`, and `updateGitHubPath` with writable path validation.
- Name edits: `handleUpdateRepoNameKeys`, `handleEditNameConfirmKeys`, `updateRepositoryName` ensure uniqueness and length constraints without dirty checks.

### Manual refresh and delete

- Manual refresh uses `handleManualRefreshKeys` and `checkDirtyState(refreshDirtyStateMsg factory)` before `triggerRefresh`; while the refresh runs the UI sits in `SettingsStateRefreshInProgress` until `refreshCompleteMsg` returns.
- Delete starts at `handleConfirmDeleteKeys`, conditional dirty checks for GitHub repos, and `deleteRepository` (which removes the entry, persists config, and refreshes the prepared list).
- Each path flows through dedicated error states (`SettingsStateRefreshError`, `SettingsStateDeleteError`) that pass their error to `layout.SetError` before returning to the main menu.

### PAT management

- `SettingsStateUpdateGitHubPAT` is the only place where users can update or remove a PAT outside of the add flow. Tokens are stored through `credManager.StoreGitHubToken`, and all GitHub repo operations rely on the same stored token.
- The optional PAT step during GitHub repo addition reuses the same validation routines so the token is valid for all configured repos.

## Testing and implementation reminders

- Any test that persists config must call `helpers.SetTestConfigPath(t)` so it operates in a temporary directory; this prevents accidental changes to the real config.
- Stubbed persistence hooks (`createLocalRepository`, `createGitHubRepository`, `deleteRepository`) must each persist `m.currentConfig`, rebuild prepared repositories, and trigger the background prepare/scan flow rather than blocking the UI.
- Error states should contain actionable messages (e.g., name/path conflicts, dirty-repo notices) so users can recover instead of seeing generic failures.
