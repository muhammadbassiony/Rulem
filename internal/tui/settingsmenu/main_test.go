package settingsmenu

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestMain swaps go-keyring's provider for its in-memory mock before any test
// in this package runs. Most settings tests use a mock credential manager, but
// integration tests that construct repository.NewCredentialManager() would
// otherwise reach the real OS credential store, which is unavailable on
// headless CI runners (no D-Bus Secret Service).
func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}
