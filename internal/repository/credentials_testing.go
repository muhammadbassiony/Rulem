package repository

import (
	"fmt"
	"testing"

	"github.com/zalando/go-keyring"
)

// credentials_testing.go provides test helpers for safely testing credential operations
// that interact with the OS keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service).
//
// # Why This File Exists
//
// Testing credential management requires special handling because:
//   1. We need to use the ACTUAL OS keyring (not mocks) to test real keyring behavior
//   2. Tests must not interfere with production credentials stored on the developer's machine
//   3. Tests must clean up after themselves to avoid polluting the keyring
//   4. Tests should skip gracefully in CI environments where keyring may not be available
//
// # When To Use These Helpers
//
// Use these helpers when:
//   - Testing any code that stores/retrieves tokens from the OS keyring
//   - Testing credential validation flows in UI components (e.g., setup menu)
//   - Writing integration tests that need temporary credentials
//
// Example usage in tests:
//
//	func TestSomethingWithCredentials(t *testing.T) {
//	    // Creates isolated credential manager with automatic cleanup
//	    testCM := repository.NewTestCredentialManager(t)
//
//	    // Use testCM.CredentialManager for your tests
//	    model.credManager = testCM.CredentialManager
//
//	    // Cleanup happens automatically via t.Cleanup()
//	}
//
// # How It Works
//
// 1. Each test gets a unique keyring service name (e.g., "rulem-test-TestMyTest")
// 2. This isolates test credentials from production and from other tests
// 3. Automatic cleanup via t.Cleanup() removes test credentials when test completes
// 4. Tests can safely run in parallel without conflicts

// TestCredentialManager wraps CredentialManager with test-specific functionality
// that allows for safe cleanup and isolation between tests.
//
// It uses a unique service name per test to prevent conflicts with:
//   - Production credentials on the developer's machine
//   - Other tests running in parallel
//
// Cleanup is automatic via t.Cleanup() registration.
type TestCredentialManager struct {
	*CredentialManager
	testService string
	t           *testing.T
}

// NewTestCredentialManager creates a new credential manager for testing.
//
// This is the primary entry point for testing credential operations. It:
//   - Creates an isolated credential manager with a unique service name
//   - Registers automatic cleanup to remove test credentials
//   - Returns a TestCredentialManager that embeds the real CredentialManager
//
// Usage:
//
//	testCM := repository.NewTestCredentialManager(t)
//	// Access the actual credential manager via embedded field
//	err := testCM.StoreGitHubToken("ghp_test_token")
//	// Or pass it to code under test
//	model.credManager = testCM.CredentialManager
//
// The cleanup happens automatically when the test completes (pass or fail).
func NewTestCredentialManager(t *testing.T) *TestCredentialManager {
	t.Helper()

	// Use a unique service name for testing to avoid conflicts
	testService := fmt.Sprintf("rulem-test-%s", t.Name())

	cm := &TestCredentialManager{
		CredentialManager: &CredentialManager{
			service: testService,
		},
		testService: testService,
		t:           t,
	}

	// Register cleanup to remove all test credentials
	t.Cleanup(func() {
		cm.Cleanup()
	})

	return cm
}

// Cleanup removes all test credentials from the keyring.
// This is automatically called via t.Cleanup() but can also be called manually.
func (tcm *TestCredentialManager) Cleanup() {
	tcm.t.Helper()

	// Try to delete the GitHub token if it exists
	// Ignore errors as the key might not exist
	_ = keyring.Delete(tcm.testService, githubTokenKey)
}

// SetupTestKeyring ensures the keyring is available for testing.
//
// This helper is useful when you need to verify keyring availability
// before running a test suite. It's optional - NewTestCredentialManager
// will work fine without it, but this provides an early check.
//
// Returns a cleanup function that should be called when done.
// If the keyring is not available (e.g., in CI environments), it skips the test.
//
// Usage:
//
//	func TestKeyringOperations(t *testing.T) {
//	    cleanup := repository.SetupTestKeyring(t) // Skips if keyring unavailable
//	    defer cleanup()
//	    // ... rest of test
//	}
//
// Note: Most tests don't need this - they can just use NewTestCredentialManager
// which handles keyring errors gracefully.
func SetupTestKeyring(t *testing.T) func() {
	t.Helper()

	// Test if keyring is available
	testService := fmt.Sprintf("rulem-keyring-test-%s", t.Name())
	testKey := "test_availability"
	testValue := "test_value"

	err := keyring.Set(testService, testKey, testValue)
	if err != nil {
		t.Skipf("Keyring not available, skipping test: %v", err)
	}

	// Clean up the test key
	cleanup := func() {
		_ = keyring.Delete(testService, testKey)
	}

	// Return cleanup function
	return cleanup
}

// CreateTestToken generates a valid-format test token for testing purposes.
//
// The token matches GitHub's format requirements but is not a real token.
// It will pass format validation but fail when used with actual GitHub operations.
//
// Parameters:
//   - prefix: Token prefix to use (e.g., "ghp_", "github_pat_"). Empty string uses "ghp_"
//
// Returns: A properly formatted test token string
//
// Usage:
//
//	token := repository.CreateTestToken("")           // Uses default "ghp_" prefix
//	token := repository.CreateTestToken("github_pat_") // Uses fine-grained PAT format
//	err := cm.ValidateGitHubToken(token)
//	assert.NoError(t, err) // Should pass format validation
func CreateTestToken(prefix string) string {
	if prefix == "" {
		prefix = "ghp_"
	}
	// Generate a token that matches GitHub's format requirements (40+ chars)
	return prefix + "1234567890abcdefghijklmnopqrstuvwxyzABCD"
}

// CreateInvalidFormatToken generates an invalid-format token for testing validation.
//
// This creates a token that will fail format validation. Use this to test
// error handling for invalid token formats.
//
// Returns: "invalid_token_format" (too short, wrong prefix)
//
// Usage:
//
//	token := repository.CreateInvalidFormatToken()
//	err := cm.ValidateGitHubToken(token)
//	assert.Error(t, err) // Should fail validation
func CreateInvalidFormatToken() string {
	return "invalid_token_format"
}

// AssertTokenStored verifies that a token is stored in the credential manager.
//
// This is a convenience assertion for tests that need to verify token storage.
// It retrieves the token and compares it to the expected value.
//
// Usage:
//
//	testCM := repository.NewTestCredentialManager(t)
//	testCM.StoreGitHubToken("ghp_test")
//	repository.AssertTokenStored(t, testCM, "ghp_test")
func AssertTokenStored(t *testing.T, cm *TestCredentialManager, expectedToken string) {
	t.Helper()

	token, err := cm.GetGitHubToken()
	if err != nil {
		t.Fatalf("Expected token to be stored, but got error: %v", err)
	}

	if token != expectedToken {
		t.Errorf("Expected token %q, got %q", expectedToken, token)
	}
}

// AssertTokenNotStored verifies that no token is stored in the credential manager.
//
// This is a convenience assertion for tests that need to verify tokens
// were not stored or were properly deleted.
//
// Usage:
//
//	testCM := repository.NewTestCredentialManager(t)
//	repository.AssertTokenNotStored(t, testCM) // Initially empty
//	testCM.StoreGitHubToken("ghp_test")
//	testCM.DeleteGitHubToken()
//	repository.AssertTokenNotStored(t, testCM) // After deletion
func AssertTokenNotStored(t *testing.T, cm *TestCredentialManager) {
	t.Helper()

	hasToken := cm.HasGitHubToken()
	if hasToken {
		t.Error("Expected no token to be stored, but HasGitHubToken returned true")
	}

	_, err := cm.GetGitHubToken()
	if err == nil {
		t.Error("Expected error when getting non-existent token, but got nil")
	}
}
