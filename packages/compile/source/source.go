// Package source loads a spwn project from disk into the rich
// ProjectSource struct that compile Runtimes consume. It is the
// disk-IO half of the two-phase compiler: tsc analogy == the parser.
// Runtimes themselves are pure functions — they never touch the
// filesystem.
//
// A ProjectSource is intentionally generous: it carries every file a
// renderer might want (AGENTS.md, agent.yaml, profile, layer dirs,
// skills, hooks, profiles) so the claude-code renderer — and the
// future codex renderer — can pick whatever subset they need.
package source

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/project"
)

// ProjectSource is the in-memory representation of a spwn project on
// disk, ready to feed into a compile Runtime. Fields mirror what
// runtimes need to translate, not what the manifest strictly defines.
// This is the parsed IR — a Runtime consumes it.
type ProjectSource struct {
	// Manifest is the parsed spwn.yaml.
	Manifest *project.Manifest

	// Agents is every agent directory on disk under spwn/agents/ whose
	// name is referenced by at least one world in the manifest. Order
	// is deterministic (sorted by Name).
	Agents []AgentSource

	// Skills are every bare-markdown skill at the top of
	// spwn/packs/*.md. Nested .md files inside a directory-form
	// package belong to that pack's own pack.yaml and are not
	// enumerated here.
	Skills []SkillSource

	// Hooks is every file directly under spwn/hooks/.
	Hooks []HookSource

	// Profiles is every *.md file under spwn/profiles/ (optional).
	// Empty if the directory does not exist.
	Profiles []ProfileSource

	// RootDir is the absolute path of the project root — useful for
	// error messages and debugging.
	RootDir string
}

// AgentSource is a fully loaded agent directory.
type AgentSource struct {
	// Name is the directory basename under spwn/agents/.
	Name string

	// AgentMD is the raw bytes of spwn/agents/<name>/AGENTS.md — the
	// provider-neutral prompt. May be nil if the file is missing.
	AgentMD []byte

	// Config is the parsed spwn/agents/<name>/agent.yaml. Zero value
	// if the file is missing.
	Config AgentConfig

	// Layers contains the four layer directories: skills/, knowledge/,
	// playbooks/, journal/. Keys are file paths relative to the layer
	// root (e.g. "foo.md", "category/bar.md").
	Layers LayerFiles
}

// AgentConfig is the parsed agent.yaml. Only fields the compiler cares
// about are kept — the full agent package has a richer type that the
// host-side runtime uses for things like auth and mounts.
type AgentConfig struct {
	Name     string        `yaml:"name,omitempty"`
	Role     string        `yaml:"role,omitempty"`
	Team     string        `yaml:"team,omitempty"`
	Runtime  RuntimeConfig `yaml:"runtime,omitempty"`
	Deps []string `yaml:"deps,omitempty"`
}

// RuntimeConfig is the per-agent runtime override section of agent.yaml.
type RuntimeConfig struct {
	Backend  string `yaml:"backend,omitempty"`
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
	Auth     string `yaml:"auth,omitempty"`
}

// LayerFiles are the four per-agent layer directories.
type LayerFiles struct {
	Skills    map[string][]byte
	Knowledge map[string][]byte
	Playbooks map[string][]byte
	Journal   map[string][]byte
}

// SkillSource is one bare-markdown skill file directly under
// spwn/packs/*.md.
type SkillSource struct {
	// Name is the skill identifier — the path relative to
	// spwn/packs/ with the .md extension stripped.
	Name string

	// Content is the raw bytes of the skill file.
	Content []byte
}

// HookSource is one file directly under spwn/hooks/.
type HookSource struct {
	Name    string
	Content []byte
	Mode    os.FileMode
}

// ProfileSource is one file under spwn/profiles/.
type ProfileSource struct {
	Name    string
	Content []byte
}

// Load parses a spwn project rooted at projectRoot and returns a
// ProjectSource ready for a compile.Runtime to consume.
//
// projectRoot must be an absolute path to a directory containing
// spwn.yaml. On any I/O or parse error Load returns a detailed error
// with the offending path.
func Load(projectRoot string) (*ProjectSource, error) {
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	manifestPath := filepath.Join(abs, "spwn.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no spwn.yaml found at %s", manifestPath)
		}
		return nil, fmt.Errorf("stat %s: %w", manifestPath, err)
	}

	p, err := project.Load(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("load manifest %s: %w", manifestPath, err)
	}

	src := &ProjectSource{
		Manifest: p.Manifest,
		RootDir:  abs,
	}

	// Agents
	agents, err := loadAgents(abs, p)
	if err != nil {
		return nil, err
	}
	src.Agents = agents

	// Skills
	skills, err := loadSkills(abs)
	if err != nil {
		return nil, err
	}
	src.Skills = skills

	// Hooks
	hooks, err := loadHooks(abs)
	if err != nil {
		return nil, err
	}
	src.Hooks = hooks

	// Profiles (optional)
	profiles, err := loadProfiles(abs)
	if err != nil {
		return nil, err
	}
	src.Profiles = profiles

	return src, nil
}

func loadAgents(root string, p *project.Project) ([]AgentSource, error) {
	// The set of agent directories to load = the union of p.Agents
	// (deployable) and p.OrphanAgents (present on disk but not
	// referenced). Renderers decide what to do with orphans — Load's
	// job is just to surface every agent directory the user has.
	seen := map[string]string{}
	for _, a := range p.Agents {
		if !a.Exists {
			// Manifest references a name with no directory on
			// disk. Skip it here — the caller (manifest validate
			// or ToCompileInput) is the one that reports it.
			continue
		}
		seen[a.Name] = a.Path
	}
	for _, a := range p.OrphanAgents {
		if !a.Exists {
			continue
		}
		if _, ok := seen[a.Name]; !ok {
			seen[a.Name] = a.Path
		}
	}

	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]AgentSource, 0, len(names))
	for _, name := range names {
		dir := seen[name]
		src, err := loadAgent(name, dir)
		if err != nil {
			return nil, err
		}
		out = append(out, src)
	}

	// Also pick up directories on disk that neither list surfaced —
	// defensive against drift between manifest and disk (e.g. user
	// added a directory without touching spwn.yaml). They show up
	// here so renderers / validation see them.
	agentsDir := filepath.Join(root, "spwn", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", agentsDir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, ok := seen[e.Name()]; ok {
			continue
		}
		src, err := loadAgent(e.Name(), filepath.Join(agentsDir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, src)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func loadAgent(name, dir string) (AgentSource, error) {
	src := AgentSource{Name: name}

	// AGENTS.md
	agentMDPath := filepath.Join(dir, "AGENTS.md")
	if b, err := os.ReadFile(agentMDPath); err == nil {
		src.AgentMD = b
	} else if !os.IsNotExist(err) {
		return src, fmt.Errorf("read %s: %w", agentMDPath, err)
	}

	// agent.yaml
	configPath := filepath.Join(dir, "agent.yaml")
	if b, err := os.ReadFile(configPath); err == nil {
		var cfg AgentConfig
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return src, fmt.Errorf("parse %s: %w", configPath, err)
		}
		src.Config = cfg
	} else if !os.IsNotExist(err) {
		return src, fmt.Errorf("read %s: %w", configPath, err)
	}

	// Layer dirs
	layers, err := loadLayers(dir)
	if err != nil {
		return src, err
	}
	src.Layers = layers

	return src, nil
}

func loadLayers(agentDir string) (LayerFiles, error) {
	var lf LayerFiles
	var err error
	if lf.Skills, err = readTree(filepath.Join(agentDir, "skills")); err != nil {
		return lf, err
	}
	if lf.Knowledge, err = readTree(filepath.Join(agentDir, "knowledge")); err != nil {
		return lf, err
	}
	if lf.Playbooks, err = readTree(filepath.Join(agentDir, "playbooks")); err != nil {
		return lf, err
	}
	if lf.Journal, err = readTree(filepath.Join(agentDir, "journal")); err != nil {
		return lf, err
	}
	return lf, nil
}

// readTree walks dir recursively and returns a map of relative-path ->
// content. Returns nil (not an error) if dir does not exist. Skips
// .gitkeep and other hidden files.
func readTree(dir string) (map[string][]byte, error) {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", dir, err)
	}
	out := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		out[rel] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// loadSkills reads bare-markdown skill packages from
// <root>/spwn/packs/*.md. Files are loaded flat at the top level
// only — nested .md files inside a package directory belong to that
// pack's own pack.yaml and are not project-level skills.
func loadSkills(root string) ([]SkillSource, error) {
	dir := filepath.Join(root, "spwn", "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var out []SkillSource
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		id := strings.TrimSuffix(name, ".md")
		out = append(out, SkillSource{Name: id, Content: b})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func loadHooks(root string) ([]HookSource, error) {
	dir := filepath.Join(root, "spwn", "hooks")
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", dir, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var out []HookSource
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dir, name)
		info, err := e.Info()
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", full, err)
		}
		b, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", full, err)
		}
		out = append(out, HookSource{
			Name:    name,
			Content: b,
			Mode:    info.Mode(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func loadProfiles(root string) ([]ProfileSource, error) {
	dir := filepath.Join(root, "spwn", "profiles")
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", dir, err)
	}
	var out []ProfileSource
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if !strings.HasSuffix(name, ".md") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		id := strings.TrimSuffix(rel, ".md")
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		out = append(out, ProfileSource{Name: id, Content: b})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
