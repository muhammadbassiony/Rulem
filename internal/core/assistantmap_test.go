package core

import (
	"testing"
)

func TestGetAssistantLocation_Defaults(t *testing.T) {
	tests := []struct {
		assistant string
		expect    string
	}{
		{"copilot", "~/.config/Code/User/prompts/"},
		{"cursor", "~/.cursor/prompts/"},
		{"claude", "~/.claude/prompts/"},
		{"gemini-cli", "~/.gemini/prompts/"},
		{"opencode", "~/.opencode/prompts/"},
	}
	for _, tc := range tests {
		loc, err := GetAssistantLocation(tc.assistant, "")
		if err != nil {
			t.Errorf("unexpected error for %s: %v", tc.assistant, err)
		}
		if loc != tc.expect {
			t.Errorf("expected %s, got %s", tc.expect, loc)
		}
	}
}

func TestGetAssistantLocation_Custom(t *testing.T) {
	loc, err := GetAssistantLocation("copilot", "/tmp/custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc != "/tmp/custom" {
		t.Errorf("expected custom location, got %s", loc)
	}
}

func TestGetAssistantLocation_Unsupported(t *testing.T) {
	_, err := GetAssistantLocation("notareal", "")
	if err == nil {
		t.Error("expected error for unsupported assistant")
	}
}
