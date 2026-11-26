package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/teams"
)

type Teams interface {
	// invariant: the teams are the user's own, declared once for every
	// project; a team's members are the agents of the project asked, its
	// root an absolute path on this machine, empty for the user's own
	// agents. Nothing here writes a team.

	// invariant: every team, by slug, its members by name.
	List(ctx context.Context, project string) ([]teams.Team, error)
	// invariant: the team of the slug; a slug no team has is teams.ErrNotFound.
	Team(ctx context.Context, project, slug string) (teams.Team, error)
}
