package validation

import (
	"os"
	"testing"
)

func TestValidateMarkdownFile(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		f, err := os.CreateTemp("", "valid-*.md")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())
		_, _ = f.WriteString("# Title\nSome content\n")
		f.Close()
		err = ValidateMarkdownFile(f.Name())
		if err != nil {
			t.Errorf("expected valid file, got error: %v", err)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		f, err := os.CreateTemp("", "empty-*.md")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())
		f.Close()
		err = ValidateMarkdownFile(f.Name())
		if err == nil || err.Error() != "file is empty" {
			t.Errorf("expected 'file is empty', got: %v", err)
		}
	})

	t.Run("empty first line", func(t *testing.T) {
		f, err := os.CreateTemp("", "emptyfirst-*.md")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())
		_, _ = f.WriteString("\nSome content\n")
		f.Close()
		err = ValidateMarkdownFile(f.Name())
		if err == nil || err.Error() != "first line (title/description) is empty" {
			t.Errorf("expected 'first line (title/description) is empty', got: %v", err)
		}
	})

	t.Run("whitespace first line", func(t *testing.T) {
		f, err := os.CreateTemp("", "wsfirst-*.md")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())
		_, _ = f.WriteString("   \nSome content\n")
		f.Close()
		err = ValidateMarkdownFile(f.Name())
		if err == nil || err.Error() != "first line (title/description) is empty" {
			t.Errorf("expected 'first line (title/description) is empty', got: %v", err)
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		err := ValidateMarkdownFile("/tmp/does-not-exist-12345.md")
		if err == nil || err.Error() == "" {
			t.Errorf("expected error for missing file, got: %v", err)
		}
	})
}
