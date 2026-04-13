package auth

import (
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/packages/foundation"
)

// SaveToken persists a token to the auth cache file.
func SaveToken(token string) error {
	path := tokenPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(token)), 0600)
}

// ClearToken removes the cached token.
func ClearToken() error {
	return os.Remove(tokenPath())
}

// DisableProvider marks a provider as disabled (credentials won't be synced).
func DisableProvider(p Provider) error {
	path := disabledPath(p)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("disabled"), 0600)
}

// EnableProvider removes the disabled marker for a provider.
func EnableProvider(p Provider) error {
	os.Remove(disabledPath(p))
	return nil
}

// IsProviderDisabled checks if a provider has been explicitly disabled.
func IsProviderDisabled(p Provider) bool {
	_, err := os.Stat(disabledPath(p))
	return err == nil
}

func disabledPath(p Provider) string {
	return filepath.Join(foundation.CredentialsDir(), ".disabled-"+string(p))
}

// ReadCachedToken reads the cached token from disk.
func ReadCachedToken() string {
	data, err := os.ReadFile(tokenPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func tokenPath() string {
	return foundation.BaseDir() + "/.auth-token"
}
