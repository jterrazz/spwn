package spwn

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CopyGateTools copies every catalog tool under spwn:<name> that
// declares a `gate:` section in its tool.yaml into <gateToolsRoot>/
// <short>/. Returns the slugs that were copied (or refreshed) so
// the CLI can show them to the user.
//
// Idempotent: existing files are overwritten with current catalog
// contents on every install, so `spwn install spwn:x` after a
// catalog update picks up the new tool code.
//
// Returns an empty list (no error) when the entry has no
// gate-shaped tools — most catalog entries are agent-side only
// (spwn:python, spwn:gh, ...).
func CopyGateTools(refName, gateToolsRoot string) ([]string, error) {
	entryRoot := path.Join(contentRoot, refName)
	if _, err := fs.Stat(catalogFS, entryRoot); err != nil {
		// Entry doesn't ship in the embedded catalog — caller likely
		// resolved a local or registry ref. Not our problem.
		return nil, nil
	}
	toolsDir := path.Join(entryRoot, "tools")
	subs, err := fs.ReadDir(catalogFS, toolsDir)
	if err != nil {
		// Entry has no tools/ subdir (skill-only or template).
		return nil, nil
	}
	var copied []string
	for _, sub := range subs {
		if !sub.IsDir() {
			continue
		}
		toolDir := path.Join(toolsDir, sub.Name())
		if !hasGateSection(toolDir) {
			continue
		}
		dest := filepath.Join(gateToolsRoot, sub.Name())
		if err := copyEmbedDir(toolDir, dest); err != nil {
			return nil, fmt.Errorf("copy %s → %s: %w", toolDir, dest, err)
		}
		copied = append(copied, sub.Name())
	}
	return copied, nil
}

// hasGateSection peeks at a tool.yaml to decide whether it's
// gate-shaped. Avoids materializing the full schema — we only need
// to know if `gate:` is present and non-empty.
func hasGateSection(toolDir string) bool {
	raw, err := fs.ReadFile(catalogFS, path.Join(toolDir, "tool.yaml"))
	if err != nil {
		return false
	}
	var probe struct {
		Gate yaml.Node `yaml:"gate"`
	}
	if err := yaml.Unmarshal(raw, &probe); err != nil {
		return false
	}
	return probe.Gate.Kind != 0 && len(probe.Gate.Content) > 0
}

// copyEmbedDir recursively writes every file in `src` (a path inside
// the catalog embed FS) to `dst` on the host filesystem. Directories
// are 0700 (gate cred-dir convention), files are 0644 except *.js
// scripts which get 0755 so they're directly executable.
func copyEmbedDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return err
	}
	return fs.WalkDir(catalogFS, src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, src)
		rel = strings.TrimPrefix(rel, "/")
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, err := fs.ReadFile(catalogFS, p)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".sh") {
			mode = 0o755
		}
		return os.WriteFile(target, data, mode)
	})
}
