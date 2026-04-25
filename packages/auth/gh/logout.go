package gh

import (
	"fmt"
	"os"
)

// Logout removes the spwn-side gh cache. Returns os.ErrNotExist
// when nothing was stored. Does NOT touch the host's
// ~/.config/gh — `gh auth logout` on the host is the user's call.
func Logout() error {
	dir := CacheDir()
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}
	return nil
}
