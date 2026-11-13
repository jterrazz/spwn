package calls

import (
	"context"
	"errors"
	"slices"
	"strings"

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
	Name: "machines.start", Where: call.InEngine, Applies: machine.Anywhere, Needs: machines.VMNeeds(),
	Run: func(ctx context.Context, d Deps, in VMIn, _ call.Progress) (StartOut, error) {
		if err := d.Machines.CheckStart(in.VM); err != nil {
			return StartOut{}, err
		}
		if err := d.Machines.EngineUp(ctx); err != nil {
			return StartOut{}, err
		}
		kind := d.Machines.LifetimeOf(in.VM).Kind()
		if running, err := d.Machines.Running(ctx, in.VM); err != nil || running {
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
	// value: SPEC B5.22: its agents, stopped with it.
	Agents []string
	// value: each agent that did not stop, said; the VM stopped all the same.
	Warnings []string
}

var Stop = call.Call[Deps, StopIn, StopOut]{
	// invariant: a stop keeps the VM, ephemeral, persistent or always on
	// (SP-5); a workspace asks yes or no first, a server its name typed, and
	// the rule is checked here, on the VM's own Mac, whoever sent the call.
	// One already stopped asks nothing.
	Name: "machines.stop", Where: call.InEngine, Applies: machine.Anywhere, Confirm: confirm.Stop,
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
		if err := naming(confirm.Ask(confirm.Stop, kind, "", in.VM, in.Confirm), live(ctx, d, in.VM)); err != nil {
			return StopOut{Kind: kind}, err
		}
		stopped, warned := stopAgents(ctx, d, in.VM, "its workspace stopped")
		held := heldBy(ctx, d, in.VM)
		if err := d.Machines.Stop(ctx, in.VM); err != nil {
			return StopOut{Kind: kind, Agents: stopped, Warnings: warned}, err
		}
		var kept error
		if held != nil {
			kept = d.Machines.KeepAgents(in.VM, held)
		}
		return StopOut{Kind: kind, Agents: stopped, Warnings: warned}, kept
	},
}

type DeleteIn struct {
	VM      string
	Confirm string
}

var Delete = call.Call[Deps, DeleteIn, machines.DeleteReport]{
	Name: "machines.delete", Where: call.InEngine, Applies: machine.Anywhere, Confirm: confirm.Delete,
	Errors: []fail.Code{confirm.Needed, confirm.Mismatch, BeingCreated},
	Run: func(ctx context.Context, d Deps, in DeleteIn, _ call.Progress) (machines.DeleteReport, error) {
		if err := d.Machines.CheckDelete(in.VM); err != nil {
			return machines.DeleteReport{}, Worded(err)
		}
		running, _ := d.Machines.Running(ctx, in.VM)
		var named []string
		if running {
			named = live(ctx, d, in.VM)
		}
		if err := naming(confirm.Ask(confirm.Delete, d.Machines.LifetimeOf(in.VM).Kind(), "", in.VM, in.Confirm), named); err != nil {
			return machines.DeleteReport{}, err
		}
		if err := d.Machines.EngineUp(ctx); err != nil {
			return machines.DeleteReport{}, err
		}
		var stopped, warned []string
		if running {
			stopped, warned = stopAgents(ctx, d, in.VM, "its workspace was deleted")
		}
		report, err := d.Machines.Delete(ctx, in.VM)
		report.Agents, report.Warnings = stopped, slices.Concat(warned, report.Warnings)
		return report, Worded(err)
	},
}

// value: the live agents of a workspace of this Mac; none for a server.
func live(ctx context.Context, d Deps, vm string) []string {
	if d.Agents == nil || d.Machines.LifetimeOf(vm).Kind() != machine.Workspace {
		return nil
	}
	return d.Agents.Live(ctx, vm)
}

// value: the name of every agent of a workspace of this Mac, finished ones
// too; nil when it does not answer, or for a server.
func heldBy(ctx context.Context, d Deps, vm string) []string {
	if d.Agents == nil || d.Machines.LifetimeOf(vm).Kind() != machine.Workspace {
		return nil
	}
	return d.Agents.Held(ctx, vm)
}

func stopAgents(ctx context.Context, d Deps, vm, reason string) (stopped, warnings []string) {
	if d.Agents == nil || d.Machines.LifetimeOf(vm).Kind() != machine.Workspace {
		return nil, nil
	}
	return d.Agents.Stop(ctx, vm, reason)
}

// invariant: SPEC A5: the confirmation of a VM that hosts live agents
// names them, which stop first.
func naming(err error, agents []string) error {
	var e *fail.Error
	if len(agents) > 0 && errors.As(err, &e) {
		e.Detail["agents"] = strings.Join(agents, ", ")
	}
	return err
}

type RestartIn struct {
	// value: a Mac (a FileVault-aware reboot) or a VM (a poweroff
	// and a start).
	Name    string
	Confirm string
}

var Restart = call.Call[Deps, RestartIn, machines.RestartReport]{
	Name: "machines.restart", Where: call.AtTarget, Applies: machine.Anywhere, Confirm: confirm.RestartMac,
	Errors: []fail.Code{confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in RestartIn, _ call.Progress) (machines.RestartReport, error) {
		if err := d.Machines.CheckMac(); err != nil {
			return machines.RestartReport{}, err
		}
		vm, err := d.Machines.CheckRestart(in.Name)
		if err != nil {
			return machines.RestartReport{}, err
		}
		gesture := confirm.RestartMac
		if vm {
			gesture = confirm.Restart
		}
		if err := confirm.Require(confirm.For(gesture, d.Kind), in.Name, in.Confirm); err != nil {
			return machines.RestartReport{}, err
		}
		return d.Machines.Restart(ctx, in.Name)
	},
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
	Run: func(ctx context.Context, d Deps, in EditIn, _ call.Progress) (machines.Edited, error) {
		change := machines.Change{Resources: in.Resources, Check: in.Check, NoCheck: in.NoCheck}
		if err := d.Machines.CheckEdit(in.VM, change); err != nil {
			return machines.Edited{}, err
		}
		return d.Machines.Edit(ctx, in.VM, change)
	},
}
