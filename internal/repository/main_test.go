package repository

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestMain swaps go-keyring's provider for its in-memory mock before any test
// in this package runs. This keeps tests from touching the real OS credential
// store (macOS Keychain, Windows Credential Manager, Linux Secret Service) and
// lets the suite run on headless CI runners where no D-Bus Secret Service is
// available. The mock returns keyring.ErrNotFound for missing keys, matching
// the error semantics CredentialManager relies on.
func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}
