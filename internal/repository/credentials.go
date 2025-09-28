package repository

import (
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	// Service name for OS credential store
	credentialService = "rulem"
	// Key for GitHub Personal Access Token
	githubTokenKey = "github_pat"
)

// CredentialManager handles secure storage and retrieval of authentication credentials
type CredentialManager struct {
	service string
}

// NewCredentialManager creates a new credential manager instance
func NewCredentialManager() *CredentialManager {
	return &CredentialManager{
		service: credentialService,
	}
}

// StoreGitHubToken securely stores a GitHub Personal Access Token in the OS credential store.
// The token is validated before storage to ensure it has the required format.
//
// Parameters:
//   - token: GitHub Personal Access Token to store
//
// Returns:
//   - error: Storage errors or validation failures
func (cm *CredentialManager) StoreGitHubToken(token string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Validate token format (GitHub PATs have specific prefixes)
	if err := validateTokenFormat(token); err != nil {
		return fmt.Errorf("invalid token format: %w", err)
	}

	// Store in OS credential store
	if err := keyring.Set(cm.service, githubTokenKey, token); err != nil {
		return fmt.Errorf("failed to store token in credential store: %w", err)
	}

	return nil
}

// GetGitHubToken retrieves the stored GitHub Personal Access Token from the OS credential store.
//
// Returns:
//   - string: The stored Personal Access Token
//   - error: Retrieval errors or if no token is stored
func (cm *CredentialManager) GetGitHubToken() (string, error) {
	token, err := keyring.Get(cm.service, githubTokenKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", fmt.Errorf("no GitHub token found - please configure authentication in Settings → GitHub Authentication")
		}
		return "", fmt.Errorf("failed to retrieve token from credential store: %w", err)
	}

	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("stored token is empty - please update authentication in Settings → GitHub Authentication")
	}

	return token, nil
}

// DeleteGitHubToken removes the stored GitHub Personal Access Token from the OS credential store.
// This is useful for token rotation or when switching authentication methods.
//
// Returns:
//   - error: Deletion errors (returns nil if token doesn't exist)
func (cm *CredentialManager) DeleteGitHubToken() error {
	err := keyring.Delete(cm.service, githubTokenKey)
	if err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("failed to delete token from credential store: %w", err)
	}
	return nil
}

// HasGitHubToken checks if a GitHub Personal Access Token is stored without retrieving it.
// This is useful for UI state management and setup flow decisions.
//
// Returns:
//   - bool: true if a token is stored, false otherwise
func (cm *CredentialManager) HasGitHubToken() bool {
	_, err := keyring.Get(cm.service, githubTokenKey)
	return err == nil
}

// UpdateGitHubToken replaces the existing stored token with a new one.
// This is equivalent to calling StoreGitHubToken with proper validation.
//
// Parameters:
//   - newToken: New GitHub Personal Access Token to store
//
// Returns:
//   - error: Update errors or validation failures
func (cm *CredentialManager) UpdateGitHubToken(newToken string) error {
	// Validate new token before attempting to replace
	if err := cm.StoreGitHubToken(newToken); err != nil {
		return fmt.Errorf("failed to update token: %w", err)
	}
	return nil
}

// validateTokenFormat validates that the token matches GitHub PAT format expectations.
// GitHub Personal Access Tokens have specific prefixes depending on their type:
//   - Classic PATs: ghp_*
//   - Fine-grained PATs: github_pat_*
//   - OAuth tokens: gho_*
//   - User-to-server tokens: ghu_*
//   - Server-to-server tokens: ghs_*
//
// Parameters:
//   - token: Token string to validate
//
// Returns:
//   - error: Validation error if token format is invalid
func validateTokenFormat(token string) error {
	token = strings.TrimSpace(token)

	// Check minimum length (GitHub PATs are typically 40+ characters)
	if len(token) < 20 {
		return fmt.Errorf("token too short (minimum 20 characters)")
	}

	// Check for GitHub PAT prefixes
	validPrefixes := []string{
		"ghp_",        // Classic Personal Access Token
		"github_pat_", // Fine-grained Personal Access Token
		"gho_",        // OAuth token (sometimes used)
		"ghu_",        // User-to-server token
		"ghs_",        // Server-to-server token
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(token, prefix) {
			return nil
		}
	}

	// If no valid prefix found, it might still be a valid token but warn the user
	// Some older tokens or custom GitHub Enterprise tokens might not follow these patterns
	return fmt.Errorf("token does not match expected GitHub PAT format (should start with ghp_ or github_pat_)")
}

// GetCredentialStoreStatus returns information about the credential store availability
// and any potential issues. This is useful for troubleshooting and setup validation.
//
// Returns:
//   - map[string]any: Status information including availability, OS type, and any errors
func (cm *CredentialManager) GetCredentialStoreStatus() map[string]any {
	status := make(map[string]any)

	// Test if we can access the credential store
	testKey := "rulem_test"
	testValue := "test_value"

	// Try to set a test value
	setErr := keyring.Set(cm.service, testKey, testValue)
	if setErr != nil {
		status["available"] = false
		status["error"] = setErr.Error()
		return status
	}

	// Try to get the test value
	retrievedValue, getErr := keyring.Get(cm.service, testKey)
	if getErr != nil {
		status["available"] = false
		status["error"] = getErr.Error()
		// Clean up test key
		keyring.Delete(cm.service, testKey)
		return status
	}

	// Verify the value matches
	if retrievedValue != testValue {
		status["available"] = false
		status["error"] = "credential store corrupted - values don't match"
		// Clean up test key
		keyring.Delete(cm.service, testKey)
		return status
	}

	// Clean up test key
	deleteErr := keyring.Delete(cm.service, testKey)
	if deleteErr != nil {
		status["available"] = true
		status["warning"] = "credential store works but cleanup failed: " + deleteErr.Error()
		return status
	}

	status["available"] = true
	status["error"] = nil
	return status
}
