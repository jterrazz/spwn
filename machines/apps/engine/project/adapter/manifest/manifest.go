package manifest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	ReadFile(path string) ([]byte, error)
	Readlink(path string) (string, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	Symlink(target, path string) error
	Chmod(path string, perm fs.FileMode) error
	Remove(path string) error
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

func groupings(src *source.ProjectSource) []string {
	g := source.WorldNames(src)
	if len(g) < 2 {
		return []string{""}
	}
	return g
}

// value: the harness an agent declares: its own, else the project's, else
// the catalog's first; never one picked by a sign-in.
func harnessOf(src *source.ProjectSource, agent string) string {
	for _, a := range src.Agents {
		if a.Name == agent && a.Config.Runtime.Backend != "" {
			return source.CanonicalRuntime(a.Config.Runtime.Backend)
		}
	}
	if src.Manifest != nil && src.Manifest.Runtime.Backend != "" {
		return source.CanonicalRuntime(src.Manifest.Runtime.Backend)
	}
	return catalog.Harnesses()[0]
}

// invariant: the Render hook of a harness kind: the files of one agent of
// a declaration, out of what that harness's renderer makes of it; pure.
func render(harness string, input transpile.Input, agent string) ([]project.File, error) {
	tree, err := transpile.Compile(harness, input)
	if err != nil {
		return nil, err
	}
	prefix := "agents/" + agent + "/"
	var out []project.File
	for _, p := range tree.Paths() {
		rest, ok := strings.CutPrefix(p, prefix)
		if !ok {
			continue
		}
		data, _ := tree.Get(p)
		out = append(out, project.File{Path: rest, Data: data, Mode: 0o644})
	}
	return out, nil
}

func (p Projects) Render(_ context.Context, root, agent string) ([]project.File, error) {
	// invariant: an agent is rendered in the first grouping that holds it;
	// one that no grouping holds renders nothing.
	return in(root, func() ([]project.File, error) {
		src, err := source.Load(root)
		if err != nil {
			return nil, err
		}
		for _, g := range groupings(src) {
			input, err := source.ToCompileInput(src, g)
			if err != nil {
				return nil, err
			}
			if slices.ContainsFunc(input.Agents, func(a transpile.AgentInput) bool { return a.Name == agent }) {
				return render(harnessOf(src, agent), input, agent)
			}
		}
		return nil, nil
	})
}

func (p Projects) Applied(_ context.Context, root string, paths []string) (record, disk project.Record, err error) {
	record = project.Record{}
	data, err := p.fs.ReadFile(filepath.Join(root, project.StateFolder, project.RecordName))
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, nil, err
	default:
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, nil, fmt.Errorf("%s/%s does not read: %w", project.StateFolder, project.RecordName, err)
		}
	}
	disk = project.Record{}
	for _, rel := range paths {
		f, ok, err := p.held(root, rel)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			disk[rel] = project.Digest(f)
		}
	}
	return record, disk, nil
}

// value: what the disk holds at rel under root, as a File; a link is read
// as its target, never followed.
func (p Projects) held(root, rel string) (project.File, bool, error) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	if target, err := p.fs.Readlink(full); err == nil {
		return project.File{Path: rel, Data: []byte(target), Link: true}, true, nil
	}
	info, err := p.fs.Stat(full)
	if errors.Is(err, fs.ErrNotExist) {
		return project.File{}, false, nil
	}
	if err != nil {
		return project.File{}, false, err
	}
	if !info.Mode().IsRegular() {
		return project.File{Path: rel, Data: []byte(info.Mode().Type().String())}, true, nil
	}
	data, err := p.fs.ReadFile(full)
	return project.File{Path: rel, Data: data, Mode: uint32(info.Mode().Perm())}, err == nil, err
}

func (p Projects) Write(_ context.Context, root string, changes []project.Change, record project.Record) error {
	for _, c := range changes {
		if !fs.ValidPath(c.Path) || !strings.HasPrefix(c.Path, project.StateFolder+"/") {
			return fmt.Errorf("%w: %s", project.ErrOutside, c.Path)
		}
		full := filepath.Join(root, filepath.FromSlash(c.Path))
		if c.Kind == project.Remove {
			if err := p.fs.Remove(full); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			continue
		}
		if err := p.fs.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if c.Kind == project.Overwrite {
			if err := p.fs.Remove(full); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
		if c.File.Link {
			if err := p.fs.Symlink(string(c.File.Data), full); err != nil {
				return err
			}
			continue
		}
		perm := fs.FileMode(c.File.Mode).Perm()
		if err := p.fs.WriteFile(full, c.File.Data, perm); err != nil {
			return err
		}
		if err := p.fs.Chmod(full, perm); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	state := filepath.Join(root, project.StateFolder)
	if err := p.fs.MkdirAll(state, 0o755); err != nil {
		return err
	}
	return p.fs.WriteFile(filepath.Join(state, project.RecordName), append(data, '\n'), 0o644)
}

// invariant: each agent is rendered by the Render hook of the harness it
// declares, each grouping of the declaration apart, and nothing is
// written; a harness that refuses is said once: an agent with
// no prompt, or that renders no file, is an issue of the plan. A harness
// the manifest's rule accepts is one whose renderer is linked.
func rendered(root string) []project.Issue {
	src, err := source.Load(root)
	if err != nil {
		return []project.Issue{{Level: project.Error, Path: root, Message: "the project's files do not read: " + err.Error(), Hint: "every file under spwn/ must be readable", Source: project.SourceCompile}}
	}
	var out []project.Issue
	prompts := map[string]bool{}
	for _, a := range src.Agents {
		prompts[a.Name] = len(a.AgentMD) > 0
	}
	for _, g := range groupings(src) {
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
		refused := map[string]bool{}
		for _, a := range input.Agents {
			harness := harnessOf(src, a.Name)
			if refused[harness] {
				continue
			}
			files, err := render(harness, input, a.Name)
			if err != nil {
				refused[harness] = true
				out = append(out, project.Issue{Level: project.Error, Path: "compile:" + harness, Message: err.Error(), Hint: "the harness's renderer refused the project", Source: project.SourceCompile})
				continue
			}
			if len(files) == 0 {
				out = append(out, project.Issue{Level: project.Warning, Path: "agents/" + a.Name, Message: "agent renders no file",
					Hint: "spwn/agents/" + a.Name + "/ lacks its AGENTS.md or its agent.yaml", Source: project.SourceCompile})
			}
		}
	}
	return out
}
