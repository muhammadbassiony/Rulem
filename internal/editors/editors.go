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
		// https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file
		Name:         "Github Copilot - General instructions",
		Explanation:  "This is a general instructions file that will be added to all messages.\nFor more information, see https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file",
		RulePath:     ".github/",
		RenameOption: RenameOptionFull,
		NewName:      "copilot-instructions.md",
	},
	{
		// https://code.visualstudio.com/docs/copilot/copilot-customization#_use-instructionsmd-files
		Name:         "Github Copilot - Instructions",
		Explanation:  "These are instructions that Github Copilot will be attached to all messages but used depending on the files in the chat's context.\nFor more information, see https://code.visualstudio.com/docs/copilot/copilot-customization#_use-instructionsmd-files",
		RulePath:     ".github/instructions/",
		RenameOption: RenameOptionSuffix,
		NewName:      ".instructions.md",
	},
	{
		// https://opencode.ai/docs/rules/
		Name:         "AGENTS.md",
		Explanation:  "This is a general instructions file that will be added to all messages. This name is expected by some tools such as SST Opencode.\nFor more information, see https://opencode.ai/docs/rules/",
		RulePath:     "./",
		RenameOption: RenameOptionFull,
		NewName:      "AGENTS.md",
	},
	{
		// https://docs.cursor.com/en/context/rules
		Name:         "Cursor rules",
		Explanation:  "This is a general instructions file that will be added to all messages. Cursor supports having scoped rules per directory, to use this run this tool inside the directory where you want to save these rules.\nFor more information, see https://docs.cursor.com/en/context/rules",
		RulePath:     ".cursor/rules/",
		RenameOption: RenameOptionNone,
		NewName:      "",
	},
	{
		// https://docs.anthropic.com/en/docs/claude-code/memory#determine-memory-type
		Name:         "Claude code",
		Explanation:  "This is a general instructions file that will be added to all messages.\nFor more information, see https://docs.anthropic.com/en/docs/claude-code/memory#determine-memory-type",
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
