package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

type InitIn struct {
	// value: the name this Mac takes; empty asks it at a terminal, or keeps
	// the one it has.
	Name        string
	Interactive bool
}

var Init = call.Call[Deps, InitIn, setup.Report]{
	Name: "setup.init", Where: call.AtTarget, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in InitIn, _ call.Progress) (setup.Report, error) {
		if err := d.Setup.CheckName(in.Name); err != nil {
			return setup.Report{}, err
		}
		var ask func() (string, error)
		if d.Ask != nil {
			ask = func() (string, error) { return d.Ask.Name(ctx) }
		}
		return d.Setup.Init(ctx, in.Name, in.Interactive, ask)
	},
}
