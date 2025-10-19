package filemanager

import (
	"fmt"
	"strings"
)

// File item that is compatible with bubble's List model
// Includes repository metadata for multi-repository support
type FileItem struct {
	Name string // Base filename for display
	Path string // Absolute filesystem path (validated during scan)

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
