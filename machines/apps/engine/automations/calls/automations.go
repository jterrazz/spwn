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

// value: the newest runs a log shows when the surface is not told how many.
const (
	DefaultLast = 50
)

type ListIn struct {
	Project string `json:"project,omitempty"`
}

var List = call.Call[Deps, ListIn, []automations.Automation]{
	Errors: []fail.Code{InvalidProject},
	Name:   "automations.list", Where: call.AtTarget, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) ([]automations.Automation, error) {
		all, err := d.Automations.List(ctx, d.root(in.Project))
		return all, coded(err, in.Project, "", "automations")
	},
}

type ShowIn struct {
	Project    string `json:"project,omitempty"`
	Automation string
}

var Show = call.Call[Deps, ShowIn, automations.Automation]{
	Name: "automations.show", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, NotUnique, InvalidProject},
	Run: func(ctx context.Context, d Deps, in ShowIn, _ call.Progress) (automations.Automation, error) {
		a, err := d.Automations.Automation(ctx, d.root(in.Project), in.Automation)
		return a, coded(err, in.Project, in.Automation, "automations")
	},
}

type LogIn struct {
	Project string `json:"project,omitempty"`
	// value: one automation's runs; empty for every run of the project.
	Automation string `json:"automation,omitempty"`
	// value: how many of the newest runs; 0 is every run the file holds.
	Last int `json:"last,omitempty"`
}

var Log = call.Call[Deps, LogIn, []automations.Run]{
	Name: "automations.log", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, NotUnique, InvalidProject},
	Run: func(ctx context.Context, d Deps, in LogIn, _ call.Progress) ([]automations.Run, error) {
		runs, err := d.Automations.Log(ctx, d.root(in.Project), in.Automation, in.Last)
		return runs, coded(err, in.Project, in.Automation, "automations log")
	},
}
