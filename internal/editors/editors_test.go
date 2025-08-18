package editors

import (
	"testing"
)

func TestGenerateRuleFileFullPath(t *testing.T) {
	tests := []struct {
		name        string
		config      EditorRuleConfig
		currentName string
		expected    string
	}{
		// RenameOptionNone tests
		{
			name: "none option keeps original name",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionNone,
				NewName:      "",
			},
			currentName: "rules.md",
			expected:    "./rules.md",
		},
		{
			name: "none option with subdirectory",
			config: EditorRuleConfig{
				RulePath:     "config/",
				RenameOption: RenameOptionNone,
				NewName:      "",
			},
			currentName: "guidelines.md",
			expected:    "config/guidelines.md",
		},

		// RenameOptionPrefix tests
		{
			name: "prefix option adds prefix",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionPrefix,
				NewName:      "copilot-",
			},
			currentName: "instructions.md",
			expected:    "./copilot-instructions.md",
		},
		{
			name: "prefix option with subdirectory",
			config: EditorRuleConfig{
				RulePath:     ".github/",
				RenameOption: RenameOptionPrefix,
				NewName:      "gh-",
			},
			currentName: "rules.md",
			expected:    ".github/gh-rules.md",
		},
		{
			name: "prefix option with empty prefix",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionPrefix,
				NewName:      "",
			},
			currentName: "test.md",
			expected:    "./test.md",
		},

		// RenameOptionSuffix tests
		{
			name: "suffix option adds suffix with extension",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionSuffix,
				NewName:      ".instructions.md",
			},
			currentName: "rules.md",
			expected:    "./rules.instructions.md",
		},
		{
			name: "suffix option with complex filename",
			config: EditorRuleConfig{
				RulePath:     ".github/instructions/",
				RenameOption: RenameOptionSuffix,
				NewName:      ".instructions.md",
			},
			currentName: "coding-standards.md",
			expected:    ".github/instructions/coding-standards.instructions.md",
		},
		{
			name: "suffix option with no extension",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionSuffix,
				NewName:      "_backup",
			},
			currentName: "README",
			expected:    "./README_backup",
		},
		{
			name: "suffix option with multiple extensions",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionSuffix,
				NewName:      ".min",
			},
			currentName: "script.js.map",
			expected:    "./script.js.min",
		},
		{
			name: "suffix option with hidden file",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionSuffix,
				NewName:      "_config",
			},
			currentName: ".gitignore",
			expected:    "./.gitignore_config",
		},
		{
			name: "suffix option with empty suffix",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionSuffix,
				NewName:      "",
			},
			currentName: "test.md",
			expected:    "./test.md",
		},

		// RenameOptionFull tests
		{
			name: "full option replaces entire name",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionFull,
				NewName:      "copilot-instructions.md",
			},
			currentName: "whatever.txt",
			expected:    "./copilot-instructions.md",
		},
		{
			name: "full option with subdirectory",
			config: EditorRuleConfig{
				RulePath:     ".github/",
				RenameOption: RenameOptionFull,
				NewName:      "AGENTS.md",
			},
			currentName: "old-name.md",
			expected:    ".github/AGENTS.md",
		},
		{
			name: "full option with empty new name",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionFull,
				NewName:      "",
			},
			currentName: "test.md",
			expected:    "./",
		},

		// Real-world examples from EditorRuleConfigs
		{
			name: "github copilot general instructions",
			config: EditorRuleConfig{
				RulePath:     ".github/",
				RenameOption: RenameOptionFull,
				NewName:      "copilot-instructions.md",
			},
			currentName: "my-rules.md",
			expected:    ".github/copilot-instructions.md",
		},
		{
			name: "github copilot scoped instructions",
			config: EditorRuleConfig{
				RulePath:     ".github/instructions/",
				RenameOption: RenameOptionSuffix,
				NewName:      ".instructions.md",
			},
			currentName: "react-rules.md",
			expected:    ".github/instructions/react-rules.instructions.md",
		},
		{
			name: "cursor rules no rename",
			config: EditorRuleConfig{
				RulePath:     ".cursor/rules/",
				RenameOption: RenameOptionNone,
				NewName:      "",
			},
			currentName: "typescript.md",
			expected:    ".cursor/rules/typescript.md",
		},
		{
			name: "AGENTS.md full replacement",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionFull,
				NewName:      "AGENTS.md",
			},
			currentName: "custom-rules.md",
			expected:    "./AGENTS.md",
		},

		// Edge cases
		{
			name: "empty current name",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionPrefix,
				NewName:      "prefix-",
			},
			currentName: "",
			expected:    "./prefix-",
		},
		{
			name: "current name with spaces",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionSuffix,
				NewName:      "_modified",
			},
			currentName: "my rules.md",
			expected:    "./my rules_modified",
		},
		{
			name: "current name with dots in filename",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionSuffix,
				NewName:      ".backup",
			},
			currentName: "version.1.2.md",
			expected:    "./version.1.2.backup",
		},
		{
			name: "just extension",
			config: EditorRuleConfig{
				RulePath:     "./",
				RenameOption: RenameOptionSuffix,
				NewName:      "_test",
			},
			currentName: ".md",
			expected:    "./.md_test",
		},
		{
			name: "file with path separators in name should not be affected",
			config: EditorRuleConfig{
				RulePath:     "./config/",
				RenameOption: RenameOptionSuffix,
				NewName:      "_backup",
			},
			currentName: "file.md",
			expected:    "./config/file_backup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GenerateRuleFileFullPath(tt.currentName)
			if result != tt.expected {
				t.Errorf("GenerateRuleFileFullPath() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestRemoveExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "simple file with extension",
			filename: "file.txt",
			expected: "file",
		},
		{
			name:     "file with multiple dots",
			filename: "archive.tar.gz",
			expected: "archive.tar",
		},
		{
			name:     "hidden file",
			filename: ".gitignore",
			expected: ".gitignore",
		},
		{
			name:     "hidden file with extension",
			filename: ".bashrc.backup",
			expected: ".bashrc",
		},
		{
			name:     "no extension",
			filename: "README",
			expected: "README",
		},
		{
			name:     "empty string",
			filename: "",
			expected: "",
		},
		{
			name:     "just dot",
			filename: ".",
			expected: ".",
		},
		{
			name:     "just extension",
			filename: ".txt",
			expected: ".txt",
		},
		{
			name:     "multiple extensions",
			filename: "script.min.js",
			expected: "script.min",
		},
		{
			name:     "file with spaces",
			filename: "my file.txt",
			expected: "my file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeExtension(tt.filename)
			if result != tt.expected {
				t.Errorf("removeExtension(%q) = %q, expected %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestEditorRuleConfigsIntegrity(t *testing.T) {
	// Test that all predefined configs are valid and work correctly
	for _, config := range EditorRuleConfigs {
		t.Run(config.Name, func(t *testing.T) {
			// Test with a sample filename
			testFilename := "test-rules.md"
			result := config.GenerateRuleFileFullPath(testFilename)

			// Basic validation - result should not be empty for most cases
			if config.RenameOption != RenameOptionFull || config.NewName != "" {
				if result == "" {
					t.Errorf("Config %q produced empty result for filename %q", config.Name, testFilename)
				}
			}

			// Result should start with the RulePath
			if config.RulePath != "" && !stringHasPrefix(result, config.RulePath) {
				t.Errorf("Config %q result %q does not start with RulePath %q", config.Name, result, config.RulePath)
			}

			// Test that the function doesn't panic
			_ = config.GenerateRuleFileFullPath("")
			_ = config.GenerateRuleFileFullPath("no-extension")
			_ = config.GenerateRuleFileFullPath(".hidden")
		})
	}
}

// stringHasPrefix is a simple prefix check since we can't import strings in tests without dependencies
func stringHasPrefix(s, prefix string) bool {
	if len(prefix) > len(s) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}
