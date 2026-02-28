package mind

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

var subdirs = []string{"personas", "skills", "knowledge", "playbooks", "journal"}

// ResolvePath returns the filesystem path for a mind ID.
func ResolvePath(mindID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".universe", "minds", mindID), nil
}

// EnsureDir creates the mind directory and all subdirectories.
func EnsureDir(path string) error {
	for _, sub := range subdirs {
		dir := filepath.Join(path, sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating mind dir %s: %w", dir, err)
		}
	}
	return nil
}

// Validate logs warnings for missing mind subdirectories.
func Validate(path string) {
	for _, sub := range subdirs {
		dir := filepath.Join(path, sub)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			slog.Warn("mind subdirectory missing", "subdir", sub, "path", path)
		}
	}
}
