package manifest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
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
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
)

type disk interface {
	MkdirAll(path string, perm fs.FileMode) error
	WriteFile(path string, data []byte, perm fs.FileMode) error
}

type Projects struct {
	// value: the disk of the machine a new project's folder is made on.
	fs disk
	// value: every run folder and the record, reached from the project's
	// root one name at a time, never through a link.
	under local.Beneath
}

var _ contract.Projects = Projects{}

func New(fsys disk) Projects { return Projects{fs: fsys} }

// value: err as the project names it: the path from root it is about, a
// link or a file of two names there said as project.ErrLink.
func unwritten(root, rel string, err error) error {
	var at *fs.PathError
	if errors.As(err, &at) {
		if inside, relErr := filepath.Rel(root, at.Path); relErr == nil && !strings.HasPrefix(inside, "..") {
			rel = filepath.ToSlash(inside)
		}
		err = at.Err
	}
	if errors.Is(err, local.ErrLink) || errors.Is(err, local.ErrHardLink) {
		err = project.ErrLink
	}
	return &project.Unwritten{Path: rel, Err: err}
}

// value: a read's err, named as unwritten names it when it is a link; any
// other failure to read as it is.
func linked(root, rel string, err error) error {
	if errors.Is(err, local.ErrLink) || errors.Is(err, local.ErrHardLink) {
		return unwritten(root, rel, err)
	}
	return err
}

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

var manifestName = regexp.MustCompile(`(?m)^name:.*$`)

// invariant: an example's files are copied as they are, its manifest
// renamed to the project's name; the folder is made before any file.
func (p Projects) copyExample(root, name, example string) error {
	files, ok := catalog.Example(example)
	if !ok {
		return fmt.Errorf("no example named %q", example)
	}
	return p.copyFolder(files, ".", root, name)
}

func (p Projects) copyFolder(files fs.FS, dir, root, name string) error {
	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		at := e.Name()
		if dir != "." {
			at = dir + "/" + at
		}
		target := filepath.Join(root, filepath.FromSlash(at))
		if e.IsDir() {
			if err := p.fs.MkdirAll(target, 0o755); err != nil {
				return err
			}
			if err := p.copyFolder(files, at, root, name); err != nil {
				return err
			}
			continue
		}
		data, err := fs.ReadFile(files, at)
		if err != nil {
			return err
		}
		if at == project.ManifestName {
			data = manifestName.ReplaceAll(data, []byte("name: "+name))
		}
		if err := p.fs.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (p Projects) Create(_ context.Context, folder, name, harness, example string) (project.Project, error) {
	root := filepath.Join(folder, name)
	return in(root, func() (project.Project, error) {
		switch _, err := p.under.Lstat(folder, name); {
		case err == nil:
			return project.Project{}, project.ErrExists
		case !errors.Is(err, fs.ErrNotExist):
			return project.Project{}, err
		}
		if err := p.fs.MkdirAll(root, 0o755); err != nil {
			return project.Project{}, err
		}
		if example != "" {
			if err := p.copyExample(root, name, example); err != nil {
				return project.Project{}, err
			}
			if err := spwnproject.AppendGitignore(root); err != nil {
				return project.Project{}, err
			}
		} else if err := spwnproject.Init(root, spwnproject.InitOpts{Name: name, Backend: "spwn:" + harness}); err != nil {
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

const recordPath = project.StateFolder + "/" + project.RecordName

// value: the folder dir of root, a slash path from it, opened one name at
// a time; "" is root itself, the caller's to trust.
func (p Projects) open(root, dir string) (*local.Folder, error) {
	if dir == "" {
		return p.under.Open(filepath.Dir(root), filepath.Base(root))
	}
	return p.under.Open(root, dir)
}

// value: the record apply keeps, and its bytes as read, for a write that
// replaces it only while it still holds them.
func (p Projects) record(root string) (project.Record, local.Seen, error) {
	record := project.Record{}
	f, err := p.open(root, project.StateFolder)
	if errors.Is(err, fs.ErrNotExist) {
		return record, local.Seen{}, nil
	}
	if err != nil {
		return nil, local.Seen{}, linked(root, project.StateFolder, err)
	}
	defer f.Close()
	data, _, err := f.ReadFile(project.RecordName)
	if errors.Is(err, fs.ErrNotExist) {
		return record, local.Seen{}, nil
	}
	if err != nil {
		return nil, local.Seen{}, linked(root, recordPath, err)
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, local.Seen{}, fmt.Errorf("%s does not read: %w", recordPath, err)
	}
	return record, local.Seen{Data: data, There: true}, nil
}

func (p Projects) Applied(_ context.Context, root string, paths []string) (record, disk project.Record, err error) {
	if record, _, err = p.record(root); err != nil {
		return nil, nil, err
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

// value: what the disk holds at rel under root, as a File; a link at its
// own name is read as its target, never followed, and one on a folder of
// its path is refused (project.ErrLink).
func (p Projects) held(root, rel string) (project.File, bool, error) {
	dir, name := path.Split(rel)
	f, err := p.open(root, strings.TrimSuffix(dir, "/"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return project.File{}, false, nil
	case err != nil:
		return project.File{}, false, linked(root, rel, err)
	}
	defer f.Close()
	e, err := f.Lstat(name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return project.File{}, false, nil
	case err != nil:
		return project.File{}, false, linked(root, rel, err)
	case e.Mode&fs.ModeSymlink != 0:
		target, readErr := f.Readlink(name)
		if readErr != nil {
			return project.File{}, false, linked(root, rel, readErr)
		}
		return project.File{Path: rel, Data: []byte(target), Link: true}, true, nil
	case !e.Mode.IsRegular():
		return project.File{Path: rel, Data: []byte(e.Mode.Type().String())}, true, nil
	}
	data, e, err := f.ReadFile(name)
	if err != nil {
		return project.File{}, false, linked(root, rel, err)
	}
	return project.File{Path: rel, Data: data, Mode: uint32(e.Mode.Perm())}, true, nil
}

func (p Projects) folders(root, dir string) error {
	return p.under.Folders(root, dir, 0o755, local.Self())
}

func (p Projects) Write(_ context.Context, root string, changes []project.Change, record project.Record) error {
	// invariant: a change apply could not make stops it there, and the
	// record it keeps is that of the changes it made, so the next apply
	// owns them and starts again where this one stopped.
	for _, c := range changes {
		if !project.InRunFolder(c.Path) {
			return fmt.Errorf("%w: %s", project.ErrOutside, c.Path)
		}
	}
	was, _, err := p.record(root)
	if err != nil {
		return err
	}
	for i, c := range changes {
		if err := p.change(root, c); err != nil {
			failed := unwritten(root, c.Path, err)
			if kept := p.keep(root, project.Kept(was, record, changes, i)); kept != nil {
				return errors.Join(failed, kept)
			}
			return failed
		}
	}
	return p.keep(root, record)
}

func (p Projects) change(root string, c project.Change) error {
	dir, name := path.Split(c.Path)
	dir = strings.TrimSuffix(dir, "/")
	if c.Kind != project.Remove {
		if err := p.folders(root, dir); err != nil {
			return err
		}
	}
	f, err := p.open(root, dir)
	if c.Kind == project.Remove && errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	return made(f, name, c)
}

// invariant: name made what c says, through the folder's descriptor: a
// file over a file is replaced at once, a link is never followed, and
// anything else there is taken away first, a file only while it holds
// what was just read.
func made(f *local.Folder, name string, c project.Change) error {
	e, err := f.Lstat(name)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	held, link := err == nil, err == nil && e.Mode&fs.ModeSymlink != 0
	seen := local.Seen{}
	switch {
	case link && (!c.File.Link || c.Kind == project.Remove):
		if err := f.RemoveLink(name); err != nil {
			return err
		}
	case held && !link:
		data, _, err := f.ReadFile(name)
		if err != nil {
			return err
		}
		seen = local.Seen{Data: data, There: true}
		if c.Kind == project.Remove || c.File.Link {
			if err := f.Remove(name, seen); err != nil {
				return err
			}
		}
	}
	switch {
	case c.Kind == project.Remove:
		return nil
	case c.File.Link:
		return f.Link(name, string(c.File.Data))
	}
	perm := fs.FileMode(c.File.Mode).Perm()
	if err := f.Replace(name, seen, c.File.Data, perm, local.Self()); err != nil {
		return err
	}
	return f.ChmodFile(name, local.Self().UID, func(fs.FileMode) fs.FileMode { return perm })
}

// value: record kept as what apply wrote, replacing the one read only
// while it still holds what was read.
func (p Projects) keep(root string, record project.Record) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	_, seen, err := p.record(root)
	if err != nil {
		return err
	}
	if err = p.folders(root, project.StateFolder); err != nil {
		return unwritten(root, recordPath, err)
	}
	f, err := p.open(root, project.StateFolder)
	if err != nil {
		return unwritten(root, recordPath, err)
	}
	defer f.Close()
	if err := f.Replace(project.RecordName, seen, append(data, '\n'), 0o644, local.Self()); err != nil {
		return unwritten(root, recordPath, err)
	}
	return nil
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
		prompts[a.Name] = strings.TrimSpace(string(a.AgentMD)) != ""
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
