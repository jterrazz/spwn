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

	"spwn.sh/packages/architect"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/dependency/resolver"
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
		return nil, fmt.Errorf("no spwn.yaml found in %s or any parent: run `spwn init` first", cwd)
	}

	src, err := source.Load(proj.Root)
	if err != nil {
		return nil, err
	}

	reg := resolver.NewRegistry()
	if err := dependency.RegisterBuiltins(reg); err != nil {
		return nil, fmt.Errorf("register catalog: %w", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		return nil, fmt.Errorf("register runtimes: %w", err)
	}
	if err := registerLocalTools(reg, proj.Root); err != nil {
		return nil, fmt.Errorf("register local tools: %w", err)
	}

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

	sort.Slice(views, func(i, j int) bool { return views[i].Name < views[j].Name })

	if opts.Agent != "" && len(views) == 0 {
		return nil, fmt.Errorf("agent %q not found in this project", opts.Agent)
	}

	return &Model{Agents: views}, nil
}

func buildNode(reg *resolver.Registry, ref string, seen map[string]struct{}) DepNode {
	if _, already := seen[ref]; already {
		n := DepNode{Name: ref, DedupSeen: true}
		if t := reg.Get(ref); t != nil {
			n.Version = t.Version()
		}
		return n
	}
	seen[ref] = struct{}{}

	t := reg.Get(ref)
	if t == nil {
		return DepNode{Name: ref}
	}
	node := DepNode{
		Name:    t.Name(),
		Version: t.Version(),
		Skills:  countSkills(t.Skills()),
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

func registerLocalTools(reg *resolver.Registry, root string) error {
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
		tool, err := dependency.LoadLocalTool(root, e.Name())
		if err != nil {
			continue
		}
		wrapped := &localToolAdapter{inner: tool, name: e.Name()}
		_ = reg.Register(wrapped)
	}
	return nil
}

type localToolAdapter struct {
	inner tool.Tool
	name  string
}

func (t *localToolAdapter) Name() string              { return t.name }
func (t *localToolAdapter) Version() string           { return t.inner.Version() }
func (t *localToolAdapter) Dependencies() []string    { return t.inner.Dependencies() }
func (t *localToolAdapter) Install() tool.InstallSpec { return t.inner.Install() }
func (t *localToolAdapter) Verify() []string          { return t.inner.Verify() }
func (t *localToolAdapter) Skills() fs.FS             { return t.inner.Skills() }

func findWorldFor(m *project.Manifest, agentName string) string {
	if m == nil {
		return ""
	}
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

func collectSkills(_ source.AgentSource, src *source.ProjectSource, fullDeps []string, reg *resolver.Registry) []SkillRef {
	var out []SkillRef

	for _, s := range src.Skills {
		out = append(out, SkillRef{Name: s.Name, Origin: "spwn/skills"})
	}

	seen := map[string]struct{}{}
	for _, ref := range fullDeps {
		collectToolSkills(reg, ref, seen, &out)
	}

	sortSkills(out)
	return out
}

func collectToolSkills(reg *resolver.Registry, ref string, seen map[string]struct{}, out *[]SkillRef) {
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
	_ = a
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
