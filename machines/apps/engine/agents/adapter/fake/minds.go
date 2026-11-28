package fake

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	projectcontract "github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Minds struct {
	// invariant: the Minds one machine of the world keeps, each call refused
	// first as its project's layout refuses it.
	w       *world.World
	machine string
	layouts projectcontract.Layouts
}

var _ contract.Minds = (*Minds)(nil)

func Of(w *world.World, machine string, layouts projectcontract.Layouts) *Minds {
	return &Minds{w: w, machine: machine, layouts: layouts}
}

func (m *Minds) laid(project string) error {
	return m.layouts.Within(project, func() error { return nil })
}

func (m *Minds) store() *world.Stores { return m.w.Mac(m.machine).Store() }

func (m *Minds) find(project, name string) (*agents.Mind, error) {
	s := m.store()
	for i := range s.Agents[project] {
		if s.Agents[project][i].Name == name {
			return &s.Agents[project][i], nil
		}
	}
	return nil, agents.ErrNotFound
}

func (m *Minds) put(project string, mind agents.Mind) {
	s := m.store()
	if s.Agents == nil {
		s.Agents = map[string][]agents.Mind{}
	}
	s.Agents[project] = append(s.Agents[project], mind)
	slices.SortFunc(s.Agents[project], func(a, b agents.Mind) int { return strings.Compare(a.Name, b.Name) })
}

func pathOf(project, name string) string {
	if project == "" {
		return "~/.spwn/agents/" + name
	}
	return filepath.Join(project, "agents", name)
}

func empty() []agents.Layer {
	return []agents.Layer{{Name: "journal"}, {Name: "playbooks"}}
}

func clone(mind agents.Mind) agents.Mind {
	out := mind
	out.Layers = make([]agents.Layer, len(mind.Layers))
	for i, l := range mind.Layers {
		out.Layers[i] = agents.Layer{Name: l.Name, Files: slices.Clone(l.Files)}
	}
	out.Journal = slices.Clone(mind.Journal)
	return out
}

func (m *Minds) List(_ context.Context, project string) ([]agents.Agent, error) {
	if err := m.laid(project); err != nil {
		return nil, err
	}
	m.w.Lock()
	defer m.w.Unlock()
	all := m.store().Agents[project]
	out := make([]agents.Agent, 0, len(all))
	for _, mind := range all {
		out = append(out, mind.Agent)
	}
	return out, nil
}

func (m *Minds) Mind(_ context.Context, project, name string) (agents.Mind, error) {
	if err := m.laid(project); err != nil {
		return agents.Mind{}, err
	}
	m.w.Lock()
	defer m.w.Unlock()
	mind, err := m.find(project, name)
	if err != nil {
		return agents.Mind{}, err
	}
	return clone(*mind), nil
}

func (m *Minds) Create(_ context.Context, project, name string, e agents.Edit) (agents.Mind, error) {
	if err := m.laid(project); err != nil {
		return agents.Mind{}, err
	}
	m.w.Lock()
	defer m.w.Unlock()
	if _, err := m.find(project, name); err == nil {
		return agents.Mind{}, agents.ErrExists
	}
	mind := agents.Mind{Agent: e.Apply(agents.Blank(name)), Path: pathOf(project, name), Layers: empty()}
	m.put(project, mind)
	return clone(mind), nil
}

func (m *Minds) Edit(_ context.Context, project, name string, e agents.Edit) (agents.Agent, error) {
	if err := m.laid(project); err != nil {
		return agents.Agent{}, err
	}
	m.w.Lock()
	defer m.w.Unlock()
	mind, err := m.find(project, name)
	if err != nil {
		return agents.Agent{}, err
	}
	mind.Agent = e.Apply(mind.Agent)
	return mind.Agent, nil
}

func (m *Minds) Delete(_ context.Context, project, name string) error {
	if err := m.laid(project); err != nil {
		return err
	}
	m.w.Lock()
	defer m.w.Unlock()
	if _, err := m.find(project, name); err != nil {
		return err
	}
	s := m.store()
	s.Agents[project] = slices.DeleteFunc(s.Agents[project], func(mind agents.Mind) bool { return mind.Name == name })
	return nil
}

func (m *Minds) Fork(_ context.Context, project, from, to string) (agents.Forked, error) {
	if err := m.laid(project); err != nil {
		return agents.Forked{}, err
	}
	m.w.Lock()
	defer m.w.Unlock()
	src, err := m.find(project, from)
	if err != nil {
		return agents.Forked{}, err
	}
	if _, err := m.find(project, to); err == nil {
		return agents.Forked{}, agents.ErrExists
	}
	mind := clone(*src)
	mind.Name, mind.Path, mind.Journal = to, pathOf(project, to), nil
	m.put(project, mind)
	return agents.Forked{From: from, To: to, Layers: []string{"playbooks", "journal"}}, nil
}

func (m *Minds) Export(_ context.Context, project, name, folder string) (agents.Exported, error) {
	if err := m.laid(project); err != nil {
		return agents.Exported{}, err
	}
	m.w.Lock()
	defer m.w.Unlock()
	mind, err := m.find(project, name)
	if err != nil {
		return agents.Exported{}, err
	}
	s := m.store()
	if s.AgentArchives == nil {
		s.AgentArchives = map[string]agents.Mind{}
	}
	archive := filepath.Join(folder, name+agents.ArchiveSuffix)
	s.AgentArchives[archive] = clone(*mind)
	return agents.Exported{Agent: name, Archive: archive}, nil
}

func (m *Minds) Import(_ context.Context, project, archive, as string) (agents.Mind, error) {
	if err := m.laid(project); err != nil {
		return agents.Mind{}, err
	}
	m.w.Lock()
	defer m.w.Unlock()
	if as == "" {
		as = agents.ArchiveName(archive)
	}
	if _, err := m.find(project, as); err == nil {
		return agents.Mind{}, agents.ErrExists
	}
	held, ok := m.store().AgentArchives[archive]
	if !ok {
		return agents.Mind{}, agents.ErrBadArchive
	}
	mind := clone(held)
	mind.Name, mind.Path = as, pathOf(project, as)
	m.put(project, mind)
	return clone(mind), nil
}

func (m *Minds) Dream(_ context.Context, project, name string) (agents.Dreamed, error) {
	if err := m.laid(project); err != nil {
		return agents.Dreamed{}, err
	}
	m.w.Lock()
	defer m.w.Unlock()
	mind, err := m.find(project, name)
	if err != nil {
		return agents.Dreamed{}, err
	}
	if len(mind.Journal) == 0 {
		return agents.Dreamed{Agent: name, Skipped: "no journal entries"}, nil
	}
	d := agents.Dreamed{Agent: name, Entries: len(mind.Journal), Written: mind.Path + "/playbooks/auto-reflexion.md"}
	for _, e := range mind.Journal {
		switch {
		case e.Code == 0:
			d.Completed++
		case e.Code < 0:
			d.Destroyed++
		default:
			d.Failed++
		}
	}
	if scored := d.Completed + d.Failed; scored > 0 {
		d.Rate = float64(d.Completed) / float64(scored)
	}
	mind.Layers = written(mind.Layers, "playbooks", "auto-reflexion.md")
	return d, nil
}

func (m *Minds) Sleep(_ context.Context, project, name string) (agents.Slept, error) {
	if err := m.laid(project); err != nil {
		return agents.Slept{}, err
	}
	m.w.Lock()
	defer m.w.Unlock()
	mind, err := m.find(project, name)
	if err != nil {
		return agents.Slept{}, err
	}
	mind.Layers = written(mind.Layers, "journal", "sleep-"+m.w.Now.Format(time.DateOnly)+".md")
	return agents.Slept{Agent: name}, nil
}

func written(all []agents.Layer, layer, file string) []agents.Layer {
	for i := range all {
		if all[i].Name == layer && !slices.Contains(all[i].Files, file) {
			all[i].Files = append(all[i].Files, file)
			slices.Sort(all[i].Files)
		}
	}
	return all
}
