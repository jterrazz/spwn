package fake

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Projects struct {
	// invariant: the projects one machine of the world keeps, by root.
	w       *world.World
	machine string
	folders Folders
}

var _ contract.Projects = (*Projects)(nil)

func Starter() project.File {
	// value: what a new project's starter agent renders in the fake world:
	// the prompt a harness reads.
	return project.File{Path: "CLAUDE.md", Data: []byte("# " + project.Starter + "\n"), Mode: 0o644}
}

// value: the folders of a project held as files, by kind, out of its
// spwn.yaml (nil when it has none) and every path it holds, refused as the
// real disk refuses them: the real adapter's own reading, handed in.

type Folders func(spwnYAML []byte, paths []string) (map[string]string, error)

func New(folders Folders) *Projects { return Of(&world.World{}, "this-mac", folders) }

func Of(w *world.World, machine string, folders Folders) *Projects {
	return &Projects{w: w, machine: machine, folders: folders}
}

func (p *Projects) kept() map[string]project.State { return p.w.Mac(p.machine).Store().Projects }

func (p *Projects) state(root string) (project.State, error) {
	s, ok := p.kept()[root]
	if !ok {
		return project.State{}, project.ErrNotFound
	}
	return s, nil
}

// value: the project at root as a read of its declaration finds it,
// refused as the real disk refuses it when a name of the declaration is a
// link or neither a file nor a folder.
func (p *Projects) read(root string) (project.State, error) {
	s, err := p.state(root)
	if err != nil {
		return s, err
	}
	return s, p.declared(s)
}

func (p *Projects) declared(s project.State) error {
	byKind, err := p.layoutOf(s)
	if err != nil {
		return err
	}
	folders := slices.Sorted(maps.Values(byKind))
	if err := within(s, byKind); err != nil {
		return err
	}
	for _, at := range slices.Sorted(maps.Keys(s.Disk)) {
		if project.Declared(folders, at) && s.Disk[at].Link {
			return &project.Unwritten{Path: at, Kind: kindOf(byKind, at), Err: project.ErrLink}
		}
		if s.Disk[at].Link {
			for k, f := range byKind {
				if strings.HasPrefix(f, at+"/") {
					return &project.Unwritten{Path: at, Kind: k, Err: project.ErrLink}
				}
			}
		}
	}
	for _, at := range s.Irregular {
		if project.Declared(folders, at) {
			return &project.Unwritten{Path: at, Kind: kindOf(byKind, at), Err: project.ErrNotAFile}
		}
	}
	return nil
}

// value: nil when the names under the layout's folders, each folder on a
// held path's way counted once with its files, fit one project.Budget, as
// the real walk counts them; else the first name past it, in path order.
func within(s project.State, byKind map[string]string) error {
	names := map[string]int64{}
	for _, at := range slices.Concat(slices.Collect(maps.Keys(s.Disk)), s.Irregular) {
		if kindOf(byKind, at) == "" {
			continue
		}
		names[at] = int64(len(s.Disk[at].Data))
		for dir := path.Dir(at); dir != "."; dir = path.Dir(dir) {
			if kindOf(byKind, dir) != "" {
				names[dir] = 0
			}
		}
	}
	var b project.Budget
	for _, at := range slices.Sorted(maps.Keys(names)) {
		if b.Past(at, names[at]) {
			return &project.Unwritten{Path: at, Kind: kindOf(byKind, at), Err: project.ErrTooLarge}
		}
	}
	return nil
}

// value: the kind whose folder holds at, or is at; "" when none does.
func kindOf(byKind map[string]string, at string) string {
	for k, f := range byKind {
		if at == f || strings.HasPrefix(at, f+"/") {
			return k
		}
	}
	return ""
}

// value: the folders the project's spwn.yaml lays out, by kind, out of
// what the world keeps of it, as p.folders reads them.
func (p *Projects) layoutOf(s project.State) (map[string]string, error) {
	var manifest []byte
	if f, ok := s.Disk[project.ManifestName]; ok {
		manifest = f.Data
	}
	return p.folders(manifest, slices.Sorted(maps.Keys(s.Disk)))
}

func (p *Projects) Find(_ context.Context, folder string) (string, error) {
	p.w.Lock()
	defer p.w.Unlock()
	for dir := filepath.Clean(folder); ; dir = filepath.Dir(dir) {
		if s, ok := p.kept()[dir]; ok {
			if err := p.declared(s); err != nil {
				return "", err
			}
			return dir, nil
		}
		if filepath.Dir(dir) == dir {
			return "", project.ErrNotFound
		}
	}
}

func (p *Projects) Show(_ context.Context, root string) (project.Project, error) {
	p.w.Lock()
	defer p.w.Unlock()
	s, err := p.read(root)
	return s.Project, err
}

func (p *Projects) agentsOf(example string) []string {
	// value: the agents an example holds, by name, in the agents folder its
	// spwn.yaml lays out; the starter alone when none is named.
	files, ok := catalog.Example(example)
	if !ok {
		return []string{project.Starter}
	}
	manifest, _ := fs.ReadFile(files, project.ManifestName)
	folders, err := p.folders(manifest, nil)
	if err != nil {
		return nil
	}
	entries, _ := fs.ReadDir(files, folders["agents"])
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func (p *Projects) Create(_ context.Context, folder, name, _, example string) (project.Project, error) {
	p.w.Lock()
	defer p.w.Unlock()
	root := filepath.Join(folder, name)
	if _, err := p.state(root); err == nil {
		return project.Project{}, project.ErrExists
	}
	made := project.Project{Name: name, Root: root, Manifest: filepath.Join(root, project.ManifestName), Agents: p.agentsOf(example)}
	rendered := map[string][]project.File{}
	for _, a := range made.Agents {
		rendered[a] = []project.File{Starter()}
	}
	s := p.w.Mac(p.machine).Store()
	if s.Projects == nil {
		s.Projects = map[string]project.State{}
	}
	s.Projects[root] = project.State{Project: made, Rendered: rendered}
	return made, nil
}

func (p *Projects) Plan(_ context.Context, root string) (project.Plan, error) {
	p.w.Lock()
	defer p.w.Unlock()
	s, err := p.read(root)
	if err != nil {
		return project.Plan{}, err
	}
	issues := slices.Clone(s.Issues)
	for _, agent := range slices.Sorted(maps.Keys(s.Rendered)) {
		for _, f := range s.Rendered[agent] {
			rel := project.RunFolder(agent) + "/" + f.Path
			if filled(s, rel) || slices.Contains(s.Irregular, rel) {
				issues = append(issues, project.Issue{Level: project.Error, Path: rel, Message: "is not a file, and apply writes a file there",
					Hint: "move it away, then station project plan", Source: project.SourceCompile})
			}
		}
	}
	return project.PlanOf(root, s.Project.Manifest, issues), nil
}

func (p *Projects) Drift(_ context.Context, root string) (project.Drift, error) {
	p.w.Lock()
	defer p.w.Unlock()
	s, err := p.read(root)
	if err != nil {
		return project.Drift{}, err
	}
	return project.Drift{Root: root, Manifest: s.Project.Manifest, Rows: append([]project.Issue{}, s.Drift...)}, nil
}

func (p *Projects) Render(_ context.Context, root, agent string) ([]project.File, error) {
	p.w.Lock()
	defer p.w.Unlock()
	s, err := p.read(root)
	if err != nil {
		return nil, err
	}
	return slices.Clone(s.Rendered[agent]), nil
}

const recordPath = project.StateFolder + "/" + project.RecordName

// value: the link or the file the disk holds on the way to rel, on a
// folder of its path, as the real disk refuses it; nil when none is.
func linkOn(s project.State, rel string) error {
	for at := path.Dir(rel); at != "."; at = path.Dir(at) {
		switch f, ok := s.Disk[at]; {
		case ok && f.Link:
			return &project.Unwritten{Path: at, Err: project.ErrLink}
		case ok:
			return &project.Unwritten{Path: at, Err: project.ErrNotAFolder}
		}
	}
	return nil
}

// value: the link on the way to the record or at it; nil when none is.
func recordLinked(s project.State) error {
	if f, ok := s.Disk[recordPath]; ok && f.Link {
		return &project.Unwritten{Path: recordPath, Err: project.ErrLink}
	}
	return linkOn(s, recordPath)
}

func (p *Projects) Applied(_ context.Context, root string, paths []string) (record, disk project.Record, err error) {
	p.w.Lock()
	defer p.w.Unlock()
	s, err := p.state(root)
	if err != nil {
		return nil, nil, err
	}
	if err := recordLinked(s); err != nil {
		return nil, nil, err
	}
	disk = project.Record{}
	for _, rel := range paths {
		if err := linkOn(s, rel); err != nil {
			return nil, nil, err
		}
		if f, ok := s.Disk[rel]; ok {
			disk[rel] = project.Digest(f)
		} else if filled(s, rel) {
			disk[rel] = project.Digest(project.File{Path: rel, Data: []byte(fs.ModeDir.String())})
		}
	}
	return maps.Clone(s.Record), disk, nil
}

func (p *Projects) Write(_ context.Context, root string, changes []project.Change, record project.Record) error {
	p.w.Lock()
	defer p.w.Unlock()
	s, err := p.state(root)
	if err != nil {
		return err
	}
	if err := recordLinked(s); err != nil {
		return err
	}
	for _, c := range changes {
		if err := linkOn(s, c.Path); err != nil {
			return err
		}
	}
	if s.Disk == nil {
		s.Disk = map[string]project.File{}
	}
	defer func() { p.kept()[root] = s }()
	unrecorded := &project.Unwritten{Path: recordPath, Err: errReadOnly}
	for i, c := range changes {
		var refused error
		switch {
		case slices.Contains(s.ReadOnly, path.Dir(c.Path)):
			refused = errReadOnly
		case filled(s, c.Path):
			refused = errFolderInTheWay
		}
		if refused != nil {
			failed := &project.Unwritten{Path: c.Path, Err: refused}
			if slices.Contains(s.ReadOnly, project.StateFolder) {
				return errors.Join(failed, unrecorded)
			}
			s.Record = project.Kept(s.Record, record, changes, i)
			return failed
		}
		if c.Kind == project.Remove {
			delete(s.Disk, c.Path)
			continue
		}
		s.Disk[c.Path] = c.File
	}
	if slices.Contains(s.ReadOnly, project.StateFolder) {
		return unrecorded
	}
	s.Record = maps.Clone(record)
	return nil
}

var (
	errFolderInTheWay = errors.New("is a folder that holds files")
	errReadOnly       = errors.New("is in a folder without its write bit")
)

// value: rel names a folder that holds a file, as a file the world wrote
// under it says.
func filled(s project.State, rel string) bool {
	for at := range s.Disk {
		if strings.HasPrefix(at, rel+"/") {
			return true
		}
	}
	return false
}
