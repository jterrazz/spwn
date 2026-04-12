// Package examples ships a curated gallery of ready-made spwn
// templates — full worlds and agents with pre-written personas —
// that first-time users can install with one click or one command.
//
// Every template lives at /examples/<slug>/ at the repo root and is
// embedded into the binary at build time via go:embed. The template
// directories sit alongside this Go file so contributors can discover
// and edit them directly from the repo root. The package exposes:
//
//   List(): the metadata for every template in the gallery
//   Get(slug): one template's metadata + its bundled README
//   Install(slug, baseDir): copy the template's world configs and
//       agent directories into the user's ~/.spwn tree
//
// The templates themselves are intentionally small and read-only on
// disk — install is a one-time copy operation, and once copied the
// user can edit the files freely without affecting the source.
//
// The embed directive below lists every slug explicitly. When adding
// a new template, add a new directory AND append it here AND to
// shippedSlugs — the list is load-bearing and the shipped-templates
// test will fail loudly if a directory exists without a matching
// embed entry.
package examples

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/core/foundation"

	"gopkg.in/yaml.v3"
)

//go:embed all:macrohard all:matrix all:paperclip-factory all:research-lab all:startup
var templatesFS embed.FS

// shippedSlugs is the canonical list of bundled templates, in
// display order. The TestShippedSlugsMatchEmbed test asserts this
// matches both the embed directive above and the directories
// actually present on disk.
// shippedSlugs controls the gallery display order. Startup first
// because it's the best showcase of multi-agent collaboration.
var shippedSlugs = []string{
	"startup",
	"matrix",
	"paperclip-factory",
	"research-lab",
	"macrohard",
}

// ShippedSlugs returns the list of bundled templates as a fresh copy.
// Exposed for binary-level bundling tests that need to verify every
// expected slug is present without reading the embed FS directly.
func ShippedSlugs() []string {
	out := make([]string, len(shippedSlugs))
	copy(out, shippedSlugs)
	return out
}

// Example is the public-facing description of one template.
type Example struct {
	Slug        string   `json:"slug" yaml:"slug"`
	Name        string   `json:"name" yaml:"name"`
	Tagline     string   `json:"tagline" yaml:"tagline"`
	Description string   `json:"description" yaml:"description"`
	Agents      []string `json:"agents" yaml:"agents"`
	Worlds      []string `json:"worlds" yaml:"worlds"`
	Command     string   `json:"command,omitempty" yaml:"command,omitempty"`
	// Readme holds the bundled README.md content. Populated by Get,
	// nil by List (so listing stays cheap).
	Readme string `json:"readme,omitempty"`
}

// InstallReport describes everything Install wrote to disk.
type InstallReport struct {
	Slug         string   `json:"slug"`
	AgentsAdded  []string `json:"agentsAdded"`
	AgentsSkipped []string `json:"agentsSkipped"`
	WorldsAdded  []string `json:"worldsAdded"`
	WorldsSkipped []string `json:"worldsSkipped"`
}

// ErrNotFound is returned by Get and Install when a slug does not
// match any bundled template.
var ErrNotFound = errors.New("example not found")

// List returns every template the binary knows about, sorted by slug
// for stable output. README bodies are omitted; call Get for details.
//
// Iteration order is driven by shippedSlugs rather than embed.FS root
// ReadDir so the list is deterministic even if new entries appear in
// the embed before shippedSlugs is updated — keeping the "canonical
// list" guarantee explicit.
func List() ([]Example, error) {
	out := make([]Example, 0, len(shippedSlugs))
	for _, slug := range shippedSlugs {
		ex, err := loadMetadata(slug)
		if err != nil {
			// Skip broken templates rather than failing the whole list.
			// A malformed example should never break the gallery for
			// users who only care about the other four.
			continue
		}
		out = append(out, ex)
	}
	// No sort — shippedSlugs order IS the gallery order.
	return out, nil
}

// Get returns one template's metadata + its README. Returns
// ErrNotFound if the slug is unknown.
func Get(slug string) (Example, error) {
	ex, err := loadMetadata(slug)
	if err != nil {
		return Example{}, err
	}
	readmeBytes, err := templatesFS.ReadFile(path(slug, "README.md"))
	if err == nil {
		ex.Readme = string(readmeBytes)
	}
	return ex, nil
}

// Install copies a template's world configs and agent directories
// into baseDir (typically ~/.spwn). Existing files are NEVER
// overwritten — the slug and filename you already have on disk win.
// The report tells the caller what was added vs skipped.
//
// After Install, the caller can `spwn up -c <world>` to actually
// spawn a container.
func Install(slug, baseDir string) (InstallReport, error) {
	ex, err := loadMetadata(slug)
	if err != nil {
		return InstallReport{}, err
	}
	rep := InstallReport{Slug: slug}

	// --- worlds ---
	worldsRoot := filepath.Join(baseDir, "worlds")
	if err := os.MkdirAll(worldsRoot, 0o755); err != nil {
		return rep, fmt.Errorf("create worlds dir: %w", err)
	}
	worldsSrc := path(slug, "worlds")
	worldEntries, err := templatesFS.ReadDir(worldsSrc)
	if err == nil {
		for _, e := range worldEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			dst := filepath.Join(worldsRoot, e.Name())
			if exists(dst) {
				rep.WorldsSkipped = append(rep.WorldsSkipped, e.Name())
				continue
			}
			data, rerr := templatesFS.ReadFile(path(worldsSrc, e.Name()))
			if rerr != nil {
				return rep, fmt.Errorf("read %s: %w", e.Name(), rerr)
			}
			if werr := os.WriteFile(dst, data, 0o644); werr != nil {
				return rep, fmt.Errorf("write %s: %w", dst, werr)
			}
			rep.WorldsAdded = append(rep.WorldsAdded, e.Name())
		}
	}

	// --- agents ---
	agentsRoot := filepath.Join(baseDir, "agents")
	if err := os.MkdirAll(agentsRoot, 0o755); err != nil {
		return rep, fmt.Errorf("create agents dir: %w", err)
	}
	agentsSrc := path(slug, "agents")
	agentEntries, err := templatesFS.ReadDir(agentsSrc)
	if err == nil {
		for _, e := range agentEntries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			dst := filepath.Join(agentsRoot, name)
			if exists(dst) {
				rep.AgentsSkipped = append(rep.AgentsSkipped, name)
				continue
			}
			if cperr := copyDirFS(templatesFS, path(agentsSrc, name), dst); cperr != nil {
				return rep, fmt.Errorf("copy agent %s: %w", name, cperr)
			}
			rep.AgentsAdded = append(rep.AgentsAdded, name)
		}
	}

	_ = ex // reserved for future use (e.g. emitting an activity event)
	return rep, nil
}

// InstallInto is a convenience wrapper that targets the default
// ~/.spwn home directory.
func InstallInto(slug string) (InstallReport, error) {
	return Install(slug, foundation.BaseDir())
}

// ── internals ─────────────────────────────────────────────────────────

func loadMetadata(slug string) (Example, error) {
	data, err := templatesFS.ReadFile(path(slug, "example.yaml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Example{}, ErrNotFound
		}
		return Example{}, err
	}
	var ex Example
	if err := yaml.Unmarshal(data, &ex); err != nil {
		return Example{}, fmt.Errorf("parse %s/example.yaml: %w", slug, err)
	}
	if ex.Slug == "" {
		ex.Slug = slug
	}
	return ex, nil
}

// path joins with forward slashes — required because embed.FS always
// uses forward slashes regardless of the host OS.
func path(parts ...string) string {
	return strings.Join(parts, "/")
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// copyDirFS recursively copies a directory from the embedded FS onto
// the host filesystem. Preserves relative structure, creates parent
// directories as needed, never overwrites existing files (Install
// pre-checks the destination dir).
func copyDirFS(src fs.FS, srcRoot, dstRoot string) error {
	return fs.WalkDir(src, srcRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, srcRoot)
		rel = strings.TrimPrefix(rel, "/")
		target := filepath.Join(dstRoot, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
