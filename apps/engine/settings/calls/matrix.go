package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type AllMachines interface {
	// value: what the matrix reads beyond this machine.
	Machines(ctx context.Context) (machines.AllMachines, error)
	// invariant: each machine's settings.list through the Router, a few at
	// a time, each within its budget (remote/each): a machine that does not
	// answer in time is its own answer's error, never the others' delay.
	Read(ctx context.Context, refs []machine.Ref) map[machine.Ref]Answer
	// value: the last read of each workspace, kept by this Mac's engine in
	// its memory alone: an engine that restarted keeps none.
	Keep(ctx context.Context, vm string, read settings.LastRead)
	Last(ctx context.Context, vm string) (settings.LastRead, bool)
}

type Answer struct {
	// value: one machine's rows, or why it gave none, in the owner's words.
	Rows []settings.Row
	Why  string
}

type MatrixIn struct{}

var Matrix = call.Call[Deps, MatrixIn, settings.Matrix]{
	Name: "settings.matrix", Where: call.InProcess, Applies: settings.ReadsAllMachines,
	Run: func(ctx context.Context, d Deps, _ MatrixIn, _ call.Progress) (settings.Matrix, error) {
		// invariant: every place read at once; a stopped workspace shows its
		// last read, a machine that does not answer is its own column, and
		// the others are filled.
		allMachines, err := d.All.Machines(ctx)
		if err != nil {
			return settings.Matrix{}, err
		}
		livesOn, servers := settings.LivesOnAll(allMachines)
		var asked []machine.Ref
		for _, p := range livesOn {
			if p.Away == settings.Present {
				asked = append(asked, p.Ref)
			}
		}
		answers := d.All.Read(ctx, asked)
		if err := ctx.Err(); err != nil {
			return settings.Matrix{}, err
		}
		now := d.Now()
		cols := make([]settings.Column, 0, len(livesOn))
		for _, p := range livesOn {
			col := settings.Column{LivesOn: p}
			switch p.Away {
			case settings.Present:
				a := answers[p.Ref]
				if a.Rows == nil {
					col.Why = a.Why
					break
				}
				col.Rows, col.Read = a.Rows, now
				if !p.Mac {
					d.All.Keep(ctx, p.Name, settings.LastRead{At: now, Rows: a.Rows})
				}
			case settings.Stopped:
				if last, ok := d.All.Last(ctx, p.Name); ok {
					col.Rows, col.Read = last.Rows, last.At
				}
			case settings.Silent:
			}
			cols = append(cols, col)
		}
		return settings.MatrixOf(cols, servers), nil
	},
}
