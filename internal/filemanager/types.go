package filemanager

import (
	"fmt"
	"strings"
)

// FileItem is a rule file discovered by a scan, in a shape bubbles' List model
// can render. It includes repository metadata for multi-repository support.
//
// A FileItem carries a (directory, relative path) pair rather than a bare
// absolute path. RelPath is the half that is used to *act* on the file: it is
// meaningful only against the directory it was scanned from, which is exactly
// the property that makes it safe to pass around. Path is the same file
// rendered absolutely so it can be shown to a user and searched over; it must
// not be reopened directly.
type FileItem struct {
	Name    string // Base filename for display
	RelPath string // Path relative to the repository / scan root - use this to address the file
	Path    string // Absolute filesystem path - DISPLAY AND FILTERING ONLY

	// Repository metadata (for multi-repository support)
	RepositoryID   string // Links to RepositoryEntry.ID (e.g., "personal-rules-1728756432")
	RepositoryName string // Denormalized for display (e.g., "Personal Rules")
	RepositoryType string // "local" or "github" (for styling/icons)
}

// Title returns the file name for display in bubble tea list
func (i FileItem) Title() string {
	return i.Name
}

// Description returns repository information for display in bubble tea list
// Shows the repository name with an icon based on repository type
func (i FileItem) Description() string {
	if i.RepositoryName != "" {
		icon := "📁"
		if i.RepositoryType == "github" {
			icon = "🔗"
		}
		return fmt.Sprintf("%s %s", icon, i.RepositoryName)
	}
	return " "
}

// FilterValue returns the combined search string for bubble tea filtering
// Includes file name, path, and repository name for comprehensive search
func (i FileItem) FilterValue() string {
	parts := []string{i.Name, i.Path}
	if i.RepositoryName != "" {
		parts = append(parts, i.RepositoryName)
	}
	return strings.Join(parts, " ")
}
