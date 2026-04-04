package auth

import (
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/core/foundation"
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
