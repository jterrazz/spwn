package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type AllMachines interface {
	// value: what the matrix reads beyond this machine.
	Machines(ctx context.Context) (machines.AllMachines, error)
	// invariant: each machine's apps.list through the Router, a few at a
	// time, each within its budget (remote/each): a machine that does not
	// answer in time is its own answer's error, never the others' delay.
	Read(ctx context.Context, refs []machine.Ref) map[machine.Ref]Answer
}

type Answer struct {
	// value: one machine's apps, or why it gave none, in the owner's words.
	Reports []apps.Report
	Why     string
}

type MatrixIn struct{}

var Matrix = call.Call[Deps, MatrixIn, apps.Matrix]{
	Name: "apps.matrix", Where: call.InProcess, Applies: apps.ReadsAllMachines,
	Run: func(ctx context.Context, d Deps, _ MatrixIn, _ call.Progress) (apps.Matrix, error) {
		// invariant: every machine that answers is read at once; a stopped
		// workspace and a machine that does not answer are their own
		// columns, and the others are filled.
		allMachines, err := d.All.Machines(ctx)
		if err != nil {
			return apps.Matrix{}, err
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
			return apps.Matrix{}, err
		}
		now := d.Now()
		cols := make([]apps.Column, 0, len(livesOn))
		reads := make([]apps.Reading, 0, len(livesOn))
		for _, p := range livesOn {
			col := apps.Column{LivesOn: p}
			var rd apps.Reading
			if p.Away == settings.Present {
				a := answers[p.Ref]
				if a.Reports == nil {
					col.Why = a.Why
				} else {
					col.Read = now
					rd = apps.Reading{Reports: a.Reports, Read: true}
				}
			}
			cols = append(cols, col)
			reads = append(reads, rd)
		}
		return apps.MatrixOf(cols, reads, servers), nil
	},
}
