// This file is the init-template face of the catalog — the
// List/Get/Install surface used by `spwn init <slug>` and the web UI
// gallery. Every catalog entry is a raw directory (no spwn/ nesting);
// the subset with a `worlds:` section in its spwn.yaml is what the
// gallery surfaces as a ready-to-scaffold project.
//
// Install() is a generic recursive copy: spwn.yaml and spwn.lock land
// at the destination root; every other top-level subdir (agents/,
// skills/, tools/, hooks/, files/) is wrapped under dest/spwn/ to
// match the user-project layout.
package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/platform"
)

// Example is the public-facing description of one gallery entry.
type Example struct {
	Slug        string   `json:"slug" yaml:"slug"`
	Name        string   `json:"name" yaml:"name"`
	Tagline     string   `json:"tagline" yaml:"tagline"`
	Description string   `json:"description" yaml:"description"`
	Agents      []string `json:"agents" yaml:"agents"`
	Worlds      []string `json:"worlds" yaml:"worlds"`
	Command     string   `json:"command,omitempty" yaml:"command,omitempty"`
}

// InstallReport describes everything Install wrote to disk.
type InstallReport struct {
	Slug          string   `json:"slug"`
	AgentsAdded   []string `json:"agentsAdded"`
	AgentsSkipped []string `json:"agentsSkipped"`
	ManifestAdded bool     `json:"manifestAdded"`
	WorldsAdded   []string `json:"worldsAdded"`
	WorldsSkipped []string `json:"worldsSkipped"`
}

// ErrNotFound is returned by Get and Install when a slug does not
// match any gallery-eligible entry (i.e. one with worlds: defined).
var ErrNotFound = errors.New("example not found")

// topLevelSubdirs enumerates which directories under a catalog entry
// get wrapped under dest/spwn/ at init time. spwn.yaml and spwn.lock
// stay at the root; everything listed here moves under spwn/.
var topLevelSubdirs = []string{"agents", "skills", "tools", "hooks", "files"}

// ShippedSlugs returns the list of gallery-eligible entries (those
// with a `worlds:` section in spwn.yaml), sorted by display order.
func ShippedSlugs() []string {
	slugs := galleryBacked()
	// Canonical gallery order: startup first (multi-agent showcase),
	// matrix second (zero-to-hello-world), then the rest
	// alphabetically.
	sort.SliceStable(slugs, func(i, j int) bool {
		p := func(s string) int {
			switch s {
			case "startup":
				return 0
			case "matrix":
				return 1
			default:
				return 2
			}
		}
		if p(slugs[i]) != p(slugs[j]) {
			return p(slugs[i]) < p(slugs[j])
		}
		return slugs[i] < slugs[j]
	})
	return slugs
}

// galleryBacked returns every embedded subdir whose spwn.yaml
// defines a non-empty `worlds:` section.
func galleryBacked() []string {
	entries, err := fs.ReadDir(catalogFS, ".")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		schema, err := loadEntrySchema(e.Name())
		if err != nil {
			continue
		}
		if hasWorlds(schema) {
			out = append(out, e.Name())
		}
	}
	return out
}

// List returns every gallery-eligible example in canonical display
// order.
func List() ([]Example, error) {
	slugs := ShippedSlugs()
	out := make([]Example, 0, len(slugs))
	for _, slug := range slugs {
		ex, err := Get(slug)
		if err != nil {
			continue
		}
		out = append(out, ex)
	}
	return out, nil
}

// Get returns one example's metadata (from its spwn.yaml). Returns
// ErrNotFound when the slug does not exist or does not expose worlds.
func Get(slug string) (Example, error) {
	schema, err := loadEntrySchema(slug)
	if err != nil || !hasWorlds(schema) {
		return Example{}, ErrNotFound
	}
	worlds, agents := worldsAndAgents(schema)
	ex := Example{
		Slug:        slug,
		Name:        deriveTitle(slug, schema),
		Tagline:     schema.Tagline,
		Description: schema.Description,
		Agents:      agents,
		Worlds:      worlds,
	}
	if len(worlds) > 0 {
		ex.Command = "spwn up " + worlds[0]
	}
	return ex, nil
}

// Install materialises an example into baseDir as a project tree:
//
//	baseDir/
//	├── spwn.yaml            (verbatim copy)
//	├── spwn.lock            (verbatim copy, if present)
//	└── spwn/
//	    ├── agents/<name>/   (from <slug>/agents/...)
//	    ├── skills/          (from <slug>/skills/...)
//	    └── …                (hooks/, tools/, files/)
//
// Existing files are NEVER overwritten — whatever the user already
// has on disk wins. The report records what was added vs skipped.
func Install(slug, baseDir string) (InstallReport, error) {
	schema, err := loadEntrySchema(slug)
	if err != nil || !hasWorlds(schema) {
		return InstallReport{}, ErrNotFound
	}
	rep := InstallReport{Slug: slug}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return rep, fmt.Errorf("create project dir: %w", err)
	}

	worldNames, _ := worldsAndAgents(schema)

	manifestDst := filepath.Join(baseDir, "spwn.yaml")
	if exists(manifestDst) {
		rep.WorldsSkipped = append(rep.WorldsSkipped, worldNames...)
	} else {
		data, rerr := catalogFS.ReadFile(slug + "/spwn.yaml")
		if rerr != nil {
			return rep, fmt.Errorf("read %s/spwn.yaml: %w", slug, rerr)
		}
		if werr := os.WriteFile(manifestDst, data, 0o644); werr != nil {
			return rep, fmt.Errorf("write %s: %w", manifestDst, werr)
		}
		rep.ManifestAdded = true
		rep.WorldsAdded = append(rep.WorldsAdded, worldNames...)
	}

	lockDst := filepath.Join(baseDir, "spwn.lock")
	if !exists(lockDst) {
		if data, rerr := catalogFS.ReadFile(slug + "/spwn.lock"); rerr == nil {
			_ = os.WriteFile(lockDst, data, 0o644)
		}
	}

	spwnRoot := filepath.Join(baseDir, "spwn")
	if err := os.MkdirAll(spwnRoot, 0o755); err != nil {
		return rep, fmt.Errorf("create spwn dir: %w", err)
	}
	for _, sub := range topLevelSubdirs {
		src := slug + "/" + sub
		if _, err := fs.Stat(catalogFS, src); err != nil {
			continue
		}
		if sub == "agents" {
			if err := copyAgents(catalogFS, src, spwnRoot, &rep); err != nil {
				return rep, err
			}
			continue
		}
		dst := filepath.Join(spwnRoot, sub)
		if exists(dst) {
			continue
		}
		if err := copyDirFS(catalogFS, src, dst); err != nil {
			return rep, fmt.Errorf("copy %s: %w", sub, err)
		}
	}

	return rep, nil
}

// copyAgents implements the per-agent granularity Install wants:
// records per-agent added/skipped in the report and repairs an
// agent dir that exists but is missing SOUL.md.
func copyAgents(src fs.FS, agentsSrc, spwnRoot string, rep *InstallReport) error {
	agentsDst := filepath.Join(spwnRoot, "agents")
	if err := os.MkdirAll(agentsDst, 0o755); err != nil {
		return fmt.Errorf("create agents dir: %w", err)
	}
	entries, err := fs.ReadDir(src, agentsSrc)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		dst := filepath.Join(agentsDst, name)
		if exists(dst) {
			soulDst := filepath.Join(dst, platform.SoulFileName)
			if !exists(soulDst) {
				// Repair: seed the missing SOUL.md from the catalog copy.
				if data, rerr := fs.ReadFile(src, agentsSrc+"/"+name+"/"+platform.SoulFileName); rerr == nil {
					_ = os.MkdirAll(dst, 0o755)
					_ = os.WriteFile(soulDst, data, 0o644)
				}
				manifestDst := filepath.Join(dst, "agent.yaml")
				if !exists(manifestDst) {
					if data, rerr := fs.ReadFile(src, agentsSrc+"/"+name+"/agent.yaml"); rerr == nil {
						_ = os.WriteFile(manifestDst, data, 0o644)
					}
				}
				rep.AgentsAdded = append(rep.AgentsAdded, name+" (repaired)")
			} else {
				rep.AgentsSkipped = append(rep.AgentsSkipped, name)
			}
			continue
		}
		if err := copyDirFS(src, agentsSrc+"/"+name, dst); err != nil {
			return fmt.Errorf("copy agent %s: %w", name, err)
		}
		rep.AgentsAdded = append(rep.AgentsAdded, name)
	}
	return nil
}

// InstallInto is a convenience wrapper that installs an example
// into the active project root when one is discoverable, else into
// the user-global ~/.spwn (legacy global mode).
func InstallInto(slug string) (InstallReport, error) {
	root := platform.ProjectRoot()
	if root == "" {
		root = platform.BaseDir()
	}
	return Install(slug, root)
}

// ── internals ─────────────────────────────────────────────────────────

// loadEntrySchema reads an entry's spwn.yaml into the shared
// dependency.Schema so every catalog face reads the same bytes.
func loadEntrySchema(slug string) (*dependency.Schema, error) {
	data, err := catalogFS.ReadFile(slug + "/spwn.yaml")
	if err != nil {
		return nil, err
	}
	var s dependency.Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s/spwn.yaml: %w", slug, err)
	}
	return &s, nil
}

func hasWorlds(s *dependency.Schema) bool {
	return s != nil && len(s.Worlds.Content) > 0
}

// worldsAndAgents derives the flat world-name and agent-name slices
// from the parsed worlds yaml.Node, sorted for stability.
func worldsAndAgents(s *dependency.Schema) (worlds, agents []string) {
	if s == nil || s.Worlds.Kind != yaml.MappingNode {
		return nil, nil
	}
	seenAgent := map[string]bool{}
	for i := 0; i+1 < len(s.Worlds.Content); i += 2 {
		key := s.Worlds.Content[i]
		val := s.Worlds.Content[i+1]
		worlds = append(worlds, key.Value)
		if val.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(val.Content); j += 2 {
			if val.Content[j].Value != "agents" {
				continue
			}
			agentList := val.Content[j+1]
			if agentList.Kind != yaml.SequenceNode {
				continue
			}
			for _, a := range agentList.Content {
				if !seenAgent[a.Value] {
					seenAgent[a.Value] = true
					agents = append(agents, a.Value)
				}
			}
		}
	}
	sort.Strings(worlds)
	sort.Strings(agents)
	return worlds, agents
}

func deriveTitle(slug string, s *dependency.Schema) string {
	if s != nil && s.Title != "" {
		return s.Title
	}
	return strings.Title(strings.ReplaceAll(slug, "-", " "))
}

// copyDirFS recursively copies a directory from the embedded FS onto
// the host filesystem. Never overwrites existing files.
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

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
