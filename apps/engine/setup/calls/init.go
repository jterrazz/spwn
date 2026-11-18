package calls

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

type InitIn struct {
	// value: the name this Mac takes; empty asks it at a terminal, or keeps
	// the one it has.
	Name        string
	Interactive bool
	// value: yes once the owner agreed to the moves init announced; empty
	// while a move is due refuses before any step.
	Confirm string
	// value: only what init would move, in Moves: nothing is done.
	Plan bool
}

var Init = call.Call[Deps, InitIn, setup.Report]{
	Name: "setup.init", Where: call.AtTarget, Applies: machine.Anywhere, Confirm: confirm.Relocate,
	Errors: []fail.Code{confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in InitIn, _ call.Progress) (setup.Report, error) {
		if err := d.Setup.CheckName(in.Name); err != nil {
			return setup.Report{}, err
		}
		moves := d.Setup.Moves(ctx)
		if in.Plan {
			return setup.Report{Moves: moves}, nil
		}
		if len(moves) > 0 {
			from := make([]string, 0, len(moves))
			for _, m := range moves {
				from = append(from, m.From)
			}
			if err := confirm.Require(confirm.For(confirm.Relocate, machine.Computer), strings.Join(from, ", "), in.Confirm); err != nil {
				return setup.Report{Moves: moves}, err
			}
		}
		var ask func() (string, error)
		if d.Ask != nil {
			ask = func() (string, error) { return d.Ask.Name(ctx) }
		}
		return d.Setup.Init(ctx, in.Name, in.Interactive, ask)
	},
}
