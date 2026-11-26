package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/teams"
)

// value: every call's Project is the root of the project whose agents are
// the teams' members, an absolute path on the target; empty for the user's
// own agents.

type ListIn struct {
	Project string `json:"project,omitempty"`
}

var List = call.Call[Deps, ListIn, []teams.Team]{
	Name: "teams.list", Where: call.AtTarget, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) ([]teams.Team, error) {
		return d.Teams.List(ctx, d.root(in.Project))
	},
}

type ShowIn struct {
	Project string `json:"project,omitempty"`
	Team    string
}

var Show = call.Call[Deps, ShowIn, teams.Team]{
	Name: "teams.show", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound},
	Run: func(ctx context.Context, d Deps, in ShowIn, _ call.Progress) (teams.Team, error) {
		t, err := d.Teams.Team(ctx, d.root(in.Project), in.Team)
		return t, coded(err, in.Team)
	},
}
