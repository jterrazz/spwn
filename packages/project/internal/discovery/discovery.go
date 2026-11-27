// Package discovery walks filesystem paths to locate the nearest
// spwn.yaml. The walk stops at the filesystem root - same behavior as
// `git rev-parse --show-toplevel`.
package discovery

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ManifestFileName is the canonical manifest filename at the root of
// every spwn project.
const ManifestFileName = "spwn.yaml"

// ErrLink is the error of a spwn.yaml found as a link.
var ErrLink = errors.New("is a link, which a project's manifest never is")

// Find walks up from startPath searching for ManifestFileName. Returns
// the absolute manifest path and the project root (the directory that
// contains the manifest). The third return value reports whether
// anything was found - the caller can decide whether that's an error.
// A spwn.yaml that is a link is an error naming it: a project's manifest
// is never reached through a link.
func Find(startPath string) (manifestPath, root string, found bool, err error) {
	abs, err := filepath.Abs(startPath)
	if err != nil {
		return "", "", false, fmt.Errorf("resolve %s: %w", startPath, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", "", false, fmt.Errorf("stat %s: %w", abs, err)
	}
	dir := abs
	if !info.IsDir() {
		dir = filepath.Dir(abs)
	}
	for {
		candidate := filepath.Join(dir, ManifestFileName)
		if fi, statErr := os.Lstat(candidate); statErr == nil && fi.Mode()&fs.ModeSymlink != 0 {
			return "", "", false, &fs.PathError{Op: "find", Path: candidate, Err: ErrLink}
		} else if statErr == nil && !fi.IsDir() {
			return candidate, dir, true, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", false, nil
		}
		dir = parent
	}
}

// RootFor returns the project root given an explicit manifest path.
// Useful for Load when the caller already knows where spwn.yaml lives.
func RootFor(manifestPath string) (string, error) {
	abs, err := filepath.Abs(manifestPath)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", manifestPath, err)
	}
	return filepath.Dir(abs), nil
}
