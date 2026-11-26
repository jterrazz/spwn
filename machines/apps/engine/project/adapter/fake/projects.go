package fake

import (
	"context"
	"path/filepath"

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

func (p *Projects) Create(_ context.Context, folder, name, _ string) (project.Project, error) {
	p.w.Lock()
	defer p.w.Unlock()
	root := filepath.Join(folder, name)
	if _, err := p.state(root); err == nil {
		return project.Project{}, project.ErrExists
	}
	made := project.Project{Name: name, Root: root, Manifest: filepath.Join(root, project.ManifestName), Agents: []string{project.Starter}}
	s := p.w.Mac(p.machine).Store()
	if s.Projects == nil {
		s.Projects = map[string]project.State{}
	}
	s.Projects[root] = project.State{Project: made}
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
