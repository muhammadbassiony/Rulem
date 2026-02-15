package settingsmenu

import (
	"context"

	"rulem/internal/repository"
)

type mockCredentialManager struct {
	validateTokenErr error
	validateRepoErrs map[string]error
	validateReposErr error
	storeErr         error
	storedToken      string
	getToken         string
	getErr           error
}

func (m *mockCredentialManager) ValidateGitHubToken(token string) error {
	return m.validateTokenErr
}

func (m *mockCredentialManager) ValidateGitHubTokenWithRepo(ctx context.Context, token string, repoURL string) error {
	if err, ok := m.validateRepoErrs[repoURL]; ok {
		return err
	}
	return nil
}

func (m *mockCredentialManager) ValidateGitHubTokenForRepos(ctx context.Context, token string, repos []repository.RepositoryEntry) error {
	return m.validateReposErr
}

func (m *mockCredentialManager) StoreGitHubToken(token string) error {
	if m.storeErr != nil {
		return m.storeErr
	}
	m.storedToken = token
	return nil
}

func (m *mockCredentialManager) GetGitHubToken() (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	return m.getToken, nil
}
