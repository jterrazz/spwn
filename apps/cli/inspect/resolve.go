package inspect

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ib "spwn.sh/packages/compile"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/dependency/tool"


	"spwn.sh/packages/project"
	"spwn.sh/packages/runtimes"
	"spwn.sh/packages/transpile/source"
	wmodels "spwn.sh/packages/world/models"
)

// Opts configures Build. All fields are optional.
type Opts struct {
	// Agent restricts the output to the named agent. Empty means
	// "every deployable agent in this project".
	Agent string

	// LiveStatus, when true, queries the architect for live world
	// state. When false (or when the architect is unreachable),
	// every agent is shown with status StatusUnknown and the
	// renderer degrades to "○ stopped".
	LiveStatus bool
}

// Build walks the project rooted at cwd and returns a Model ready
// for Render. No disk writes, no Docker calls except the optional
// live-status lookup (gated by Opts.LiveStatus). Every failure is
// an error; missing agents and empty worlds are not.
func Build(cwd string, opts Opts) (*Model, error) {
	proj, err := project.Find(cwd)
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}
	if proj == nil {
		return nil, fmt.Errorf("no spwn.yaml found in %s or any parent — run `spwn init` first", cwd)
	}

	src, err := source.Load(proj.Root)
	if err != nil {
		return nil, err
	}

	reg := ib.NewRegistry()
	if err := dependency.RegisterBuiltins(reg); err != nil {
		return nil, fmt.Errorf("register catalog: %w", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		return nil, fmt.Errorf("register runtimes: %w", err)
	}
	if err := registerLocalTools(reg, proj.Root); err != nil {
		return nil, fmt.Errorf("register local tools: %w", err)
	}

	// world-name → Status for every running agent.
	var statusByAgent map[string]Status
	var worldByAgent map[string]string
	if opts.LiveStatus {
		statusByAgent, worldByAgent = liveAgentStatus()
	}

	projectDeps := src.Manifest.Deps

	var views []AgentView
	for _, a := range src.Agents {
		if opts.Agent != "" && a.Name != opts.Agent {
			continue
		}

		agentDeps := a.Config.Deps
		fullDeps := dedupStrings(append(append([]string{}, projectDeps...), agentDeps...))
		directCount := len(fullDeps)

		seen := map[string]struct{}{}
		var roots []DepNode
		for _, ref := range fullDeps {
			roots = append(roots, buildNode(reg, ref, seen))
		}
		transitiveCount := 0
		for name := range seen {
			if !containsString(fullDeps, name) {
				transitiveCount++
			}
		}

		world := findWorldFor(src.Manifest, a.Name)
		status := StatusUnknown
		if opts.LiveStatus {
			if s, ok := statusByAgent[a.Name]; ok {
				status = s
			}
			// If the agent's world isn't tracked but a live world
			// with the same config name is, use it as a tiebreaker.
			if status == StatusUnknown {
				if wname, ok := worldByAgent[a.Name]; ok {
					world = wname
					status = StatusRunning
				}
			}
		}

		view := AgentView{
			Name:                a.Name,
			Role:                defaultRole(a.Config.Role),
			Runtime:             shortRuntime(a.Config.Runtime.Backend),
			World:               world,
			Status:              status,
			Deps:                roots,
			DirectDepsCount:     directCount,
			TransitiveDepsCount: transitiveCount,
			Skills:              collectSkills(a, src, fullDeps, reg),
			Hooks:               collectHooks(src, a),
		}
		views = append(views, view)
	}

	// Stable order: alphabetical by name so two runs against the
	// same tree produce identical output.
	sort.Slice(views, func(i, j int) bool { return views[i].Name < views[j].Name })

	if opts.Agent != "" && len(views) == 0 {
		return nil, fmt.Errorf("agent %q not found in this project", opts.Agent)
	}

	return &Model{Agents: views}, nil
}

// buildNode recursively constructs a DepNode. `seen` carries the
// set of dep names already emitted at any depth — the second (and
// further) visits return a DedupSeen node (cargo-tree `(*)` marker)
// with no children.
func buildNode(reg *ib.Registry, ref string, seen map[string]struct{}) DepNode {
	if _, already := seen[ref]; already {
		// Still resolve version/kind for the display, but short-circuit children.
		n := DepNode{Name: ref, DedupSeen: true}
		if t := reg.Get(ref); t != nil {
			n.Version = t.Version()
			n.Kind = t.Kind()
		}
		return n
	}
	seen[ref] = struct{}{}

	t := reg.Get(ref)
	if t == nil {
		// Unknown ref — render as-is with no composition. Lets us
		// show `spwn inspect` even when `spwn check` would flag the
		// ref as missing.
		return DepNode{Name: ref}
	}
	node := DepNode{
		Name:    t.Name(),
		Version: t.Version(),
		Kind:    t.Kind(),
		Skills:  countSkills(t.Skills()),
		Config:  len(t.Runtimes()) > 0,
	}
	for _, child := range t.Dependencies() {
		node.Children = append(node.Children, buildNode(reg, child, seen))
	}
	return node
}

func countSkills(fsys fs.FS) int {
	if fsys == nil {
		return 0
	}
	n := 0
	_ = fs.WalkDir(fsys, ".", func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			n++
		}
		return nil
	})
	return n
}

// registerLocalTools walks <root>/spwn/tools/ and registers every
// directory-form dependency it finds, prefixing names with "local:"
// to stay out of the spwn: namespace (mirrors the hydration path
// in packages/architect/localtools.go).
func registerLocalTools(reg *ib.Registry, root string) error {
	toolsDir := filepath.Join(root, "spwn", "tools")
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// dependency.LoadLocalTool reads tool.yaml + skills/ + files/ and wraps
		// The result as a tool.Tool — the same path the spawn
		// Pipeline takes. Missing tool.yaml is not fatal; users may
		// Author the dir before filling it in, so we skip silently.
		tool, err := dependency.LoadLocalTool(root, e.Name())
		if err != nil {
			continue
		}
		wrapped := &localToolAdapter{inner: tool, name: e.Name()}
		_ = reg.Register(wrapped)
	}
	return nil
}

// localToolAdapter forces Name() to a bare basename (no @-scope) so
// the renderer prints `my-parser` instead of `local:my-parser`.
type localToolAdapter struct {
	inner tool.Tool
	name  string
}

func (t *localToolAdapter) Name() string                 { return t.name }
func (t *localToolAdapter) Kind() tool.Kind        { return t.inner.Kind() }
func (t *localToolAdapter) Version() string              { return t.inner.Version() }
func (t *localToolAdapter) Dependencies() []string       { return t.inner.Dependencies() }
func (t *localToolAdapter) Install() tool.InstallSpec      { return t.inner.Install() }
func (t *localToolAdapter) Verify() []string             { return t.inner.Verify() }
func (t *localToolAdapter) Skills() fs.FS                { return t.inner.Skills() }
func (t *localToolAdapter) Runtimes() []string           { return t.inner.Runtimes() }
func (t *localToolAdapter) Config(runtime string) []byte { return t.inner.Config(runtime) }

// findWorldFor returns the first world name that lists the agent.
// Returns "" when no world claims it (orphan agent).
func findWorldFor(m *project.Manifest, agentName string) string {
	if m == nil {
		return ""
	}
	// Deterministic order — a project may declare the same agent
	// under multiple worlds; return the alphabetically-first.
	var names []string
	for name := range m.Worlds {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		w := m.Worlds[name]
		for _, ag := range w.Agents {
			if ag == agentName {
				return name
			}
		}
	}
	return ""
}

// liveAgentStatus queries the architect for current world state and
// maps every live agent name to its world's status. Unreachable
// architect (typical in CI / tests) returns empty maps so callers
// fall back to StatusUnknown without error.
func liveAgentStatus() (map[string]Status, map[string]string) {
	arc, err := architect.NewFromEnv()
	if err != nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	worlds, err := arc.List(ctx)
	if err != nil {
		return nil, nil
	}
	statusByAgent := map[string]Status{}
	worldByAgent := map[string]string{}
	for _, w := range worlds {
		s := fromWorldStatus(w.Status)
		if w.Agent != "" {
			statusByAgent[w.Agent] = s
			worldByAgent[w.Agent] = w.Config
		}
		for _, a := range w.Agents {
			statusByAgent[a.Name] = s
			worldByAgent[a.Name] = w.Config
		}
	}
	return statusByAgent, worldByAgent
}

func fromWorldStatus(s wmodels.Status) Status {
	switch s {
	case wmodels.StatusRunning, wmodels.StatusCreating:
		return StatusRunning
	case wmodels.StatusIdle:
		return StatusIdle
	default:
		return StatusStopped
	}
}

// collectSkills enumerates every markdown skill the agent will see
// at spawn-time: project-local (spwn/skills/) and tool-provided
// (spwn:<tool>/skills/ or my-tool/skills/).
//
// Per-agent skill directories (spwn/agents/<name>/skills/) are the
// agent's Mind memory layer — written to at runtime — and are
// deliberately NOT enumerated here: spwn does not discover, inject,
// or surface them as composable skills.
func collectSkills(a source.AgentSource, src *source.ProjectSource, fullDeps []string, reg *ib.Registry) []SkillRef {
	_ = a // agent-local skills/ is an opaque Mind memory layer
	var out []SkillRef

	// Project-wide bare-markdown skills.
	for _, s := range src.Skills {
		out = append(out, SkillRef{Name: s.Name, Origin: "spwn/skills"})
	}

	// Tool-provided skills (via each dep's Skills() fs.FS). Walk
	// the tree with dedup so a skill doesn't appear twice when two
	// agents share a tool ancestor.
	seen := map[string]struct{}{}
	for _, ref := range fullDeps {
		collectToolSkills(reg, ref, seen, &out)
	}

	sortSkills(out)
	return out
}

func collectToolSkills(reg *ib.Registry, ref string, seen map[string]struct{}, out *[]SkillRef) {
	if _, ok := seen[ref]; ok {
		return
	}
	seen[ref] = struct{}{}
	t := reg.Get(ref)
	if t == nil {
		return
	}
	if fsys := t.Skills(); fsys != nil {
		_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				*out = append(*out, SkillRef{Name: p, Origin: t.Name()})
			}
			return nil
		})
	}
	for _, dep := range t.Dependencies() {
		collectToolSkills(reg, dep, seen, out)
	}
}

func collectHooks(src *source.ProjectSource, a source.AgentSource) []HookRef {
	_ = a // per-agent hooks not distinguished from project today
	out := make([]HookRef, 0, len(src.Hooks))
	for _, h := range src.Hooks {
		out = append(out, HookRef{Name: h.Name, Origin: "spwn/hooks"})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Origin != out[j].Origin {
			return out[i].Origin < out[j].Origin
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// shortRuntime strips the spwn scope so "claude-code" shows instead
// of "spwn:claude-code" or "spwn:claude-code" in the header. Falls
// back to the raw value when the ref has no recognised prefix.
func shortRuntime(backend string) string {
	if strings.HasPrefix(backend, "spwn:") {
		return backend[len("spwn:"):]
	}
	if strings.HasPrefix(backend, "spwn:") {
		return backend[len("spwn:"):]
	}
	return backend
}

func defaultRole(r string) string {
	if r == "" {
		return "worker"
	}
	return r
}

func dedupStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
