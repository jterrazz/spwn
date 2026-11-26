package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

// value: every call's Project is the root of the project whose manifest
// declares the automations, an absolute path on the target; empty for none,
// which declares none.

// value: the newest runs a log answers with when it is not told how many.
const (
	DefaultLast = 50
)

type ListIn struct {
	Project string `json:"project,omitempty"`
}

var List = call.Call[Deps, ListIn, []automations.Automation]{
	Name: "automations.list", Where: call.AtTarget, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) ([]automations.Automation, error) {
		return d.Automations.List(ctx, d.root(in.Project))
	},
}

type ShowIn struct {
	Project    string `json:"project,omitempty"`
	Automation string
}

var Show = call.Call[Deps, ShowIn, automations.Automation]{
	Name: "automations.show", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, NotUnique},
	Run: func(ctx context.Context, d Deps, in ShowIn, _ call.Progress) (automations.Automation, error) {
		a, err := d.Automations.Automation(ctx, d.root(in.Project), in.Automation)
		return a, coded(err, in.Automation, "automations")
	},
}

type LogIn struct {
	Project string `json:"project,omitempty"`
	// value: one automation's runs; empty for every run of the project.
	Automation string `json:"automation,omitempty"`
	// value: how many of the newest runs; DefaultLast when not given.
	Last int `json:"last,omitempty"`
}

var Log = call.Call[Deps, LogIn, []automations.Run]{
	Name: "automations.log", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, NotUnique},
	Run: func(ctx context.Context, d Deps, in LogIn, _ call.Progress) ([]automations.Run, error) {
		last := in.Last
		if last <= 0 {
			last = DefaultLast
		}
		runs, err := d.Automations.Log(ctx, d.root(in.Project), in.Automation, last)
		return runs, coded(err, in.Automation, "automations log")
	},
}
