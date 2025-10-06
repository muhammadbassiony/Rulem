package repository

import (
	"strings"
	"testing"
)

func TestNewCredentialManager(t *testing.T) {
	cm := NewCredentialManager()
	if cm.service != credentialService {
		t.Errorf("NewCredentialManager() service = %v, want %v", cm.service, credentialService)
	}
}

func TestValidateTokenFormat(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid classic PAT",
			token:   "ghp_1234567890abcdef1234567890abcdef12345678",
			wantErr: false,
		},
		{
			name:    "valid fine-grained PAT",
			token:   "github_pat_1234567890abcdef1234567890abcdef12345678_ABCDEFGHIJKLMNOP",
			wantErr: false,
		},
		{
			name:    "valid OAuth token",
			token:   "gho_1234567890abcdef1234567890abcdef12345678",
			wantErr: false,
		},
		{
			name:    "valid user-to-server token",
			token:   "ghu_1234567890abcdef1234567890abcdef12345678",
			wantErr: false,
		},
		{
			name:    "valid server-to-server token",
			token:   "ghs_1234567890abcdef1234567890abcdef12345678",
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
			errMsg:  "token too short",
		},
		{
			name:    "whitespace only token",
			token:   "   \t\n  ",
			wantErr: true,
			errMsg:  "token too short",
		},
		{
			name:    "too short token",
			token:   "ghp_short",
			wantErr: true,
			errMsg:  "token too short",
		},
		{
			name:    "invalid prefix",
			token:   "invalid_1234567890abcdef1234567890abcdef12345678",
			wantErr: true,
			errMsg:  "does not match expected GitHub PAT format",
		},
		{
			name:    "no prefix",
			token:   "1234567890abcdef1234567890abcdef12345678",
			wantErr: true,
			errMsg:  "does not match expected GitHub PAT format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTokenFormat(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateTokenFormat() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateTokenFormat() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateTokenFormat() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCredentialManager_ValidateGitHubToken(t *testing.T) {
	cm := NewCredentialManager()

	tests := []struct {
		name    string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid token",
			token:   "ghp_1234567890abcdef1234567890abcdef12345678",
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
			errMsg:  "token cannot be empty",
		},
		{
			name:    "whitespace only token",
			token:   "   ",
			wantErr: true,
			errMsg:  "token cannot be empty",
		},
		{
			name:    "invalid prefix",
			token:   "invalid_1234567890abcdef1234567890abcdef12345678",
			wantErr: true,
			errMsg:  "invalid token format",
		},
		{
			name:    "too short",
			token:   "ghp_short",
			wantErr: true,
			errMsg:  "token too short",
		},
		{
			name:    "fine-grained PAT",
			token:   "github_pat_11AAAAAAA0aBcDeFgHiJkLmNoPqRsTuVwXyZ123456789abcdef",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.ValidateGitHubToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateGitHubToken() expected error but got none for token: %q", tt.token)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateGitHubToken() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateGitHubToken() unexpected error for valid token: %v", err)
				}
			}
		})
	}
}

func TestCredentialManager_StoreGitHubToken(t *testing.T) {
	cm := NewCredentialManager()

	tests := []struct {
		name    string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid token",
			token:   "ghp_1234567890abcdef1234567890abcdef12345678",
			wantErr: false,
		},
		{
			name:    "fine-grained PAT",
			token:   "github_pat_11AAAAAAA0aBcDeFgHiJkLmNoPqRsTuVwXyZ123456789abcdef",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pre-validate the token first (this is now the expected workflow)
			err := cm.ValidateGitHubToken(tt.token)
			if err != nil {
				t.Errorf("ValidateGitHubToken() failed for valid token: %v", err)
				return
			}

			err = cm.StoreGitHubToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Errorf("StoreGitHubToken() expected error but got none for token: %q", tt.token)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("StoreGitHubToken() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					// This might fail in CI/testing environments without keyring access
					// So we'll accept this as expected behavior
					t.Logf("StoreGitHubToken() failed in test environment (expected): %v", err)
				}
			}
		})
	}
}

func TestCredentialManager_GetGitHubToken_NoToken(t *testing.T) {
	cm := NewCredentialManager()

	// Clean up any existing token first
	_ = cm.DeleteGitHubToken()

	_, err := cm.GetGitHubToken()
	if err == nil {
		t.Error("GetGitHubToken() should return error when no token stored")
	}

	expectedMsgs := []string{
		"no GitHub token found",
		"please configure authentication in Settings",
	}

	errStr := err.Error()
	for _, msg := range expectedMsgs {
		if !strings.Contains(errStr, msg) {
			t.Errorf("GetGitHubToken() error = %v, want error containing %q", err, msg)
		}
	}
}

func TestCredentialManager_HasGitHubToken_Initially(t *testing.T) {
	cm := NewCredentialManager()

	// Clean up any existing token first
	_ = cm.DeleteGitHubToken()

	if cm.HasGitHubToken() {
		t.Error("HasGitHubToken() should return false initially")
	}
}

func TestCredentialManager_DeleteGitHubToken_NoToken(t *testing.T) {
	cm := NewCredentialManager()

	// Delete should not fail even if no token exists
	err := cm.DeleteGitHubToken()
	if err != nil {
		t.Errorf("DeleteGitHubToken() unexpected error when no token exists: %v", err)
	}
}

func TestCredentialManager_UpdateGitHubToken(t *testing.T) {
	cm := NewCredentialManager()
	testToken := "ghp_1234567890abcdef1234567890abcdef12345678"

	tests := []struct {
		name    string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid token update",
			token:   testToken,
			wantErr: false,
		},
		{
			name:    "invalid token update",
			token:   "invalid_token",
			wantErr: true,
			errMsg:  "failed to validate new token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.UpdateGitHubToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateGitHubToken() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("UpdateGitHubToken() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					// This might fail in CI/testing environments without keyring access
					t.Logf("UpdateGitHubToken() failed in test environment (expected): %v", err)
				}
			}
		})
	}
}

func TestCredentialManager_GetCredentialStoreStatus(t *testing.T) {
	cm := NewCredentialManager()
	status := cm.GetCredentialStoreStatus()

	// Should return a map with availability information
	if status == nil {
		t.Error("GetCredentialStoreStatus() should not return nil")
	}

	// Should have 'available' key
	available, exists := status["available"]
	if !exists {
		t.Error("GetCredentialStoreStatus() should include 'available' key")
	}

	// Available should be a boolean
	if _, ok := available.(bool); !ok {
		t.Errorf("GetCredentialStoreStatus()['available'] should be bool, got %T", available)
	}

	// If not available, should have error message
	if !available.(bool) {
		if _, hasError := status["error"]; !hasError {
			t.Error("GetCredentialStoreStatus() should include 'error' key when not available")
		}
	}
}

// TestCredentialManager_FullFlow tests the complete store/retrieve/delete cycle
// This test may fail in environments without keyring access, which is expected
func TestCredentialManager_FullFlow(t *testing.T) {
	cm := NewCredentialManager()
	testToken := "ghp_1234567890abcdef1234567890abcdef12345678"

	// Clean up any existing token first
	_ = cm.DeleteGitHubToken()

	// Initially should have no token
	if cm.HasGitHubToken() {
		t.Error("HasGitHubToken() should return false initially")
	}

	// Validate and store token (may fail in test environment)
	err := cm.ValidateGitHubToken(testToken)
	if err != nil {
		t.Errorf("ValidateGitHubToken() failed: %v", err)
		return
	}

	err = cm.StoreGitHubToken(testToken)
	if err != nil {
		t.Logf("StoreGitHubToken() failed in test environment (expected): %v", err)
		return // Skip rest of test if keyring unavailable
	}

	// Should now have token
	if !cm.HasGitHubToken() {
		t.Error("HasGitHubToken() should return true after storing token")
	}

	// Should be able to retrieve token
	retrievedToken, err := cm.GetGitHubToken()
	if err != nil {
		t.Errorf("GetGitHubToken() unexpected error: %v", err)
	}
	if retrievedToken != testToken {
		t.Errorf("GetGitHubToken() = %v, want %v", retrievedToken, testToken)
	}

	// Update token
	newToken := "ghp_abcdef1234567890abcdef1234567890abcdef12"
	err = cm.UpdateGitHubToken(newToken)
	if err != nil {
		t.Errorf("UpdateGitHubToken() unexpected error: %v", err)
	}

	// Should retrieve updated token
	retrievedToken, err = cm.GetGitHubToken()
	if err != nil {
		t.Errorf("GetGitHubToken() after update unexpected error: %v", err)
	}
	if retrievedToken != newToken {
		t.Errorf("GetGitHubToken() after update = %v, want %v", retrievedToken, newToken)
	}

	// Delete token
	err = cm.DeleteGitHubToken()
	if err != nil {
		t.Errorf("DeleteGitHubToken() unexpected error: %v", err)
	}

	// Should no longer have token
	if cm.HasGitHubToken() {
		t.Error("HasGitHubToken() should return false after deletion")
	}

	_, err = cm.GetGitHubToken()
	if err == nil {
		t.Error("GetGitHubToken() should return error after deletion")
	}
}

// TestCredentialManager_KeyringUnavailable tests behavior when keyring service is unavailable
func TestCredentialManager_KeyringUnavailable(t *testing.T) {
	cm := NewCredentialManager()

	// Test that errors are handled gracefully
	// This test documents expected behavior when keyring is unavailable
	// rather than asserting specific outcomes

	status := cm.GetCredentialStoreStatus()
	if status != nil {
		available, _ := status["available"].(bool)
		if !available {
			t.Logf("Keyring unavailable in test environment: %v", status["error"])

			// Test that validation still works (doesn't require keyring)
			err := cm.ValidateGitHubToken("ghp_1234567890abcdef1234567890abcdef12345678")
			if err != nil {
				t.Errorf("ValidateGitHubToken() should work even when keyring unavailable: %v", err)
			}

			// Test that storage operations fail gracefully
			err = cm.StoreGitHubToken("ghp_1234567890abcdef1234567890abcdef12345678")
			if err == nil {
				t.Error("StoreGitHubToken() should fail when keyring unavailable")
			}

			if strings.Contains(err.Error(), "failed to store token in credential store") {
				t.Logf("Expected keyring error: %v", err)
			} else {
				t.Errorf("Unexpected error format: %v", err)
			}
		}
	}
}
