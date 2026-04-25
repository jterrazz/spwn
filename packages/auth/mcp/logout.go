package mcp

import (
	"fmt"
	"os"
	"path/filepath"
)

// Logout deletes the per-provider OAuth directory (tokens.json +
// client.json). Returns os.ErrNotExist if the provider was never
// logged in. Other directories under CacheDir() are untouched.
func Logout(p Provider) error {
	dir := filepath.Join(CacheDir(), "oauth", providerKey(p.URL))
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}
	return nil
}
