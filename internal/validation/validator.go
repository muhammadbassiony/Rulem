package validation

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ValidateMarkdownFile checks if the file is a valid instruction markdown file.
// Returns nil if valid, or an error describing the problem.
func ValidateMarkdownFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return fmt.Errorf("file is empty")
	}
	first := strings.TrimSpace(scanner.Text())
	if first == "" {
		return fmt.Errorf("first line (title/description) is empty")
	}
	// Optionally: check for required sections, e.g. starts with # or similar
	// if !strings.HasPrefix(first, "#") {
	// 	return fmt.Errorf("first line should start with a heading (#)")
	// }
	return nil
}
