package fake

import (
	"context"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/skills"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Files struct {
	// invariant: the Files one machine of the world keeps.
	w       *world.World
	machine string
}

var _ contract.Files = (*Files)(nil)

func New() *Files { return Of(&world.World{}, "this-mac") }

func Of(w *world.World, machine string) *Files {
	return &Files{w: w, machine: machine}
}

func (f *Files) store() *world.Stores { return f.w.Mac(f.machine).Store() }

func pathOf(project, name string) string {
	if project == "" {
		return "~/.spwn/skills/" + name + skills.Suffix
	}
	return filepath.Join(project, "spwn", "skills", name+skills.Suffix)
}

func (f *Files) find(project, name string) (int, error) {
	for i, s := range f.store().Skills[project] {
		if s.Name == name {
			return i, nil
		}
	}
	return -1, skills.ErrNotFound
}

func (f *Files) List(_ context.Context, project string) ([]skills.Skill, error) {
	f.w.Lock()
	defer f.w.Unlock()
	all := f.store().Skills[project]
	out := make([]skills.Skill, 0, len(all))
	for _, s := range all {
		s.Body = ""
		out = append(out, s)
	}
	return out, nil
}

func (f *Files) Show(_ context.Context, project, name string) (skills.Skill, error) {
	f.w.Lock()
	defer f.w.Unlock()
	i, err := f.find(project, name)
	if err != nil {
		return skills.Skill{}, err
	}
	return f.store().Skills[project][i], nil
}

func (f *Files) Create(_ context.Context, project, name string) (skills.Skill, error) {
	f.w.Lock()
	defer f.w.Unlock()
	if _, err := f.find(project, name); err == nil {
		return skills.Skill{}, skills.ErrExists
	}
	s := f.store()
	if s.Skills == nil {
		s.Skills = map[string][]skills.Skill{}
	}
	made := skills.Blank(name, pathOf(project, name))
	s.Skills[project] = append(s.Skills[project], made)
	slices.SortFunc(s.Skills[project], func(a, b skills.Skill) int { return strings.Compare(a.Name, b.Name) })
	return made, nil
}

func (f *Files) Delete(_ context.Context, project, name string) error {
	f.w.Lock()
	defer f.w.Unlock()
	i, err := f.find(project, name)
	if err != nil {
		return err
	}
	s := f.store()
	s.Skills[project] = slices.Delete(s.Skills[project], i, i+1)
	return nil
}
