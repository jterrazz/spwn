package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/engine"
)

type ReadIn struct{}

var Read = call.Call[Deps, ReadIn, engine.Reading]{
	Name: "engine.read", Where: call.AtTarget, Applies: Applies,
	Run: func(ctx context.Context, d Deps, _ ReadIn, _ call.Progress) (engine.Reading, error) {
		return d.Control.Read(ctx)
	},
}

type StartIn struct{}

var Start = call.Call[Deps, StartIn, engine.StartReport]{
	Name: "engine.start", Where: call.AtTarget, Applies: Applies,
	Errors: []fail.Code{StartFailed},
	Run: func(ctx context.Context, d Deps, _ StartIn, _ call.Progress) (engine.StartReport, error) {
		// invariant: start is also how the engine comes to a Mac: its job is
		// written when it is not installed, and an engine of another version
		// restarted; an engine installed and current is only started.
		var installed engine.Installed
		if d.Installer != nil {
			reading, err := d.Control.Read(ctx)
			if err != nil {
				return engine.StartReport{}, err
			}
			if reading.State == engine.EngineNotInstalled || reading.Skewed() {
				if installed, err = d.Installer.Install(ctx); err != nil {
					return engine.StartReport{}, startError(err)
				}
			}
		}
		if err := d.Control.Check(); err != nil {
			return engine.StartReport{}, err
		}
		report, err := d.Control.Start(ctx)
		report.Installed = installed
		return report, startError(err)
	},
}

func startError(err error) error {
	if failed := new(engine.StartFailedError); errors.As(err, &failed) {
		return &fail.Error{Type: fail.Failed, Code: StartFailed, Subject: "engine", Detail: map[string]string{"last": failed.Last}, Cause: err}
	}
	return err
}

type StopIn struct {
	Confirm string
	// value: what asked the stop, as a reading names it: the command, or
	// the menu bar.
	By string
}

type StopOut struct {
	// value: what the stop ends; handed back with the confirmation it asks,
	// so the question names it.
	Work engine.Work
	// value: set once the stop ran, whole or part-way.
	Report *engine.StopReport
}

var Stop = call.Call[Deps, StopIn, StopOut]{
	Name: "engine.stop", Where: call.AtTarget, Applies: Applies, Confirm: confirm.StopEngine,
	Errors: []fail.Code{confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in StopIn, _ call.Progress) (StopOut, error) {
		// invariant: an engine running no VM stops without a question; one
		// running VMs asks first, since they stop with it.
		if err := d.Control.Check(); err != nil {
			return StopOut{}, err
		}
		work, err := d.Control.Plan(ctx)
		if err != nil {
			return StopOut{}, err
		}
		if !work.Empty() {
			if refused := confirm.Require(confirm.For(confirm.StopEngine, d.Kind), "stop", in.Confirm); refused != nil {
				return StopOut{Work: work}, refused
			}
		}
		by := in.By
		if by == "" {
			by = "station engine stop"
		}
		report, err := d.Control.Stop(ctx, work, by)
		return StopOut{Work: work, Report: &report}, err
	},
}
