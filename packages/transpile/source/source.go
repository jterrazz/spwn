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
	// spwn/skills/*.md. Nested .md files inside a directory-form
	// package belong to that dependency's own spwn.yaml and are not
	// enumerated here.
	Skills []SkillSource

	// Hooks is every file directly under spwn/hooks/.
	Hooks []HookSource

	// Commands is every *.md file directly under spwn/commands/.
	// Each command becomes a slash-invoked prompt shortcut in the
	// runtime (e.g. `/refactor` in Claude Code / Codex). Empty if
	// the directory does not exist.
	Commands []CommandSource

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

	// Soul is the raw bytes of spwn/agents/<name>/SOUL.md — the
	// agent's identity body. May be nil if the file is missing.
	// Renderers that can @-import (claude-code) don't need this;
	// renderers that must inline (codex) do.
	Soul []byte

	// Config is the parsed spwn/agents/<name>/agent.yaml. Zero value
	// if the file is missing.
	Config AgentConfig

	// Layers contains the two content-bearing agent layer directories:
	// playbooks/, journal/. Keys are file paths relative to the layer
	// root (e.g. "foo.md", "category/bar.md"). Skills live at the
	// world scope (see LayerFiles doc) and knowledge lives on the
	// world (spwn/worlds/<name>/knowledge/, bind-mounted at
	// /world/knowledge/).
	Layers LayerFiles
}

// AgentConfig is the parsed agent.yaml. Only fields the compiler cares
// about are kept — the full agent package has a richer type that the
// host-side runtime uses for things like auth and mounts.
type AgentConfig struct {
	Name        string        `yaml:"name,omitempty"`
	Description string        `yaml:"description,omitempty"`
	Role        string        `yaml:"role,omitempty"`
	Team        string        `yaml:"team,omitempty"`
	Runtime     RuntimeConfig `yaml:"runtime,omitempty"`
	Deps        []string      `yaml:"dependencies,omitempty"`
}

// RuntimeConfig is the per-agent runtime override section of agent.yaml.
type RuntimeConfig struct {
	Backend  string `yaml:"backend,omitempty"`
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
	Auth     string `yaml:"auth,omitempty"`
}

// LayerFiles are the content-bearing per-agent layer directories
// (identity is loaded separately via the entry file pipeline).
// Knowledge is NOT a per-agent layer — it's world-scoped at
// spwn/worlds/<name>/knowledge/ and bind-mounted into /world/knowledge/.
// Skills are NOT a per-agent layer either — they're build-time
// dependencies resolved via the `skill:` scheme or shipped by tools,
// injected into /world/skills/ at image time.
type LayerFiles struct {
	Playbooks map[string][]byte
	Journal   map[string][]byte
}

// SkillSource is one project-authored skill under spwn/skills/. The
// canonical on-disk shape is a directory per skill:
//
//	spwn/skills/<name>/
//	  SKILL.md          (required; YAML frontmatter + body)
//	  template.md       (optional sidecar — anything goes)
//	  scripts/run.sh    (optional sidecar — preserved verbatim)
//
// Legacy bare-markdown form (`spwn/skills/<name>.md`) is still
// accepted: the loader synthesises a `<name>/SKILL.md` entry (and
// injects minimal `name:`/`description:` frontmatter when the file
// has none) so downstream renderers can treat both forms identically.
type SkillSource struct {
	// Name is the skill identifier — either the sub-directory name
	// under spwn/skills/ (dir form) or the bare filename minus .md.
	Name string

	// Files is every file that belongs to the skill, keyed by path
	// relative to the skill's own root. `SKILL.md` is always
	// present; additional keys (e.g. "template.md",
	// "scripts/run.sh") cover sidecar content the skill author
	// wants shipped alongside the entry point.
	Files map[string][]byte
}

// HookSource is one runtime hook entry loaded from
// <root>/spwn/hooks/<name>.yaml. Runtime hooks fire on runtime-defined
// events (PreToolUse, UserPromptSubmit, SessionStart, …) inside the
// container — they are NOT host-side lifecycle scripts.
//
// One file = one hook. The filename (minus .yaml) is the hook's Name;
// each file's body is just `event: / matcher: / command:`. Per-agent
// selection happens via `hook/<name>` in agent.yaml#dependencies — an
// agent only receives hooks it explicitly declares, mirroring how
// `skill/<name>` and `tool/<name>` work.
type HookSource struct {
	// Name is a stable identifier used by the renderer to disambiguate
	// multiple hooks on the same event (e.g. two PreToolUse hooks).
	Name string
	// Event is one of Claude Code's / Codex's event names. The set is
	// small and shared across runtimes for the events both support;
	// runtime-specific events degrade gracefully (the renderer skips
	// events the target runtime doesn't know).
	Event string
	// Matcher narrows the event to a subset — e.g. `Bash` for
	// PreToolUse. Empty matches every instance of the event. The
	// value is written verbatim into the target config, so both
	// runtimes' glob/regex conventions are honoured.
	Matcher string
	// Command is the shell fragment the runtime runs when the hook
	// fires. V1 supports inline commands only; if a hook needs a
	// long script, users can invoke `bash /agents/<n>/scripts/foo.sh`
	// from here after materialising the script via a skill's sidecar.
	Command string
}

// ProfileSource is one file under spwn/profiles/.
type ProfileSource struct {
	Name    string
	Content []byte
}

// CommandSource is one slash-invoked prompt shortcut loaded from
// <root>/spwn/commands/<name>.md. The body becomes the prompt the
// runtime injects when the user types `/<name>` inside an agent
// session. Per-agent selection happens via `command/<name>` in
// agent.yaml#dependencies — an agent only ships the commands it
// explicitly subscribes to, mirroring how `skill/<name>`,
// `tool/<name>`, and `hook/<name>` work.
type CommandSource struct {
	// Name is the filename minus `.md` — used as the slash invocation
	// (e.g. `refactor.md` → `/refactor`).
	Name string
	// Body is the raw markdown that gets written verbatim into each
	// runtime's command directory. spwn does not parse frontmatter
	// here — runtimes that read frontmatter (Claude Code's
	// `description:`, `allowed-tools:`) consume it directly.
	Body []byte
}

// Load parses a spwn project rooted at projectRoot and returns a
// ProjectSource ready for a transpile.Runtime to consume.
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

	// Commands (optional slash-invoked prompts)
	commands, err := loadCommands(abs)
	if err != nil {
		return nil, err
	}
	src.Commands = commands

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

	// SOUL.md — identity body. Missing is fine (renderers that need
	// it surface a clear error later; tolerant-reads keep the loader
	// useful during scaffolding and partial-project states).
	soulPath := filepath.Join(dir, "SOUL.md")
	if b, err := os.ReadFile(soulPath); err == nil {
		src.Soul = b
	} else if !os.IsNotExist(err) {
		return src, fmt.Errorf("read %s: %w", soulPath, err)
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

// loadSkills walks <root>/spwn/skills/ and returns one SkillSource per
// directory-form skill plus one per legacy bare-markdown skill. Both
// forms coexist without conflict — a directory `foo/` and a file
// `foo.md` would produce two distinct SkillSource entries called "foo"
// which the renderer would then flag as a duplicate.
//
// Directory form accepts any sidecar layout; every file under the dir
// (including nested paths) is captured in SkillSource.Files keyed by
// its path relative to the skill root. SKILL.md is required; skills
// missing it are skipped silently (spwn check surfaces the warning).
//
// Bare form wraps the file as `<name>/SKILL.md`. If the markdown
// lacks YAML frontmatter we inject minimal `name:`/`description:`
// front-matter so the output is a valid Claude/Codex skill.
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
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			skill, ok, err := loadSkillDir(filepath.Join(dir, name), name)
			if err != nil {
				return nil, err
			}
			if ok {
				out = append(out, skill)
			}
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		id := strings.TrimSuffix(name, ".md")
		out = append(out, SkillSource{
			Name:  id,
			Files: map[string][]byte{"SKILL.md": ensureSkillFrontmatter(body, id)},
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// loadSkillDir reads every file under a skill directory into a
// SkillSource. Returns ok=false (without error) when SKILL.md is
// missing — treat those dirs as "not yet a skill" rather than failing
// the whole project load so half-authored skills don't block spwn check.
func loadSkillDir(root, name string) (SkillSource, bool, error) {
	files := map[string][]byte{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		body, readErr := os.ReadFile(p)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", p, readErr)
		}
		files[rel] = body
		return nil
	})
	if err != nil {
		return SkillSource{}, false, err
	}
	if _, ok := files["SKILL.md"]; !ok {
		return SkillSource{}, false, nil
	}
	files["SKILL.md"] = ensureSkillFrontmatter(files["SKILL.md"], name)
	return SkillSource{Name: name, Files: files}, true, nil
}

// ensureSkillFrontmatter injects a minimal `name:`/`description:`
// YAML front-matter block when one is absent. Claude Code and Codex
// both reject SKILL.md files without these fields; synthesising them
// lets legacy bare-markdown skills render without forcing authors to
// rewrite every file the day we ship this.
func ensureSkillFrontmatter(body []byte, name string) []byte {
	trimmed := strings.TrimLeft(string(body), "\n\t ")
	if strings.HasPrefix(trimmed, "---") {
		return body
	}
	header := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n", name, name)
	return append([]byte(header), body...)
}

// loadHooks walks <root>/spwn/hooks/*.yaml and returns one HookSource
// per file. Filename (minus .yaml) is the hook Name; the file body
// is the entry shape (`event:` / `matcher:` / `command:`). Missing
// directory → nil, nil. Malformed file surfaces as an error so
// `spwn check` can flag it before the user hits a silent spawn.
//
// Per-file schema:
//
//	# spwn/hooks/bash-audit.yaml
//	event: PreToolUse
//	matcher: Bash
//	command: echo "[audit] $CLAUDE_TOOL_INPUT"
//
// `event` is required and passed through verbatim; `matcher` is
// optional; `command` is the shell fragment that the runtime invokes
// when the hook fires. Keep V1 inline-only — if the shell fragment
// gets long, author it as a file under a skill and invoke it here.
//
// The legacy <root>/spwn/hooks.yaml form (one project-level file with
// a `hooks:` array) is rejected with a migration hint — `spwn check`
// surfaces this so users update before downstream silently drops their
// hooks.
func loadHooks(root string) ([]HookSource, error) {
	if legacy := filepath.Join(root, "spwn", "hooks.yaml"); fileExists(legacy) {
		return nil, fmt.Errorf(
			"%s: legacy hooks.yaml is no longer supported; "+
				"migrate each entry to its own file at spwn/hooks/<name>.yaml "+
				"(filename = hook name, body = event/matcher/command)",
			legacy,
		)
	}

	dir := filepath.Join(root, "spwn", "hooks")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	out := make([]HookSource, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		path := filepath.Join(dir, e.Name())
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil, fmt.Errorf("read %s: %w", path, rerr)
		}
		var parsed struct {
			Event   string `yaml:"event"`
			Matcher string `yaml:"matcher,omitempty"`
			Command string `yaml:"command"`
		}
		if uerr := yaml.Unmarshal(data, &parsed); uerr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, uerr)
		}
		if strings.TrimSpace(parsed.Event) == "" {
			return nil, fmt.Errorf("%s: missing `event`", path)
		}
		if strings.TrimSpace(parsed.Command) == "" {
			return nil, fmt.Errorf("%s: missing `command`", path)
		}
		out = append(out, HookSource{
			Name:    name,
			Event:   parsed.Event,
			Matcher: parsed.Matcher,
			Command: parsed.Command,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// loadCommands walks <root>/spwn/commands/*.md and returns one
// CommandSource per file. Filename minus `.md` is the slash command
// name (`refactor.md` → `/refactor`). Missing directory → nil, nil.
//
// Subdirectories and dotfiles are skipped so authors can drop helper
// files alongside the command markdown without spwn picking them up
// as commands. Body bytes pass through verbatim — runtimes that read
// frontmatter (Claude Code's `description:` / `allowed-tools:`)
// consume it directly without spwn parsing it.
func loadCommands(root string) ([]CommandSource, error) {
	dir := filepath.Join(root, "spwn", "commands")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	out := make([]CommandSource, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil, fmt.Errorf("read %s: %w", path, rerr)
		}
		out = append(out, CommandSource{
			Name: strings.TrimSuffix(name, ".md"),
			Body: body,
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
