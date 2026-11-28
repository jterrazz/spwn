package manifest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
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

type Projects struct {
	// value: a new project's folder and files, every run folder and the
	// record, reached one name at a time, never through a link.
	under local.Beneath
}

var _ contract.Projects = Projects{}

func New() Projects { return Projects{} }

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
	switch {
	case errors.Is(err, local.ErrLink) || errors.Is(err, local.ErrHardLink):
		err = project.ErrLink
	case errors.Is(err, local.ErrNotFolder):
		err = project.ErrNotAFolder
	}
	return &project.Unwritten{Path: rel, Err: err}
}

// value: a read's err, named as unwritten names it when it is a link or a
// file on a folder's way; any other failure to read as it is.
func linked(root, rel string, err error) error {
	if errors.Is(err, local.ErrLink) || errors.Is(err, local.ErrHardLink) || errors.Is(err, local.ErrNotFolder) {
		return unwritten(root, rel, err)
	}
	return err
}

// invariant: Agents' packages resolve paths from one process-global
// project, its root and its folders, which platform.WithProject holds for
// one act; a root whose spwn.yaml does not read yet, the folder Create is
// about to make, is held with the default folders, and its walk refuses
// what its layout refuses before any of it is read.
func in[T any](root string, act func() (T, error)) (T, error) {
	l, err := spwnproject.LayoutAt(root)
	if err != nil {
		l = spwnproject.DefaultLayout()
	}
	return platform.WithProject(root, l.Platform(), act)
}

// invariant: the declaration is walked, one name at a time and no link
// followed, before any of it is read: the root module's loaders read by
// path, so a link there would bring an outside file into what a harness
// renders, and a FIFO would hold the read. spwn.yaml and spwn.lock are
// walked first as files, then spwn.yaml's layout is read through a root at
// the project, then only the folders it names: nothing else at the root is
// walked or listed. root itself may be a link.
func (p Projects) declared(root string) error {
	files := []string{project.ManifestName, project.LockName}
	for _, rel := range files {
		if err := p.walk(root, rel, true); err != nil {
			return err
		}
	}
	l, err := spwnproject.LayoutAt(root)
	if err != nil {
		return refusedLayout(err)
	}
	for _, rel := range project.Declaration(l.Folders())[len(files):] {
		if err := p.walk(root, rel, false); err != nil {
			return err
		}
	}
	return nil
}

// value: err as the project names a layout spwn.yaml refuses: the old
// layout's spwn/ folder, or a folders: entry naming no folder; any other
// failure to read spwn.yaml as it is.
func refusedLayout(err error) error {
	var old *spwnproject.OldLayoutError
	var folders *spwnproject.LayoutError
	switch {
	case errors.As(err, &old):
		return &project.LayoutRefused{Old: true, Said: old.Message()}
	case errors.As(err, &folders):
		return &project.LayoutRefused{Key: folders.Key, Said: folders.Message()}
	}
	return err
}

// value: nil when rel and every name under it is a file or a folder, a
// file alone when file; else the first that is not, named from root.
func (p Projects) walk(root, rel string, file bool) error {
	e, err := p.under.Lstat(root, rel)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return unwritten(root, rel, err)
	case e.Mode&fs.ModeSymlink != 0:
		return &project.Unwritten{Path: rel, Err: project.ErrLink}
	case e.Mode.IsRegular():
		return nil
	case file || !e.Mode.IsDir():
		return &project.Unwritten{Path: rel, Err: project.ErrNotAFile}
	}
	f, err := p.under.Open(root, rel)
	if err != nil {
		return unwritten(root, rel, err)
	}
	names, err := f.Names()
	f.Close()
	if err != nil {
		return unwritten(root, rel, err)
	}
	for _, name := range names {
		if err := p.walk(root, rel+"/"+name, false); err != nil {
			return err
		}
	}
	return nil
}

func (p Projects) load(root string) (*spwnproject.Project, error) {
	if err := p.declared(root); err != nil {
		return nil, err
	}
	return spwnproject.Load(filepath.Join(root, project.ManifestName))
}

func (p Projects) Find(_ context.Context, folder string) (string, error) {
	// invariant: the root is the nearest folder from folder up whose
	// spwn.yaml is there as anything but a folder; a link there is found,
	// so that its project is refused rather than one further up.
	for dir := filepath.Clean(folder); ; dir = filepath.Dir(dir) {
		if e, err := p.under.Lstat(dir, project.ManifestName); err == nil && !e.Mode.IsDir() {
			if err := p.declared(dir); err != nil {
				return "", err
			}
			return dir, nil
		}
		if filepath.Dir(dir) == dir {
			return "", project.ErrNotFound
		}
	}
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
		found, err := p.load(root)
		if err != nil {
			return project.Project{}, err
		}
		return shown(found), nil
	})
}

var manifestName = regexp.MustCompile(`(?m)^name:.*$`)

// value: the files a project named name starts with, by slash path from
// its root: an example's as they are, its manifest renamed to name, or
// else the scaffold's for harness; and its .gitignore holding `.spwn/`.
func starting(name, harness, example string) (map[string][]byte, error) {
	if example == "" {
		files, err := spwnproject.Scaffold(name, "spwn:"+harness)
		if err != nil {
			return nil, err
		}
		files[".gitignore"] = spwnproject.Gitignore(files[".gitignore"])
		return files, nil
	}
	from, ok := catalog.Example(example)
	if !ok {
		return nil, fmt.Errorf("no example named %q", example)
	}
	files := map[string][]byte{}
	if err := held(from, ".", files); err != nil {
		return nil, err
	}
	if declared, ok := files[project.ManifestName]; ok {
		files[project.ManifestName] = manifestName.ReplaceAll(declared, []byte("name: "+name))
	}
	files[".gitignore"] = spwnproject.Gitignore(files[".gitignore"])
	return files, nil
}

// value: files filled with every file under dir of from, by slash path.
func held(from fs.FS, dir string, files map[string][]byte) error {
	entries, err := fs.ReadDir(from, dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		at := path.Join(dir, e.Name())
		if e.IsDir() {
			err = held(from, at, files)
		} else {
			files[at], err = fs.ReadFile(from, at)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// invariant: a new project's folder is made by one mkdir that fails on
// any name there, a link included, and each of its files is written from
// the folder that holds it, every name on the way opened without
// following a link, as a name nothing held: a link swapped in at the
// folder's name or under it after the mkdir is refused, never written
// through.
func (p Projects) lay(folder, name string, files map[string][]byte) error {
	paths := slices.Sorted(maps.Keys(files))
	for _, at := range paths {
		dir := path.Join(name, path.Dir(at))
		if err := p.under.Folders(folder, dir, 0o755, local.Self()); err != nil {
			return err
		}
		f, err := p.under.Open(folder, dir)
		if err != nil {
			return err
		}
		err = f.Replace(path.Base(at), local.Seen{}, files[at], 0o644, local.Self())
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p Projects) Create(_ context.Context, folder, name, harness, example string) (project.Project, error) {
	root := filepath.Join(folder, name)
	return in(root, func() (project.Project, error) {
		files, err := starting(name, harness, example)
		if err != nil {
			return project.Project{}, err
		}
		switch err = p.under.Mkdir(folder, name, 0o755, local.Self()); {
		case errors.Is(err, fs.ErrExist):
			return project.Project{}, project.ErrExists
		case err != nil:
			return project.Project{}, err
		}
		err = p.lay(folder, name, files)
		if err != nil {
			return project.Project{}, err
		}
		found, err := p.load(root)
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
		found, err := p.load(root)
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
		issues = append(issues, p.inTheWay(root, shown(found).Agents)...)
		return project.PlanOf(found.Root, found.ManifestPath, issues), nil
	})
}

func (p Projects) Drift(_ context.Context, root string) (project.Drift, error) {
	return in(root, func() (project.Drift, error) {
		found, err := p.load(root)
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
		if err := p.declared(root); err != nil {
			return nil, err
		}
		return renders(root, agent)
	})
}

// value: the files agent renders, read under the act lock its caller
// holds.
func renders(root, agent string) ([]project.File, error) {
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
}

// value: an issue for each rendered file whose name holds neither a file
// nor a link, a folder or a FIFO say: apply writes a file there, and never
// takes such a thing away. A name it cannot reach is the changes' read to
// refuse, and an agent that does not render is rendered's to say.
func (p Projects) inTheWay(root string, agents []string) []project.Issue {
	var out []project.Issue
	for _, agent := range agents {
		files, err := renders(root, agent)
		if err != nil {
			continue
		}
		for _, f := range files {
			rel := project.RunFolder(agent) + "/" + f.Path
			dir, name := path.Split(rel)
			folder, err := p.open(root, strings.TrimSuffix(dir, "/"))
			if err != nil {
				continue
			}
			e, err := folder.Lstat(name)
			folder.Close()
			if err == nil && !e.Mode.IsRegular() && e.Mode&fs.ModeSymlink == 0 {
				out = append(out, project.Issue{Level: project.Error, Path: rel, Message: "is not a file, and apply writes a file there",
					Hint: "move it away, then station project plan", Source: project.SourceCompile})
			}
		}
	}
	return out
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
// file there is replaced at once, or taken away only while it holds what
// was just read; a link there is replaced or taken away, never followed;
// anything else there, a folder or a FIFO, is refused, never touched.
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
				out = append(out, project.Issue{Level: project.Error, Path: "agents/" + a.Name + "/AGENTS.md", Message: "agent prompt is missing or empty",
					Hint: "write agents/" + a.Name + "/AGENTS.md with the agent's system prompt", Source: project.SourceCompile})
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
					Hint: "agents/" + a.Name + "/ lacks its AGENTS.md or its agent.yaml", Source: project.SourceCompile})
			}
		}
	}
	return out
}
