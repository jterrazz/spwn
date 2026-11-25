package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
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
		// written whenever the job manager does not hold it (a Mac whose bare
		// jobs run, at the bundle's version or another, moves to the bundle),
		// and an engine of another version, or one that serves no ssh agent,
		// restarted; an engine installed, current and serving its agent is
		// only started.
		var installed engine.Installed
		if d.Installer != nil {
			reading, err := d.Control.Read(ctx)
			if err != nil {
				return engine.StartReport{}, err
			}
			if reading.State == machines.EngineNotInstalled || reading.Skewed() || reading.Agent.Unserved() || d.Control.Check() != nil {
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

type RestartIn struct{}

var Restart = call.Call[Deps, RestartIn, engine.RestartReport]{
	Name: "engine.restart", Where: call.AtTarget, Applies: Applies,
	Errors: []fail.Code{StartFailed},
	Run: func(ctx context.Context, d Deps, _ RestartIn, _ call.Progress) (engine.RestartReport, error) {
		// invariant: the engine's job restarts and the VMs it runs keep
		// running: they are processes of their own, adopted again by the
		// engine that comes back.
		if err := d.Control.Check(); err != nil {
			return engine.RestartReport{}, err
		}
		installed, err := d.Installer.Restart(ctx)
		if err != nil {
			return engine.RestartReport{}, startError(err)
		}
		started, err := d.Control.Start(ctx)
		if err != nil {
			return engine.RestartReport{}, startError(err)
		}
		started.Installed = installed
		started.Already = false
		reading, err := d.Control.Read(ctx)
		if err != nil {
			return engine.RestartReport{}, err
		}
		return engine.RestartReport{StartReport: started, VMs: reading.VMs}, nil
	},
}

type StopIn struct {
	Confirm string
	// value: what asked the stop, as a reading names it: the command, or
	// the menu bar.
	By string
	// value: after the stop, the engine's job is taken from the job manager
	// too, so it holds none.
	Remove bool
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
			if refused := confirm.Require(confirm.Of(confirm.StopEngine, d.Kind), "stop", in.Confirm); refused != nil {
				return StopOut{Work: work}, refused
			}
		}
		by := in.By
		if by == "" {
			by = "station engine stop"
		}
		report, err := d.Control.Stop(ctx, work, by)
		if err == nil && in.Remove && d.Installer != nil && (report.Engine == engine.EngineHalted || report.Engine == engine.EngineAlready) {
			// invariant: the jobs are removed only once the engine has stopped, never
			// while it keeps running for a server.
			if err = d.Installer.Uninstall(ctx); err == nil {
				report.Removed = true
			}
		}
		return StopOut{Work: work, Report: &report}, err
	},
}
