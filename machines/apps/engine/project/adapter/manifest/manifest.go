package manifest

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"spwn.sh/packages/platform"
	spwnproject "spwn.sh/packages/project"
	"spwn.sh/packages/runtimes"
	// invariant: each harness's renderer registers itself once linked.
	_ "spwn.sh/packages/runtimes/claudecode"
	// invariant: each harness's renderer registers itself once linked.
	_ "spwn.sh/packages/runtimes/codex"
	"spwn.sh/packages/transpile"
	"spwn.sh/packages/transpile/source"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
)

type disk interface {
	Stat(path string) (fs.FileInfo, error)
	MkdirAll(path string, perm fs.FileMode) error
}

type Projects struct {
	// value: the disk of the machine a new project's folder is made on.
	fs disk
}

var _ contract.Projects = Projects{}

func New(fsys disk) Projects { return Projects{fs: fsys} }

// invariant: Agents' packages resolve paths from one process-global
// project root, which platform.WithProjectRoot holds for one act.
func in[T any](root string, act func() (T, error)) (T, error) {
	return platform.WithProjectRoot(root, act)
}

func load(root string) (*spwnproject.Project, error) {
	return spwnproject.Load(filepath.Join(root, project.ManifestName))
}

func (p Projects) Find(_ context.Context, folder string) (string, error) {
	return in("", func() (string, error) {
		found, err := spwnproject.Find(folder)
		if err != nil {
			return "", err
		}
		if found == nil {
			return "", project.ErrNotFound
		}
		return found.Root, nil
	})
}

func shown(found *spwnproject.Project) project.Project {
	out := project.Project{Name: found.Manifest.Name, Root: found.Root, Manifest: found.ManifestPath, Agents: []string{}}
	for _, a := range slices.Concat(found.Agents, found.OrphanAgents) {
		if a.Exists && !slices.Contains(out.Agents, a.Name) {
			out.Agents = append(out.Agents, a.Name)
		}
	}
	slices.Sort(out.Agents)
	return out
}

func (p Projects) Show(_ context.Context, root string) (project.Project, error) {
	return in(root, func() (project.Project, error) {
		found, err := load(root)
		if err != nil {
			return project.Project{}, err
		}
		return shown(found), nil
	})
}

func (p Projects) Create(_ context.Context, folder, name, harness string) (project.Project, error) {
	root := filepath.Join(folder, name)
	return in(root, func() (project.Project, error) {
		if _, err := p.fs.Stat(root); err == nil {
			return project.Project{}, project.ErrExists
		}
		if err := p.fs.MkdirAll(root, 0o755); err != nil {
			return project.Project{}, err
		}
		if err := spwnproject.Init(root, spwnproject.InitOpts{Name: name, Backend: "spwn:" + harness}); err != nil {
			return project.Project{}, err
		}
		found, err := load(root)
		if err != nil {
			return project.Project{}, err
		}
		return shown(found), nil
	})
}

func supported() []string {
	names := runtimes.Names()
	out := make([]string, 0, 2*len(names))
	for _, name := range names {
		out = append(out, name, "spwn:"+name)
	}
	return out
}

func issueOf(i spwnproject.Issue, src string) project.Issue {
	return project.Issue{Level: project.Level(i.Level.String()), Path: i.Path, Message: i.Message, Hint: i.Hint, Source: src}
}

func (p Projects) Plan(_ context.Context, root string) (project.Plan, error) {
	return in(root, func() (project.Plan, error) {
		found, err := load(root)
		if err != nil {
			return project.Plan{}, err
		}
		// invariant: the built-in catalog of the old program is not the
		// engine's, so a reference to one of its entries is not judged here.
		var issues []project.Issue
		for _, i := range spwnproject.Validate(found, spwnproject.ValidateOpts{SupportedRuntimes: supported(), HookEventsByRuntime: transpile.HookEventsByRuntime()}) {
			issues = append(issues, issueOf(i, project.SourceManifest))
		}
		issues = append(issues, rendered(root)...)
		return project.PlanOf(found.Root, found.ManifestPath, issues), nil
	})
}

func (p Projects) Drift(_ context.Context, root string) (project.Drift, error) {
	return in(root, func() (project.Drift, error) {
		found, err := load(root)
		if err != nil {
			return project.Drift{}, err
		}
		d := project.Drift{Root: found.Root, Manifest: found.ManifestPath, Rows: []project.Issue{}}
		for _, i := range spwnproject.Drift(found) {
			d.Rows = append(d.Rows, issueOf(i, project.SourceLockfile))
		}
		return d, nil
	})
}

// invariant: each agent is rendered for the catalog's default harness, each
// grouping of the declaration apart, and nothing is written: an agent with
// no prompt, or that renders no file, is an issue of the plan, and so is a
// harness no renderer is registered for.
func rendered(root string) []project.Issue {
	src, err := source.Load(root)
	if err != nil {
		return []project.Issue{{Level: project.Error, Path: root, Message: "the project's files do not read: " + err.Error(), Hint: "every file under spwn/ must be readable", Source: project.SourceCompile}}
	}
	out := harnessIssues(src)
	groupings := source.WorldNames(src)
	if len(groupings) < 2 {
		groupings = []string{""}
	}
	prompts := map[string]bool{}
	for _, a := range src.Agents {
		prompts[a.Name] = len(a.AgentMD) > 0
	}
	harness := catalog.Harnesses()[0]
	for _, g := range groupings {
		input, err := source.ToCompileInput(src, g)
		if err != nil {
			out = append(out, project.Issue{Level: project.Warning, Path: "spwn.yaml", Message: "not rendered: " + err.Error(), Source: project.SourceCompile})
			continue
		}
		for _, a := range input.Agents {
			if have, ok := prompts[a.Name]; ok && !have {
				out = append(out, project.Issue{Level: project.Error, Path: "spwn/agents/" + a.Name + "/AGENTS.md", Message: "agent prompt is missing or empty",
					Hint: "write spwn/agents/" + a.Name + "/AGENTS.md with the agent's system prompt", Source: project.SourceCompile})
			}
		}
		tree, err := transpile.Compile(harness, input)
		if err != nil {
			out = append(out, project.Issue{Level: project.Error, Path: "compile:" + harness, Message: err.Error(), Hint: "the harness's renderer refused the project", Source: project.SourceCompile})
			continue
		}
		for _, a := range input.Agents {
			prefix := "agents/" + a.Name + "/"
			if !slices.ContainsFunc(tree.Paths(), func(p string) bool { return strings.HasPrefix(p, prefix) }) {
				out = append(out, project.Issue{Level: project.Warning, Path: "agents/" + a.Name, Message: "agent renders no file",
					Hint: "spwn/agents/" + a.Name + "/ lacks its AGENTS.md or its agent.yaml", Source: project.SourceCompile})
			}
		}
	}
	return out
}

func harnessIssues(src *source.ProjectSource) []project.Issue {
	registered := transpile.RegisteredRuntimes()
	var out []project.Issue
	for _, a := range src.Agents {
		raw := a.Config.Runtime.Backend
		if raw == "" || slices.Contains(registered, strings.TrimPrefix(raw, "spwn:")) {
			continue
		}
		out = append(out, project.Issue{Level: project.Warning, Path: "spwn/agents/" + a.Name + "/agent.yaml#runtime.backend",
			Message: "harness " + raw + " has no renderer", Hint: "one of " + strings.Join(registered, ", "), Source: project.SourceCompile})
	}
	return out
}
