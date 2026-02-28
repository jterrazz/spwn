package mind

import (
	"fmt"
	"os"
	"path/filepath"
)

var subdirs = []string{"personas", "skills", "knowledge", "playbooks", "journal", "sessions"}

// ResolvePath returns the filesystem path for a mind ID.
func ResolvePath(mindID string) (string, error) {
	base, err := BasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, mindID), nil
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

// Validate checks for missing mind subdirectories and returns an error listing them.
func Validate(path string) error {
	var missing []string
	for _, sub := range subdirs {
		dir := filepath.Join(path, sub)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			missing = append(missing, sub)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing mind subdirectories: %v", missing)
	}
	return nil
}

// Subdirs returns a copy of the canonical mind subdirectory names.
func Subdirs() []string {
	out := make([]string, len(subdirs))
	copy(out, subdirs)
	return out
}

// BasePath returns the base directory for all minds (~/.universe/minds/).
func BasePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".universe", "minds"), nil
}

// ListMinds returns the names of all minds in the base directory.
func ListMinds() ([]string, error) {
	base, err := BasePath()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading minds directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
