package core

import "fmt"

// Supported assistants
const (
	AssistantCopilot   = "copilot"
	AssistantCursor    = "cursor"
	AssistantClaude    = "claude"
	AssistantGeminiCLI = "gemini-cli"
	AssistantOpencode  = "opencode"
)

// AssistantLocationMap maps assistant names to their default instruction file locations.
var AssistantLocationMap = map[string]string{
	AssistantCopilot:   "~/.config/Code/User/prompts/",
	AssistantCursor:    "~/.cursor/prompts/",
	AssistantClaude:    "~/.claude/prompts/",
	AssistantGeminiCLI: "~/.gemini/prompts/",
	AssistantOpencode:  "~/.opencode/prompts/",
}

// GetAssistantLocation returns the destination directory for the given assistant.
// If a custom location is provided, it takes precedence.
func GetAssistantLocation(assistant, custom string) (string, error) {
	if custom != "" {
		return custom, nil
	}
	loc, ok := AssistantLocationMap[assistant]
	if !ok {
		return "", fmt.Errorf("unsupported assistant: %s", assistant)
	}
	return loc, nil
}
