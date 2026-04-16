// This file is the example-gallery face of the catalog: the
// List/Get/Install surface used by `spwn init <slug>` and the web UI
// marketplace. Example entries live alongside dependency entries
// under catalog/<slug>/ and share the catalogFS embed defined in
// loader.go — an entry is an "example" when it ships an
// example.yaml metadata sidecar next to its spwn.yaml.
//
// Contributors edit example directories directly from the repo root;
// add a new one, update the embed list in loader.go and the
// shippedSlugs list below, and it shows up in the gallery.
package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"spwn.sh/packages/platform"
)

// shippedSlugs is the canonical list of bundled examples, in
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

// ShippedSlugs returns the list of bundled examples as a fresh copy.
// Exposed for binary-level bundling tests that need to verify every
// expected slug is present without reading the embed FS directly.
func ShippedSlugs() []string {
	out := make([]string, len(shippedSlugs))
	copy(out, shippedSlugs)
	return out
}

// Example is the public-facing description of one example.
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
	Slug          string   `json:"slug"`
	AgentsAdded   []string `json:"agentsAdded"`
	AgentsSkipped []string `json:"agentsSkipped"`
	// ManifestAdded is true if this install wrote a fresh spwn.yaml,
	// false if one already existed at baseDir/spwn.yaml (and was left
	// untouched per the no-overwrite rule).
	ManifestAdded bool `json:"manifestAdded"`
	// WorldsAdded/WorldsSkipped are retained for API compatibility
	// with the old per-world-file install surface. With the v2
	// schema, at most one entry lands here: the example's world
	// name, populated from example.yaml#worlds.
	WorldsAdded   []string `json:"worldsAdded"`
	WorldsSkipped []string `json:"worldsSkipped"`
}

// ErrNotFound is returned by Get and Install when a slug does not
// match any bundled example.
var ErrNotFound = errors.New("example not found")

// List returns every example the binary knows about, sorted by slug
// for stable output. README bodies are omitted; call Get for details.
//
// Iteration order is driven by shippedSlugs rather than embed.FS root
// ReadDir so the list is deterministic even if new entries appear in
// the embed before shippedSlugs is updated - keeping the "canonical
// list" guarantee explicit.
func List() ([]Example, error) {
	out := make([]Example, 0, len(shippedSlugs))
	for _, slug := range shippedSlugs {
		ex, err := loadMetadata(slug)
		if err != nil {
			// Skip broken examples rather than failing the whole list.
			// A malformed example should never break the gallery for
			// users who only care about the other four.
			continue
		}
		out = append(out, ex)
	}
	// No sort - shippedSlugs order IS the gallery order.
	return out, nil
}

// Get returns one example's metadata + its README. Returns
// ErrNotFound if the slug is unknown.
func Get(slug string) (Example, error) {
	ex, err := loadMetadata(slug)
	if err != nil {
		return Example{}, err
	}
	readmeBytes, err := catalogFS.ReadFile(joinFS(slug, "README.md"))
	if err == nil {
		ex.Readme = string(readmeBytes)
	}
	return ex, nil
}

// Install materializes an example into baseDir as a project tree:
//
//	baseDir/
//	├── spwn.yaml              (copied from <slug>/spwn.yaml)
//	└── spwn/
//	    └── agents/<name>/     (copied from <slug>/agents/<name>/)
//
// Existing files are NEVER overwritten - whatever the user already
// has on disk wins. The report tells the caller what was added vs
// skipped.
//
// baseDir should be the project root. In legacy/global mode (no
// project discoverable) callers can still pass ~/.spwn or the
// platform.UserDir() and the same layout will appear underneath.
//
// After Install, the caller can `spwn up` to bring the world online.
func Install(slug, baseDir string) (InstallReport, error) {
	ex, err := loadMetadata(slug)
	if err != nil {
		return InstallReport{}, err
	}
	rep := InstallReport{Slug: slug}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return rep, fmt.Errorf("create project dir: %w", err)
	}

	// --- spwn.yaml ---
	manifestDst := filepath.Join(baseDir, "spwn.yaml")
	if exists(manifestDst) {
		// Record the world name(s) declared by the example as skipped
		// so existing callers (web UI, activity log) still see a
		// "worlds skipped" signal.
		rep.WorldsSkipped = append(rep.WorldsSkipped, ex.Worlds...)
	} else {
		manifestSrc := joinFS(slug, "spwn.yaml")
		data, rerr := catalogFS.ReadFile(manifestSrc)
		if rerr != nil {
			return rep, fmt.Errorf("read %s: %w", manifestSrc, rerr)
		}
		if werr := os.WriteFile(manifestDst, data, 0o644); werr != nil {
			return rep, fmt.Errorf("write %s: %w", manifestDst, werr)
		}
		rep.ManifestAdded = true
		rep.WorldsAdded = append(rep.WorldsAdded, ex.Worlds...)
	}

	// --- spwn.lock (committed dep pin) ---
	lockDst := filepath.Join(baseDir, "spwn.lock")
	if !exists(lockDst) {
		lockSrc := joinFS(slug, "spwn.lock")
		if data, rerr := catalogFS.ReadFile(lockSrc); rerr == nil {
			_ = os.WriteFile(lockDst, data, 0o644)
		}
	}

	// --- agents ---
	agentsRoot := filepath.Join(baseDir, "spwn", "agents")
	if err := os.MkdirAll(agentsRoot, 0o755); err != nil {
		return rep, fmt.Errorf("create agents dir: %w", err)
	}
	agentsSrc := joinFS(slug, "agents")
	agentEntries, err := catalogFS.ReadDir(agentsSrc)
	if err == nil {
		for _, e := range agentEntries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			dst := filepath.Join(agentsRoot, name)
			if exists(dst) {
				// Agent directory exists - but it might be broken
				// (e.g. created by a previous version or partially
				// cleaned up). If identity/profile.md is missing, copy
				// the example's identity/ layer on top without touching
				// user data like journal/ or knowledge/.
				identityProfile := filepath.Join(dst, "identity", "profile.md")
				if !exists(identityProfile) {
					identitySrc := joinFS(agentsSrc, name, "identity")
					identityDst := filepath.Join(dst, "identity")
					if cperr := copyDirFS(catalogFS, identitySrc, identityDst); cperr == nil {
						rep.AgentsAdded = append(rep.AgentsAdded, name+" (repaired)")
					}
					// Also copy agent.yaml if missing
					manifestDst := filepath.Join(dst, "agent.yaml")
					if !exists(manifestDst) {
						if data, rerr := catalogFS.ReadFile(joinFS(agentsSrc, name, "agent.yaml")); rerr == nil {
							_ = os.WriteFile(manifestDst, data, 0o644)
						}
					}
				} else {
					rep.AgentsSkipped = append(rep.AgentsSkipped, name)
				}
				continue
			}
			if cperr := copyDirFS(catalogFS, joinFS(agentsSrc, name), dst); cperr != nil {
				return rep, fmt.Errorf("copy agent %s: %w", name, cperr)
			}
			rep.AgentsAdded = append(rep.AgentsAdded, name)
		}
	}

	_ = ex // reserved for future use (e.g. emitting an activity event)
	return rep, nil
}

// InstallInto is a convenience wrapper that installs an example
// into the active project root when one is discoverable, else into
// the user-global ~/.spwn (legacy global mode). Callers that need
// to target an explicit path should use Install directly.
func InstallInto(slug string) (InstallReport, error) {
	root := platform.ProjectRoot()
	if root == "" {
		root = platform.BaseDir()
	}
	return Install(slug, root)
}

// ── internals ─────────────────────────────────────────────────────────

func loadMetadata(slug string) (Example, error) {
	data, err := catalogFS.ReadFile(joinFS(slug, "example.yaml"))
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

// path joins with forward slashes - required because embed.FS always
// uses forward slashes regardless of the host OS.
func joinFS(parts ...string) string {
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
