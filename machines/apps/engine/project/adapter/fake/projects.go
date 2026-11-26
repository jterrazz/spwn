package fake

import (
	"context"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Projects struct {
	// invariant: the projects one machine of the world keeps, by root.
	w       *world.World
	machine string
}

var _ contract.Projects = (*Projects)(nil)

func Starter() project.File {
	// value: what a new project's starter agent renders in the fake world:
	// the prompt a harness reads.
	return project.File{Path: "CLAUDE.md", Data: []byte("# " + project.Starter + "\n"), Mode: 0o644}
}

func New() *Projects { return Of(&world.World{}, "this-mac") }

func Of(w *world.World, machine string) *Projects {
	return &Projects{w: w, machine: machine}
}

func (p *Projects) kept() map[string]project.State { return p.w.Mac(p.machine).Store().Projects }

func (p *Projects) state(root string) (project.State, error) {
	s, ok := p.kept()[root]
	if !ok {
		return project.State{}, project.ErrNotFound
	}
	return s, nil
}

func (p *Projects) Find(_ context.Context, folder string) (string, error) {
	p.w.Lock()
	defer p.w.Unlock()
	for dir := filepath.Clean(folder); ; dir = filepath.Dir(dir) {
		if _, ok := p.kept()[dir]; ok {
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
	s, err := p.state(root)
	return s.Project, err
}

func agentsOf(example string) []string {
	// value: the agents an example holds, by name; the starter alone when
	// none is named.
	files, ok := catalog.Example(example)
	if !ok {
		return []string{project.Starter}
	}
	entries, _ := fs.ReadDir(files, "spwn/agents")
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
	made := project.Project{Name: name, Root: root, Manifest: filepath.Join(root, project.ManifestName), Agents: agentsOf(example)}
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
	s, err := p.state(root)
	if err != nil {
		return project.Plan{}, err
	}
	return project.PlanOf(root, s.Project.Manifest, s.Issues), nil
}

func (p *Projects) Drift(_ context.Context, root string) (project.Drift, error) {
	p.w.Lock()
	defer p.w.Unlock()
	s, err := p.state(root)
	if err != nil {
		return project.Drift{}, err
	}
	return project.Drift{Root: root, Manifest: s.Project.Manifest, Rows: append([]project.Issue{}, s.Drift...)}, nil
}

func (p *Projects) Render(_ context.Context, root, agent string) ([]project.File, error) {
	p.w.Lock()
	defer p.w.Unlock()
	s, err := p.state(root)
	if err != nil {
		return nil, err
	}
	return slices.Clone(s.Rendered[agent]), nil
}

const recordPath = project.StateFolder + "/" + project.RecordName

// value: the link the disk holds on the way to rel, on a folder of its
// path, as the real disk refuses it; nil when none is.
func linkOn(s project.State, rel string) error {
	for at := path.Dir(rel); at != "."; at = path.Dir(at) {
		if f, ok := s.Disk[at]; ok && f.Link {
			return &project.Unwritten{Path: at, Err: project.ErrLink}
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
	for _, c := range changes {
		if c.Kind == project.Remove {
			delete(s.Disk, c.Path)
			continue
		}
		s.Disk[c.Path] = c.File
	}
	s.Record = maps.Clone(record)
	p.kept()[root] = s
	return nil
}
