package repolist

import (
	"rulem/internal/repository"
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

// TestBuildRepositoryListItems tests conversion of PreparedRepository slice to list items.
func TestBuildRepositoryListItems(t *testing.T) {
	tests := []struct {
		name     string
		prepared []repository.PreparedRepository
		want     int
	}{
		{
			name:     "empty list",
			prepared: []repository.PreparedRepository{},
			want:     0,
		},
		{
			name: "single repository",
			prepared: []repository.PreparedRepository{
				{
					Entry: repository.RepositoryEntry{
						ID:   "test-repo-123",
						Name: "Test Repo",
						Type: repository.RepositoryTypeLocal,
					},
					LocalPath: "/path/to/repo",
				},
			},
			want: 1,
		},
		{
			name: "multiple repositories",
			prepared: []repository.PreparedRepository{
				{
					Entry: repository.RepositoryEntry{
						ID:   "local-repo-123",
						Name: "Local Repo",
						Type: repository.RepositoryTypeLocal,
					},
					LocalPath: "/path/to/local",
				},
				{
					Entry: repository.RepositoryEntry{
						ID:   "github-repo-456",
						Name: "GitHub Repo",
						Type: repository.RepositoryTypeGitHub,
					},
					LocalPath: "/path/to/github",
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := BuildRepositoryListItems(tt.prepared)
			if len(items) != tt.want {
				t.Errorf("BuildRepositoryListItems() returned %d items, want %d", len(items), tt.want)
			}

			for i, item := range items {
				repoItem, ok := item.(RepositoryListItem)
				if !ok {
					t.Errorf("Item %d is not a RepositoryListItem", i)
					continue
				}

				prep := tt.prepared[i]
				if repoItem.ID != prep.ID() {
					t.Errorf("Item %d ID = %s, want %s", i, repoItem.ID, prep.ID())
				}
				if repoItem.Name != prep.Name() {
					t.Errorf("Item %d Name = %s, want %s", i, repoItem.Name, prep.Name())
				}
				if repoItem.Path != prep.LocalPath {
					t.Errorf("Item %d Path = %s, want %s", i, repoItem.Path, prep.LocalPath)
				}
			}
		})
	}
}

// TestRepositoryListItem_Title tests the Title method.
func TestRepositoryListItem_Title(t *testing.T) {
	item := RepositoryListItem{
		ID:   "test-123",
		Name: "My Repository",
		Type: "local",
		Path: "/path/to/repo",
	}

	if got := item.Title(); got != "My Repository" {
		t.Errorf("Title() = %s, want %s", got, "My Repository")
	}
}

// TestRepositoryListItem_Description tests the Description method.
func TestRepositoryListItem_Description(t *testing.T) {
	tests := []struct {
		name string
		item RepositoryListItem
	}{
		{
			name: "local repository",
			item: RepositoryListItem{
				Type: "local",
				Path: "/path/to/repo",
			},
		},
		{
			name: "github repository",
			item: RepositoryListItem{
				Type: "github",
				Path: "/path/to/repo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.item.Description()
			if desc == "" {
				t.Error("Description() returned empty string")
			}
		})
	}
}

// TestRepositoryListItem_FilterValue tests the FilterValue method.
func TestRepositoryListItem_FilterValue(t *testing.T) {
	item := RepositoryListItem{
		ID:   "test-123",
		Name: "My Repository",
		Type: "local",
		Path: "/path/to/repo",
	}

	filterVal := item.FilterValue()
	if filterVal == "" {
		t.Error("FilterValue() returned empty string")
	}
}

// TestBuildRepositoryList tests the list.Model builder.
func TestBuildRepositoryList(t *testing.T) {
	items := BuildRepositoryListItems([]repository.PreparedRepository{
		{
			Entry: repository.RepositoryEntry{
				ID:   "test-repo-123",
				Name: "Test Repo",
				Type: repository.RepositoryTypeLocal,
			},
			LocalPath: "/path/to/repo",
		},
	})

	list := BuildRepositoryList(items, 80, 24)
	if list.FilteringEnabled() != true {
		t.Error("List should have filtering enabled")
	}
}

// TestGetSelectedRepository tests the selection helper.
func TestGetSelectedRepository(t *testing.T) {
	items := BuildRepositoryListItems([]repository.PreparedRepository{
		{
			Entry: repository.RepositoryEntry{
				ID:   "test-repo-123",
				Name: "Test Repo",
				Type: repository.RepositoryTypeLocal,
			},
			LocalPath: "/path/to/repo",
		},
		{
			Entry: repository.RepositoryEntry{
				ID:   "another-repo-456",
				Name: "Another Repo",
				Type: repository.RepositoryTypeGitHub,
			},
			LocalPath: "/path/to/another",
		},
	})

	list := BuildRepositoryList(items, 80, 24)
	selected, _ := GetSelectedRepository(list)
	if selected == nil {
		t.Fatal("GetSelectedRepository() returned nil for non-empty list")
	}

	if selected.ID != "test-repo-123" {
		t.Errorf("Selected ID = %s, want test-repo-123", selected.ID)
	}
}

// TestGetSelectedRepository_EmptyList tests selection from empty list.
func TestGetSelectedRepository_EmptyList(t *testing.T) {
	list := BuildRepositoryList([]list.Item{}, 80, 24)

	selected, _ := GetSelectedRepository(list)
	if selected != nil {
		t.Error("GetSelectedRepository() should return nil for empty list")
	}
}
