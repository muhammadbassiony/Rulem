// Package editors provides configuration and utilities for managing AI assistant and editor rule files.
//
// This package contains the definitions and configurations for various AI assistants
// and code editors that rulem supports. It defines how rule files should be named,
// where they should be placed, and how they should be transformed for different tools.
//
// Key components:
//   - EditorRuleConfig: Configuration for each supported editor/assistant
//   - RenameOption: Enumeration of file renaming strategies
//   - EditorRuleConfigs: Registry of all supported editors and assistants
//
// Supported editors and assistants:
//   - AGENTS.md (the recommended default: an open standard read by most AI tools)
//   - GitHub Copilot (Both general rules and custom instructions)
//   - Cursor
//   - Claude code
//   - Gemini CLI
//   - And more as the registry grows
//
// The first entry in EditorRuleConfigs is treated as the default: the import UI
// builds its selection list directly from this slice order, so AGENTS.md is the
// pre-selected recommendation.
//
// Each configuration specifies:
//   - The display name and description
//   - The target file path for the rule file
//   - How the file should be renamed (prefix, suffix, or full rename)
//   - The new name to use for the transformation
//
// This package serves as the central registry for supported tools and provides
// the mapping between user-friendly names and the technical file specifications
// required by each AI assistant or editor.
package editors

type RenameOption int

const (
	// RenameOptionNone means no renaming will be done
	RenameOptionNone RenameOption = iota
	// RenameOptionPrefix will add a prefix to the rule file name
	RenameOptionPrefix
	// RenameOptionSuffix will add a suffix to the rule file name
	RenameOptionSuffix
	// RenameOptionFull will rename the rule file completely
	RenameOptionFull
)

type EditorRuleConfig struct {
	// Name of the editor or the editors instruction file
	Name string

	// Explanation
	Explanation string

	// Path of the rule file to be created
	// this will NOT include the name
	RulePath string

	// Rename option specifies how the rule file should be renamed
	RenameOption RenameOption

	// NewName is the new name to be used if RenameOption is set
	// this can be used as either a prefix, suffix or full name
	// depending on the RenameOption
	NewName string
}

var EditorRuleConfigs = []EditorRuleConfig{
	{
		// https://agents.md
		Name:         "AGENTS.md (recommended)",
		Explanation:  "Open standard supported by most AI coding tools (Cursor, GitHub Copilot, Gemini CLI, Zed, Jules and 20+ more). Stewarded by the Agentic AI Foundation under the Linux Foundation. Placed at the project root so any compatible agent picks it up automatically. Start here unless you specifically need a tool-specific file below.\nFor more information, see https://agents.md",
		RulePath:     "./",
		RenameOption: RenameOptionFull,
		NewName:      "AGENTS.md",
	},
	{
		// https://code.visualstudio.com/docs/copilot/customization/custom-instructions#_use-a-githubcopilot-instructionsmd-file
		Name:         "Github Copilot - General instructions",
		Explanation:  "Repository-wide instructions applied to all Copilot chat requests in this workspace.\nFor more information, see https://code.visualstudio.com/docs/copilot/customization/custom-instructions#_use-a-githubcopilot-instructionsmd-file",
		RulePath:     ".github/",
		RenameOption: RenameOptionFull,
		NewName:      "copilot-instructions.md",
	},
	{
		// https://code.visualstudio.com/docs/copilot/customization/custom-instructions#_use-instructionsmd-files
		Name:         "Github Copilot - Instructions",
		Explanation:  "Path-scoped instructions Copilot applies depending on the files in the chat's context. Note: these files normally need an 'applyTo' frontmatter property to be scoped; since rulem copies the file verbatim, add that frontmatter yourself or prefer the repository-wide 'General instructions' option above.\nFor more information, see https://code.visualstudio.com/docs/copilot/customization/custom-instructions#_use-instructionsmd-files",
		RulePath:     ".github/instructions/",
		RenameOption: RenameOptionSuffix,
		NewName:      ".instructions.md",
	},
	{
		// https://cursor.com/docs/context/rules
		Name:         "Cursor rules",
		Explanation:  "Directory-scoped Cursor rule. Cursor only reads '.mdc' files under .cursor/rules/ (plain .md files are ignored), so the file is saved with a .mdc extension. Because rulem copies the file verbatim it has no frontmatter, so Cursor treats it as a manual/@-referenced rule rather than always-applied. For always-on rules, use the recommended AGENTS.md option, which Cursor also reads natively. Run this tool inside the directory where you want the scoped rule.\nFor more information, see https://cursor.com/docs/context/rules",
		RulePath:     ".cursor/rules/",
		RenameOption: RenameOptionSuffix,
		NewName:      ".mdc",
	},
	{
		// https://code.claude.com/docs/en/memory
		Name:         "Claude code",
		Explanation:  "This is a general instructions file that will be added to all messages. Claude Code reads CLAUDE.md, not AGENTS.md.\nFor more information, see https://code.claude.com/docs/en/memory",
		RulePath:     "./",
		RenameOption: RenameOptionFull,
		NewName:      "CLAUDE.md",
	},
	{
		// https://github.com/google-gemini/gemini-cli?tab=readme-ov-file#advanced-capabilities
		Name:         "Gemini CLI",
		Explanation:  "This is a general instructions file that will be added to all messages.\nFor more information, see https://github.com/google-gemini/gemini-cli?tab=readme-ov-file#advanced-capabilities",
		RulePath:     "./",
		RenameOption: RenameOptionFull,
		NewName:      "GEMINI.md",
	},
}

// Interface that is compatible with bubble list components
func (c EditorRuleConfig) Title() string       { return c.Name }
func (c EditorRuleConfig) Description() string { return c.Explanation }
func (c EditorRuleConfig) FilterValue() string {
	return c.Name + " " + c.Explanation + " " + c.RulePath + " " + c.NewName
}

func GetAllEditorRuleConfigs() []EditorRuleConfig {
	return EditorRuleConfigs
}

// GenerateRuleFileFullPath generates the full path for the rule file based on the configuration.
// It combines the RulePath with the NewName based on the RenameOption, this path is relative to the current working directory.
// If RenameOption is None, it returns the currentName as is.
//
// Parameters:
//   - currentName: The current name of the rule file, used to determine the final
//     name based on the RenameOption.
//
// Returns:
//   - string: The full path of the rule file, combining RulePath and NewName
func (c EditorRuleConfig) GenerateRuleFileFullPath(currentName string) string {
	var newName string
	switch c.RenameOption {
	case RenameOptionPrefix:
		newName = c.NewName + currentName
	case RenameOptionSuffix:
		// For suffix, handle empty suffix as no-op
		if c.NewName == "" {
			newName = currentName
		} else {
			// Remove extension first, then add suffix (which includes new extension)
			baseName := removeExtension(currentName)
			newName = baseName + c.NewName
		}
	case RenameOptionFull:
		newName = c.NewName
	case RenameOptionNone:
		// If no renaming is specified, return the current name as is
		newName = currentName
	default:
		// If no renaming is specified, return the current name as is
		newName = currentName
	}

	return c.RulePath + newName
}

// removeExtension removes the file extension from a filename
func removeExtension(filename string) string {
	if len(filename) == 0 {
		return filename
	}

	// Find the last dot
	lastDot := -1
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			lastDot = i
			break
		}
		// If we encounter a path separator before a dot, there's no extension
		if filename[i] == '/' || filename[i] == '\\' {
			break
		}
	}

	// If no dot found, or dot is at the beginning (hidden file), return as is
	if lastDot <= 0 {
		return filename
	}

	return filename[:lastDot]
}
