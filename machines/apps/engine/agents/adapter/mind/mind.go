package mind

import (
	"context"
	"errors"
	"slices"
	"strings"

	"spwn.sh/packages/agent"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	projectcontract "github.com/jterrazz/studio-station/apps/engine/project/contract"
)

// value: the prefix agent.yaml writes before a harness's name.
const harnessScheme = "spwn:"

// value: how many journal entries a Mind's detail reads.
const journalRead = 20

type Minds struct {
	// value: the folder removal of this machine's disk, which takes back
	// what an archive that did not read left.
	RemoveAll func(path string) error
	// value: the layout of the project a Mind is read and written in.
	Layouts projectcontract.Layouts
}

var _ contract.Minds = Minds{}

// value: what act returns, run within the layout of the project at root
// as l holds it.
func within[T any](l projectcontract.Layouts, root string, act func() (T, error)) (T, error) {
	var out T
	err := l.Within(root, func() error {
		var err error
		out, err = act()
		return err
	})
	return out, err
}

func exists(name string) bool {
	_, err := agent.InspectAgent(name)
	return err == nil
}

func found(name string) error {
	if !exists(name) {
		return agents.ErrNotFound
	}
	return nil
}

func harnessOf(backend string) string {
	return strings.TrimPrefix(backend, harnessScheme)
}

func read(name string) (agents.Agent, error) {
	m, err := agent.LoadManifest(name)
	if err != nil {
		return agents.Agent{}, err
	}
	return agents.Agent{Name: name, Description: m.Description, Team: m.Team, Harness: harnessOf(m.Runtime.Backend)}, nil
}

func edit(name string, e agents.Edit) (agents.Agent, error) {
	m, err := agent.LoadManifest(name)
	if err != nil {
		return agents.Agent{}, err
	}
	if e.Team != nil {
		m.Team = *e.Team
	}
	if e.Harness != nil {
		m.Runtime.Backend = ""
		if *e.Harness != "" {
			m.Runtime.Backend = harnessScheme + *e.Harness
		}
	}
	if err := agent.SaveManifest(name, m); err != nil {
		return agents.Agent{}, err
	}
	return read(name)
}

func mindOf(name string) (agents.Mind, error) {
	a, err := read(name)
	if err != nil {
		return agents.Mind{}, err
	}
	info, err := agent.InspectAgent(name)
	if err != nil {
		return agents.Mind{}, err
	}
	out := agents.Mind{Agent: a, Path: info.Path}
	for layer, files := range info.Layers {
		out.Layers = append(out.Layers, agents.Layer{Name: layer, Files: slices.Sorted(slices.Values(files))})
	}
	slices.SortFunc(out.Layers, func(a, b agents.Layer) int { return strings.Compare(a.Name, b.Name) })
	entries, err := agent.ListJournal(info.Path, journalRead)
	if err != nil {
		return agents.Mind{}, err
	}
	for _, e := range entries {
		out.Journal = append(out.Journal, agents.Entry{At: e.CreatedAt, World: e.WorldID, Outcome: e.Outcome, Code: e.ExitCode, Took: e.Duration})
	}
	return out, nil
}

func (m Minds) List(_ context.Context, project string) ([]agents.Agent, error) {
	return within(m.Layouts, project, func() ([]agents.Agent, error) {
		infos, err := agent.ListAgents()
		if err != nil {
			return nil, err
		}
		var out []agents.Agent
		for _, info := range infos {
			a, err := read(info.Name)
			if err != nil {
				return nil, err
			}
			out = append(out, a)
		}
		slices.SortFunc(out, func(a, b agents.Agent) int { return strings.Compare(a.Name, b.Name) })
		return out, nil
	})
}

func (m Minds) Mind(_ context.Context, project, name string) (agents.Mind, error) {
	return within(m.Layouts, project, func() (agents.Mind, error) {
		if err := found(name); err != nil {
			return agents.Mind{}, err
		}
		return mindOf(name)
	})
}

func (m Minds) Create(_ context.Context, project, name string, e agents.Edit) (agents.Mind, error) {
	return within(m.Layouts, project, func() (agents.Mind, error) {
		if exists(name) {
			return agents.Mind{}, agents.ErrExists
		}
		if _, err := agent.InitMind(name); err != nil {
			return agents.Mind{}, err
		}
		if e != (agents.Edit{}) {
			if _, err := edit(name, e); err != nil {
				return agents.Mind{}, err
			}
		}
		return mindOf(name)
	})
}

func (m Minds) Edit(_ context.Context, project, name string, e agents.Edit) (agents.Agent, error) {
	return within(m.Layouts, project, func() (agents.Agent, error) {
		if err := found(name); err != nil {
			return agents.Agent{}, err
		}
		return edit(name, e)
	})
}

func (m Minds) Delete(_ context.Context, project, name string) error {
	_, err := within(m.Layouts, project, func() (struct{}, error) {
		if err := found(name); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, agent.DeleteAgent(name)
	})
	return err
}

func (m Minds) Fork(_ context.Context, project, from, to string) (agents.Forked, error) {
	return within(m.Layouts, project, func() (agents.Forked, error) {
		if err := found(from); err != nil {
			return agents.Forked{}, err
		}
		if exists(to) {
			return agents.Forked{}, agents.ErrExists
		}
		r, err := agent.Fork(from, to, nil)
		if err != nil {
			return agents.Forked{}, err
		}
		return agents.Forked{From: r.Source, To: r.Target, Layers: r.LayersCopied}, nil
	})
}

func (m Minds) Export(_ context.Context, project, name, folder string) (agents.Exported, error) {
	return within(m.Layouts, project, func() (agents.Exported, error) {
		if err := found(name); err != nil {
			return agents.Exported{}, err
		}
		archive, err := agent.ExportMind(name, folder, nil)
		if err != nil {
			return agents.Exported{}, err
		}
		return agents.Exported{Agent: name, Archive: archive}, nil
	})
}

func (m Minds) Import(_ context.Context, project, archive, as string) (agents.Mind, error) {
	if as == "" {
		as = agents.ArchiveName(archive)
	}
	return within(m.Layouts, project, func() (agents.Mind, error) {
		if exists(as) {
			return agents.Mind{}, agents.ErrExists
		}
		if err := agent.ImportMind(as, archive); err != nil {
			// invariant: an archive that does not read leaves no agent behind.
			if rm := m.RemoveAll(agent.AgentDir(as)); rm != nil {
				return agents.Mind{}, errors.Join(agents.ErrBadArchive, err, rm)
			}
			return agents.Mind{}, errors.Join(agents.ErrBadArchive, err)
		}
		return mindOf(as)
	})
}

func (m Minds) Dream(_ context.Context, project, name string) (agents.Dreamed, error) {
	return within(m.Layouts, project, func() (agents.Dreamed, error) {
		if err := found(name); err != nil {
			return agents.Dreamed{}, err
		}
		r, err := agent.Dream(name)
		if err != nil {
			return agents.Dreamed{}, err
		}
		out := agents.Dreamed{Agent: name, Entries: r.EntriesAnalyzed, Completed: r.CompletedTasks, Failed: r.FailedTasks, Destroyed: r.DestroyedTasks, Rate: r.SuccessRate, Written: r.OutputPath}
		if r.Skipped {
			out.Skipped = r.Reason
		}
		return out, nil
	})
}

func (m Minds) Sleep(_ context.Context, project, name string) (agents.Slept, error) {
	return within(m.Layouts, project, func() (agents.Slept, error) {
		if err := found(name); err != nil {
			return agents.Slept{}, err
		}
		r, err := agent.Sleep(name)
		if err != nil {
			return agents.Slept{}, err
		}
		return agents.Slept{Agent: name, Archived: r.ArchivedPlaybooks, Pruned: r.PrunedSessions}, nil
	})
}
