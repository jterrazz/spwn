package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type VMIn struct{ VM string }

type StartOut struct {
	// value: the VM already ran; nothing was started.
	Already bool
	// value: what was started: a server's wanted state is Running again.
	Kind machine.Kind
}

var Start = call.Call[Deps, VMIn, StartOut]{
	Name: "machines.start", Where: call.InEngine, Applies: machine.Anywhere, Needs: machines.VMNeeds(), Journal: keptStart,
	Run: func(ctx context.Context, d Deps, in VMIn, _ call.Progress) (StartOut, error) {
		if err := d.Machines.CheckStart(in.VM); err != nil {
			return StartOut{}, err
		}
		if err := d.Machines.EngineUp(ctx); err != nil {
			return StartOut{}, err
		}
		kind := d.Machines.LifetimeOf(in.VM).Kind()
		// invariant: a VM whose create failed is retried even while it runs:
		// its run is not what it lacks.
		if running, err := d.Machines.Running(ctx, in.VM); err != nil || running && !d.Machines.NotCreated(in.VM) {
			return StartOut{Already: running, Kind: kind}, err
		}
		return StartOut{Kind: kind}, d.Machines.Start(ctx, in.VM)
	},
}

type StopIn struct {
	VM      string
	Confirm string
}

type StopOut struct {
	// value: the VM was already stopped; nothing was asked or done.
	Already bool
	// value: what was stopped: a server's wanted state is Stopped from now on.
	Kind machine.Kind
}

var Stop = call.Call[Deps, StopIn, StopOut]{
	// invariant: a stop keeps the VM, ephemeral, persistent or always on
	// (SP-5); a workspace asks yes or no first, a server its name typed, and
	// the rule is checked here, on the VM's own Mac, whoever sent the call.
	// One already stopped asks nothing.
	Name: "machines.stop", Where: call.InEngine, Applies: machine.Anywhere, Confirm: confirm.Stop, Journal: keptStop,
	Errors: []fail.Code{confirm.YesNeeded, confirm.Needed, confirm.Mismatch},
	Run: func(ctx context.Context, d Deps, in StopIn, _ call.Progress) (StopOut, error) {
		if err := d.Machines.CheckStop(in.VM); err != nil {
			return StopOut{}, err
		}
		kind := d.Machines.LifetimeOf(in.VM).Kind()
		if err := d.Machines.EngineUp(ctx); err != nil {
			return StopOut{Kind: kind}, err
		}
		if running, err := d.Machines.Running(ctx, in.VM); err != nil || !running {
			return StopOut{Already: err == nil, Kind: kind}, err
		}
		if err := confirm.Ask(confirm.Stop, kind, "", in.VM, in.Confirm); err != nil {
			return StopOut{Kind: kind}, err
		}
		return StopOut{Kind: kind}, d.Machines.Stop(ctx, in.VM)
	},
}

type DeleteIn struct {
	VM      string
	Confirm string
}

var Delete = call.Call[Deps, DeleteIn, machines.DeleteReport]{
	Name: "machines.delete", Where: call.InEngine, Applies: machine.Anywhere, Confirm: confirm.Delete, Journal: keptDelete,
	Errors: []fail.Code{confirm.Needed, confirm.Mismatch, BeingCreated},
	Run: func(ctx context.Context, d Deps, in DeleteIn, _ call.Progress) (machines.DeleteReport, error) {
		if err := d.Machines.CheckDelete(in.VM); err != nil {
			return machines.DeleteReport{}, Worded(err)
		}
		lifetime := d.Machines.LifetimeOf(in.VM)
		hold, _ := d.Machines.HoldOf(ctx, in.VM)
		// invariant: a throwaway nobody holds is deleted with no question; a
		// permanent one, or one someone works in, asks its name typed, and
		// the question says who holds it.
		if lifetime != machines.Ephemeral || hold.State == machines.HoldHeld {
			err := confirm.Ask(confirm.Delete, lifetime.Kind(), "", in.VM, in.Confirm)
			var asked *fail.Error
			if errors.As(err, &asked) && hold.State == machines.HoldHeld {
				asked.Detail["holder"] = hold.Sentence
			}
			if err != nil {
				return machines.DeleteReport{}, err
			}
		}
		if err := d.Machines.EngineUp(ctx); err != nil {
			return machines.DeleteReport{}, err
		}
		report, err := d.Machines.Delete(ctx, in.VM)
		return report, Worded(err)
	},
}

type RestartIn struct {
	// value: a Mac (a FileVault-aware reboot) or a VM (a poweroff
	// and a start).
	Name    string
	Confirm string
}

var Restart = call.Call[Deps, RestartIn, machines.RestartReport]{
	Name: "machines.restart", Where: call.AtTarget, Applies: machine.Anywhere, Confirm: confirm.RestartMac,
	ConfirmBy: map[machine.Kind]confirm.Action{machine.Server: RestartConfirm(machine.Server), machine.Workspace: RestartConfirm(machine.Workspace)},
	Errors:    []fail.Code{confirm.YesNeeded, confirm.Needed, confirm.Mismatch}, Journal: keptRestart,
	Run: func(ctx context.Context, d Deps, in RestartIn, _ call.Progress) (machines.RestartReport, error) {
		if err := d.Machines.CheckMac(); err != nil {
			return machines.RestartReport{}, err
		}
		kind, err := d.Machines.CheckRestart(in.Name)
		if err != nil {
			return machines.RestartReport{}, err
		}
		if err := confirm.Ask(RestartConfirm(kind), kind, "", in.Name, in.Confirm); err != nil {
			return machines.RestartReport{}, err
		}
		return d.Machines.Restart(ctx, in.Name)
	},
}

func RestartConfirm(kind machine.Kind) confirm.Action {
	// invariant: a Mac's restart asks a yes, a server's its name typed (S4),
	// a workspace's nothing: the confirmation Restart's ConfirmBy declares.
	switch kind {
	case machine.Server:
		return confirm.RestartServer
	case machine.Workspace:
		return confirm.Restart
	case machine.Computer:
	}
	return confirm.RestartMac
}

type LogOut struct {
	// value: SPEC B5.5: every meaningful line the VM's runs wrote, its
	// console's then its hypervisor's own.
	Lines []string
}

var Log = call.Call[Deps, VMIn, LogOut]{
	Name: "machines.log", Where: call.AtTarget, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in VMIn, _ call.Progress) (LogOut, error) {
		lines, err := d.Machines.Log(ctx, in.VM)
		return LogOut{Lines: lines}, err
	},
}

type EditIn struct {
	VM string
	// value: the sizes to change; a zero one is kept.
	Resources machines.Resources
	// value: a server's health check to set, or NoCheck to remove it.
	Check   string
	NoCheck bool
}

var Edit = call.Call[Deps, EditIn, machines.Edited]{
	Name: "machines.edit", Where: call.InEngine, Applies: machine.Anywhere,
	Errors: []fail.Code{BeingCreated},
	Run: func(ctx context.Context, d Deps, in EditIn, _ call.Progress) (machines.Edited, error) {
		change := machines.Change{Resources: in.Resources, Check: in.Check, NoCheck: in.NoCheck}
		err := d.Machines.CheckEdit(in.VM, change)
		if errors.As(err, new(*machines.BeingCreatedError)) {
			return machines.Edited{}, Worded(err)
		}
		if err != nil {
			return machines.Edited{}, err
		}
		return d.Machines.Edit(ctx, in.VM, change)
	},
}
