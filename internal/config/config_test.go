package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"rulem/internal/repository"
	"strings"
	"testing"
	"time"
)

func TestConfigPath(t *testing.T) {
	t.Log("Testing ConfigPath using go-app-paths")

	// Test that ConfigPaths returns valid paths
	primary, err := Path()
	fmt.Println(primary)
	if err != nil {
		t.Fatalf("Failed to get config path: %s", err)
	}

	// Both paths should be non-empty
	if primary == "" {
		t.Error("Primary config path should not be empty")
	}

	// Path should be absolute or relative
	if !filepath.IsAbs(primary) && !strings.HasPrefix(primary, ".") {
		t.Errorf("Primary path should be absolute or relative, got: %s", primary)
	}

	// The path should contain "rulem"
	if !strings.Contains(primary, "rulem") {
		t.Errorf("Primary path should contain 'rulem', got: %s", primary)
	}

	// Primary should end with config.yaml
	if !strings.HasSuffix(primary, "config.yaml") {
		t.Errorf("Primary path should end with 'config.yaml', got: %s", primary)
	}

	t.Logf("Primary config path: %s", primary)
}

func TestRepositoryEntryValidation(t *testing.T) {
	t.Log("Testing RepositoryEntry validation")

	tests := []struct {
		name    string
		entry   repository.RepositoryEntry
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid local repository entry",
			entry: repository.RepositoryEntry{
				ID:        "personal-rules-1728756432",
				Name:      "Personal Rules",
				CreatedAt: 1728756432,
				Central: repository.CentralRepositoryConfig{
					Path: "/tmp/rules",
				},
			},
			wantErr: false,
		},
		{
			name: "valid github repository entry",
			entry: repository.RepositoryEntry{
				ID:        "work-repo-1728756500",
				Name:      "Work Repo",
				CreatedAt: 1728756500,
				Central: repository.CentralRepositoryConfig{
					Path: "/tmp/work-rules",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid ID format - no timestamp",
			entry: repository.RepositoryEntry{
				ID:        "personal-rules",
				Name:      "Personal Rules",
				CreatedAt: 1728756432,
				Central: repository.CentralRepositoryConfig{
					Path: "/tmp/rules",
				},
			},
			wantErr: true,
			errMsg:  "invalid ID format",
		},
		{
			name: "empty name",
			entry: repository.RepositoryEntry{
				ID:        "test-1728756432",
				Name:      "",
				CreatedAt: 1728756432,
				Central: repository.CentralRepositoryConfig{
					Path: "/tmp/rules",
				},
			},
			wantErr: true,
			errMsg:  "name must be non-empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test ID format validation
			idPattern := regexp.MustCompile(`^[a-z0-9-]+-\d+$`)
			if !idPattern.MatchString(tt.entry.ID) && !tt.wantErr {
				t.Errorf("ID %s doesn't match expected format", tt.entry.ID)
			}

			// Test name validation
			if strings.TrimSpace(tt.entry.Name) == "" && !tt.wantErr {
				t.Error("Name should not be empty for valid entry")
			}

			// Test CreatedAt validation
			if tt.entry.CreatedAt <= 0 && !tt.wantErr {
				t.Error("CreatedAt should be positive for valid entry")
			}
		})
	}
}

func TestConfigWithMultipleRepositories(t *testing.T) {
	t.Log("Testing Config with multiple repository entries")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create config with multiple repositories
	cfg := Config{
		Version:  "1.0",
		InitTime: time.Now().Unix(),
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "personal-rules-1728756432",
				Name:      "Personal Rules",
				CreatedAt: 1728756432,
				Central: repository.CentralRepositoryConfig{
					Path: "/tmp/personal-rules",
				},
			},
			{
				ID:        "work-repo-1728756500",
				Name:      "Work Repo",
				CreatedAt: 1728756500,
				Central: repository.CentralRepositoryConfig{
					Path: "/tmp/work-rules",
				},
			},
		},
	}

	// Save and reload
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify repositories count
	if len(loaded.Repositories) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(loaded.Repositories))
	}

	// Verify first repository
	if loaded.Repositories[0].ID != "personal-rules-1728756432" {
		t.Errorf("Expected first repo ID 'personal-rules-1728756432', got %s", loaded.Repositories[0].ID)
	}
	if loaded.Repositories[0].Name != "Personal Rules" {
		t.Errorf("Expected first repo name 'Personal Rules', got %s", loaded.Repositories[0].Name)
	}

	// Verify second repository
	if loaded.Repositories[1].ID != "work-repo-1728756500" {
		t.Errorf("Expected second repo ID 'work-repo-1728756500', got %s", loaded.Repositories[1].ID)
	}
}

func TestConfigWithEmptyRepositories(t *testing.T) {
	t.Log("Testing Config with empty repositories array")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create config with no repositories (valid state)
	cfg := Config{
		Version:      "1.0",
		InitTime:     time.Now().Unix(),
		Repositories: []repository.RepositoryEntry{},
	}

	// Save and reload
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify empty array
	if len(loaded.Repositories) != 0 {
		t.Errorf("Expected 0 repositories, got %d", len(loaded.Repositories))
	}
}

func TestConfigInitTime(t *testing.T) {
	t.Log("Testing Config InitTime on Save")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	config := Config{
		Version:      "1.0",
		Repositories: []repository.RepositoryEntry{},
		// InitTime not set (0)
	}

	before := time.Now().Unix()
	if err := config.SaveTo(configPath); err != nil {
		t.Fatalf("Failed to save config: %s", err)
	}
	after := time.Now().Unix()

	// InitTime should be set during save
	if config.InitTime < before || config.InitTime > after {
		t.Errorf("InitTime %d should be between %d and %d", config.InitTime, before, after)
	}
}

func TestConfigFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	config := DefaultConfig()
	if err := config.SaveTo(configPath); err != nil {
		t.Fatalf("Failed to save config: %s", err)
	}

	// Check file permissions
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %s", err)
	}

	mode := fileInfo.Mode()
	if mode&0077 != 0 {
		t.Errorf("Config file should not be readable by group/others, got mode %o", mode)
	}
}

func TestFindRepositoryByID(t *testing.T) {
	t.Log("Testing FindRepositoryByID")

	cfg := Config{
		Version:  "1.0",
		InitTime: time.Now().Unix(),
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "repo1-123456",
				Name:      "Repo 1",
				CreatedAt: 123456,
			},
			{
				ID:        "repo2-789012",
				Name:      "Repo 2",
				CreatedAt: 789012,
			},
		},
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"existing repository", "repo1-123456", false},
		{"another existing repository", "repo2-789012", false},
		{"non-existent repository", "repo3-999999", true},
		{"empty list", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := cfg.FindRepositoryByID(tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if repo == nil {
					t.Fatal("Expected repository but got nil")
				}
				if repo.ID != tt.id {
					t.Errorf("Expected ID %s, got %s", tt.id, repo.ID)
				}
			}
		})
	}
}

func TestFindRepositoryByName(t *testing.T) {
	t.Log("Testing FindRepositoryByName (case-insensitive)")

	cfg := Config{
		Version:  "1.0",
		InitTime: time.Now().Unix(),
		Repositories: []repository.RepositoryEntry{
			{
				ID:        "personal-rules-123456",
				Name:      "Personal Rules",
				CreatedAt: 123456,
			},
			{
				ID:        "work-repo-789012",
				Name:      "Work Repo",
				CreatedAt: 789012,
			},
		},
	}

	tests := []struct {
		name       string
		searchName string
		wantID     string
		wantErr    bool
	}{
		{"exact match", "Personal Rules", "personal-rules-123456", false},
		{"case insensitive - lowercase", "personal rules", "personal-rules-123456", false},
		{"case insensitive - uppercase", "PERSONAL RULES", "personal-rules-123456", false},
		{"case insensitive - mixed", "PeRsOnAl RuLeS", "personal-rules-123456", false},
		{"another repo", "Work Repo", "work-repo-789012", false},
		{"non-existent", "Non Existent Repo", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := cfg.FindRepositoryByName(tt.searchName)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if repo == nil {
					t.Fatal("Expected repository but got nil")
				}
				if repo.ID != tt.wantID {
					t.Errorf("Expected ID %s, got %s", tt.wantID, repo.ID)
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Version == "" {
		t.Error("Default config should have a version")
	}

	if len(config.Repositories) != 0 {
		t.Errorf("Default config should have empty repositories array, got %d", len(config.Repositories))
	}

	if config.InitTime != 0 {
		t.Error("Default config InitTime should be 0 (will be set on save)")
	}
}

func TestConfigPathEnvironmentOverride(t *testing.T) {
	t.Log("Testing ConfigPath environment variable override")

	// Save original environment
	originalPath := os.Getenv("RULEM_CONFIG_PATH")
	defer func() {
		if originalPath == "" {
			os.Unsetenv("RULEM_CONFIG_PATH")
		} else {
			os.Setenv("RULEM_CONFIG_PATH", originalPath)
		}
	}()

	// Test with environment variable set
	testPath := "/tmp/test-rulem-config.yaml"
	err := os.Setenv("RULEM_CONFIG_PATH", testPath)
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}

	configPath, err := Path()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	if configPath != testPath {
		t.Errorf("Expected config path %s, got %s", testPath, configPath)
	}

	// Test with environment variable unset
	os.Unsetenv("RULEM_CONFIG_PATH")
	configPath, err = Path()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	if configPath == testPath {
		t.Error("Config path should not match test path when environment variable is unset")
	}

	if !strings.Contains(configPath, "rulem") {
		t.Errorf("Config path should contain 'rulem', got: %s", configPath)
	}
}

// Error handling tests
func TestConfigErrorHandling(t *testing.T) {
	t.Run("load non-existent file", func(t *testing.T) {
		_, err := LoadFrom("/non/existent/file.yaml")
		if err == nil {
			t.Error("Should error when loading non-existent file")
		}
	})

	t.Run("load invalid YAML", func(t *testing.T) {
		tempDir := t.TempDir()
		invalidFile := filepath.Join(tempDir, "invalid.yaml")
		if err := os.WriteFile(invalidFile, []byte("invalid: yaml: content: ["), 0644); err != nil {
			t.Fatalf("Failed to write invalid YAML file: %v", err)
		}

		_, err := LoadFrom(invalidFile)
		if err == nil {
			t.Error("Should error when loading invalid YAML")
		}
	})

	t.Run("save to read-only directory", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test as root user")
		}

		config := DefaultConfig()
		err := config.SaveTo("/root/config.yaml")
		if err == nil {
			t.Error("Should error when saving to read-only directory")
		}
	})
}

// ID Generation Tests

func TestGenerateRepositoryID(t *testing.T) {
	t.Log("Testing GenerateRepositoryID with various inputs")

	tests := []struct {
		name           string
		repoName       string
		timestamp      int64
		expectedPrefix string // Expected sanitized name part
	}{
		{
			name:           "simple name",
			repoName:       "personal",
			timestamp:      1728756432,
			expectedPrefix: "personal",
		},
		{
			name:           "name with spaces",
			repoName:       "Personal Rules",
			timestamp:      1728756432,
			expectedPrefix: "personal-rules",
		},
		{
			name:           "name with underscores",
			repoName:       "work_project",
			timestamp:      1728756500,
			expectedPrefix: "work-project",
		},
		{
			name:           "name with special characters",
			repoName:       "My@Project#123!",
			timestamp:      1728756600,
			expectedPrefix: "my-project-123",
		},
		{
			name:           "name with multiple consecutive special chars",
			repoName:       "test___project",
			timestamp:      1728756700,
			expectedPrefix: "test-project",
		},
		{
			name:           "name with leading/trailing spaces",
			repoName:       "  Project Name  ",
			timestamp:      1728756800,
			expectedPrefix: "project-name",
		},
		{
			name:           "name with unicode characters",
			repoName:       "Проект-Rules",
			timestamp:      1728756900,
			expectedPrefix: "rules", // Unicode chars removed, dash removed, left with "rules"
		},
		{
			name:           "empty name",
			repoName:       "",
			timestamp:      1728757000,
			expectedPrefix: "repo",
		},
		{
			name:           "all special characters",
			repoName:       "!!!@@@###",
			timestamp:      1728757100,
			expectedPrefix: "repo",
		},
		{
			name:           "numbers only",
			repoName:       "12345",
			timestamp:      1728757200,
			expectedPrefix: "12345",
		},
		{
			name:           "mixed case with numbers",
			repoName:       "Project2024",
			timestamp:      1728757300,
			expectedPrefix: "project2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateRepositoryID(tt.repoName, tt.timestamp)

			// Verify format: sanitized-name-timestamp
			pattern := regexp.MustCompile(`^[a-z0-9-]+-\d+$`)
			if !pattern.MatchString(id) {
				t.Errorf("ID %q doesn't match expected format ^[a-z0-9-]+-\\d+$", id)
			}

			// Verify the ID contains the timestamp
			timestampStr := formatTimestamp(tt.timestamp)
			if !contains(id, timestampStr) {
				t.Errorf("ID %q should contain timestamp %s", id, timestampStr)
			}

			// Verify the ID starts with the expected prefix
			if !startsWith(id, tt.expectedPrefix+"-") {
				t.Errorf("ID %q should start with %q", id, tt.expectedPrefix+"-")
			}
		})
	}
}

func TestGenerateRepositoryID_Uniqueness(t *testing.T) {
	t.Log("Testing ID uniqueness with same name but different timestamps")

	name := "Test Repository"
	timestamp1 := int64(1728756432)
	timestamp2 := int64(1728756500)

	id1 := GenerateRepositoryID(name, timestamp1)
	id2 := GenerateRepositoryID(name, timestamp2)

	if id1 == id2 {
		t.Errorf("Expected different IDs for different timestamps, got: %s and %s", id1, id2)
	}

	// Both should have the same prefix
	expectedPrefix := "test-repository-"
	if !startsWith(id1, expectedPrefix) {
		t.Errorf("ID1 %q should start with %q", id1, expectedPrefix)
	}
	if !startsWith(id2, expectedPrefix) {
		t.Errorf("ID2 %q should start with %q", id2, expectedPrefix)
	}
}

func TestGenerateRepositoryID_CurrentTimestamp(t *testing.T) {
	t.Log("Testing ID generation with current timestamp")

	name := "Real Time Test"
	timestamp := time.Now().Unix()

	id := GenerateRepositoryID(name, timestamp)

	// Verify format
	pattern := regexp.MustCompile(`^[a-z0-9-]+-\d+$`)
	if !pattern.MatchString(id) {
		t.Errorf("ID %q doesn't match expected format", id)
	}

	// Verify it contains current timestamp
	timestampStr := formatTimestamp(timestamp)
	if !contains(id, timestampStr) {
		t.Errorf("ID %q should contain timestamp %s", id, timestampStr)
	}
}

func TestSanitizeNameForID(t *testing.T) {
	t.Log("Testing sanitizeNameForID helper function")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lowercase name",
			input:    "personal",
			expected: "personal",
		},
		{
			name:     "simple uppercase name",
			input:    "PERSONAL",
			expected: "personal",
		},
		{
			name:     "mixed case",
			input:    "PersonalRules",
			expected: "personalrules",
		},
		{
			name:     "spaces to dashes",
			input:    "Personal Rules",
			expected: "personal-rules",
		},
		{
			name:     "underscores to dashes",
			input:    "work_project",
			expected: "work-project",
		},
		{
			name:     "special characters removed",
			input:    "My@Project#123!",
			expected: "my-project-123",
		},
		{
			name:     "multiple consecutive spaces",
			input:    "test   project",
			expected: "test-project",
		},
		{
			name:     "leading spaces",
			input:    "  project",
			expected: "project",
		},
		{
			name:     "trailing spaces",
			input:    "project  ",
			expected: "project",
		},
		{
			name:     "leading and trailing special chars",
			input:    "___project___",
			expected: "project",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "repo",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "repo",
		},
		{
			name:     "special characters only",
			input:    "!!!@@@###",
			expected: "repo",
		},
		{
			name:     "unicode characters",
			input:    "Проект",
			expected: "repo",
		},
		{
			name:     "mixed unicode and ascii",
			input:    "Test-Проект-2024",
			expected: "test-2024",
		},
		{
			name:     "numbers only",
			input:    "12345",
			expected: "12345",
		},
		{
			name:     "very long name",
			input:    "this-is-a-very-long-repository-name-that-should-still-be-sanitized-correctly",
			expected: "this-is-a-very-long-repository-name-that-should-still-be-sanitized-correctly",
		},
		{
			name:     "dots and dashes",
			input:    "my.project-2024",
			expected: "my-project-2024",
		},
		{
			name:     "parentheses and brackets",
			input:    "Project (2024) [Main]",
			expected: "project-2024-main",
		},
		{
			name:     "consecutive special chars",
			input:    "test___!!!___project",
			expected: "test-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeNameForID(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeNameForID(%q) = %q, expected %q", tt.input, result, tt.expected)
			}

			// Verify result only contains valid characters
			if result != "repo" && result != "" {
				pattern := regexp.MustCompile(`^[a-z0-9-]+$`)
				if !pattern.MatchString(result) {
					t.Errorf("sanitizeNameForID result %q contains invalid characters", result)
				}
			}
		})
	}
}

func TestSanitizeNameForID_EdgeCases(t *testing.T) {
	t.Log("Testing sanitizeNameForID edge cases")

	edgeCases := []struct {
		name     string
		input    string
		validate func(t *testing.T, result string)
	}{
		{
			name:  "no leading dashes",
			input: "---project",
			validate: func(t *testing.T, result string) {
				if startsWith(result, "-") {
					t.Errorf("Result %q should not start with dash", result)
				}
			},
		},
		{
			name:  "no trailing dashes",
			input: "project---",
			validate: func(t *testing.T, result string) {
				if endsWith(result, "-") {
					t.Errorf("Result %q should not end with dash", result)
				}
			},
		},
		{
			name:  "consecutive dashes collapsed",
			input: "test---project",
			validate: func(t *testing.T, result string) {
				if contains(result, "--") {
					t.Errorf("Result %q should not contain consecutive dashes", result)
				}
			},
		},
		{
			name:  "all lowercase",
			input: "UPPERCASE",
			validate: func(t *testing.T, result string) {
				if result != "uppercase" {
					t.Errorf("Result should be lowercase, got %q", result)
				}
			},
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeNameForID(tc.input)
			tc.validate(t, result)
		})
	}
}

func TestIDFormatValidation(t *testing.T) {
	t.Log("Testing that generated IDs match expected format")

	// Expected format: ^[a-z0-9-]+-\d+$
	pattern := regexp.MustCompile(`^[a-z0-9-]+-\d+$`)

	testCases := []string{
		"Personal Rules",
		"Work Project 2024",
		"test-repo",
		"MyProject@123",
		"!!!@@@",
		"",
		"12345",
	}

	timestamp := int64(1728756432)

	for _, name := range testCases {
		t.Run("format_validation_"+name, func(t *testing.T) {
			id := GenerateRepositoryID(name, timestamp)

			if !pattern.MatchString(id) {
				t.Errorf("ID %q doesn't match expected format ^[a-z0-9-]+-\\d+$", id)
			}

			// Verify structure: prefix-timestamp
			parts := regexp.MustCompile(`-\d+$`).Split(id, -1)
			if len(parts) < 1 {
				t.Errorf("ID %q should have prefix and timestamp parts", id)
			}
		})
	}
}

// Helper functions for ID generation tests

func formatTimestamp(ts int64) string {
	return regexp.MustCompile(`\d+`).FindString(GenerateRepositoryID("test", ts))
}

func contains(s, substr string) bool {
	return regexp.MustCompile(regexp.QuoteMeta(substr)).MatchString(s)
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
