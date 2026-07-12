package setupmenu

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

// TestMain swaps go-keyring's provider for its in-memory mock before any test
// in this package runs. Setup flow tests store GitHub PATs through
// repository.CredentialManager; the mock keeps them out of the real OS
// credential store and lets the suite run on headless CI runners where no
// D-Bus Secret Service is available.
func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}
