package core

import (
	"os"
	"os/exec"
)

// EditFile launches the user's preferred editor for the given file.
// Uses $EDITOR, or falls back to nano/vi.
func EditFile(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if _, err := exec.LookPath("nano"); err == nil {
			editor = "nano"
		} else if _, err := exec.LookPath("vi"); err == nil {
			editor = "vi"
		} else {
			return err
		}
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
