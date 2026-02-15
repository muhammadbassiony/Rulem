package settingsmenu

import (
	"context"
)

type fakeCredentialManager struct {
	validateTokenErr error
	validateRepoErrs map[string]error
	storeErr         error
	storedToken      string
	getToken         string
	getErr           error
}

func (f *fakeCredentialManager) ValidateGitHubToken(token string) error {
	return f.validateTokenErr
}

func (f *fakeCredentialManager) ValidateGitHubTokenWithRepo(ctx context.Context, token string, repoURL string) error {
	if err, ok := f.validateRepoErrs[repoURL]; ok {
		return err
	}
	return nil
}

func (f *fakeCredentialManager) StoreGitHubToken(token string) error {
	if f.storeErr != nil {
		return f.storeErr
	}
	f.storedToken = token
	return nil
}

func (f *fakeCredentialManager) GetGitHubToken() (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	return f.getToken, nil
}
