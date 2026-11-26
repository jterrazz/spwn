package fake

import (
	"context"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/teams"
	"github.com/jterrazz/studio-station/apps/engine/teams/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Teams struct {
	// invariant: the Teams one machine of the world keeps; its members are
	// read from the Minds of the same machine, never stored.
	w       *world.World
	machine string
}

var _ contract.Teams = (*Teams)(nil)

func Of(w *world.World, machine string) *Teams {
	return &Teams{w: w, machine: machine}
}

func (t *Teams) store() *world.Stores { return t.w.Mac(t.machine).Store() }

func (t *Teams) membersOf(project, slug string) []string {
	var out []string
	for _, mind := range t.store().Agents[project] {
		if mind.Team == slug {
			out = append(out, mind.Name)
		}
	}
	slices.Sort(out)
	return out
}

func (t *Teams) read(declared teams.Team, project string) teams.Team {
	declared.Members = t.membersOf(project, declared.Slug)
	return declared
}

func (t *Teams) List(_ context.Context, project string) ([]teams.Team, error) {
	t.w.Lock()
	defer t.w.Unlock()
	out := make([]teams.Team, 0, len(t.store().Teams))
	for _, declared := range t.store().Teams {
		out = append(out, t.read(declared, project))
	}
	slices.SortFunc(out, func(a, b teams.Team) int { return strings.Compare(a.Slug, b.Slug) })
	return out, nil
}

func (t *Teams) Team(_ context.Context, project, slug string) (teams.Team, error) {
	t.w.Lock()
	defer t.w.Unlock()
	for _, declared := range t.store().Teams {
		if declared.Slug == slug {
			return t.read(declared, project), nil
		}
	}
	return teams.Team{}, teams.ErrNotFound
}
