package files

import (
	"context"
	"slices"
	"strings"

	"spwn.sh/packages/platform"
	"spwn.sh/packages/project"

	"github.com/jterrazz/studio-station/apps/engine/teams"
	"github.com/jterrazz/studio-station/apps/engine/teams/contract"
)

type Teams struct{}

var _ contract.Teams = Teams{}

func read(t project.Team) (teams.Team, error) {
	members, err := project.TeamMembers(t.Slug)
	if err != nil {
		return teams.Team{}, err
	}
	slices.Sort(members)
	return teams.Team{Slug: t.Slug, Name: t.Name, Description: t.Description, Color: t.Color, Members: members}, nil
}

func (Teams) List(_ context.Context, root string) ([]teams.Team, error) {
	return platform.WithProjectRoot(root, func() ([]teams.Team, error) {
		declared, err := project.ListTeams()
		if err != nil {
			return nil, err
		}
		out := make([]teams.Team, 0, len(declared))
		for _, d := range declared {
			t, err := read(d)
			if err != nil {
				return nil, err
			}
			out = append(out, t)
		}
		slices.SortFunc(out, func(a, b teams.Team) int { return strings.Compare(a.Slug, b.Slug) })
		return out, nil
	})
}

func (Teams) Team(_ context.Context, root, slug string) (teams.Team, error) {
	return platform.WithProjectRoot(root, func() (teams.Team, error) {
		// invariant: a team is one of the files ListTeams reads, so a slug
		// that names no such file, or a path, is not found.
		declared, err := project.ListTeams()
		if err != nil {
			return teams.Team{}, err
		}
		for _, d := range declared {
			if d.Slug == slug {
				return read(d)
			}
		}
		return teams.Team{}, teams.ErrNotFound
	})
}
