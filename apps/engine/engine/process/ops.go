package process

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/ports"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/engine"

	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

type Peer interface {
	// value: what a process asks of this Mac's engine over its socket.
	Hello(ctx context.Context) (engine.Hello, error)
	VMs(ctx context.Context) ([]engine.VMView, error)
	StopAll(ctx context.Context) (engine.StopAll, error)
	AwakeView(ctx context.Context) (awake.View, error)
}

func fault(code, message, hint string) *enginecalls.OpError {
	// value: a code the caller got wrong (a VM, a verb, a page it named), not
	// one the engine failed at.
	return &enginecalls.OpError{Code: code, Message: message, Hint: hint}
}

func (d *Engine) Hello() engine.Hello {
	return engine.Hello{Version: d.o.Version, Page: d.o.Page, PID: d.o.PID, StartedAt: d.started, Executable: d.o.Executable}
}

func (d *Engine) VMs() []engine.VMView { return d.vmViews() }

func (d *Engine) Ports(ctx context.Context, vm string) ports.MacPorts { return d.surveyed(ctx, vm) }

func (d *Engine) Reserve(ctx context.Context, vm string, port int) (ports.SharedPort, error) {
	return d.shareOf(ctx, vm, port, ports.ByOwner)
}

func (d *Engine) Release(ctx context.Context, vm string, port int) (ports.Reservation, error) {
	return d.stopSharing(ctx, vm, port)
}

// invariant: a share takes back a stop of sharing, and a stop of sharing is
// remembered, on this Mac whoever asked: a command here or a VM's guest;
// by is who makes it, the owner for a command, the rights for a guest.
func (d *Engine) shareOf(ctx context.Context, vm string, port int, by string) (ports.SharedPort, error) {
	if d.o.Remember != nil {
		if err := d.o.Remember(vm, port, false); err != nil {
			return ports.SharedPort{}, err
		}
	}
	return d.reserve(ctx, vm, port, by)
}

func (d *Engine) stopSharing(ctx context.Context, vm string, port int) (ports.Reservation, error) {
	l, err := d.release(ctx, vm, port)
	if err != nil || d.o.Remember == nil {
		return l, err
	}
	return l, d.o.Remember(vm, port, true)
}

func (d *Engine) VM(ctx context.Context, verb, vm string) error {
	switch verb {
	case engine.VMRun:
		if err := d.opRun(ctx, vm); err != nil {
			return err
		}
		return d.awaitGuest(ctx, vm)
	case engine.VMStart:
		if err := d.opStart(ctx, vm); err != nil {
			return err
		}
		return d.awaitGuest(ctx, vm)
	case engine.VMStop:
		return d.opStop(ctx, vm, true)
	case engine.VMRestart:
		if err := d.opStop(ctx, vm, false); err != nil {
			return err
		}
		if err := d.opRun(ctx, vm); err != nil {
			return err
		}
		return d.awaitGuest(ctx, vm)
	case engine.VMDelete:
		return d.opDelete(ctx, vm)
	case engine.VMForget:
		d.forget(vm)
		return nil
	case engine.VMReady:
		return d.ready(ctx, vm)
	}
	return fault("unknown_op", "unknown verb \""+verb+"\" for a VM", "station machines")
}

func (d *Engine) Accounts(ctx context.Context) engine.AccountsView { return d.accounts(ctx) }

func (d *Engine) StopAll(ctx context.Context) engine.StopAll {
	shared := d.sharedVMs()
	vms := d.stopAll(ctx)
	all := engine.StopAll{VMs: vms, AwakeEnded: d.awakeJob.Stop()}
	for _, v := range vms {
		all.SharingEnded = all.SharingEnded || v.Result == engine.ResultStopped && shared[v.Name]
	}
	return all
}

func (d *Engine) AwakeView() awake.View { return d.awakeJob.View() }

func (d *Engine) Awake(r engine.AwakeIn) (awake.Awoken, error) { return d.keepAwake(r) }

func (d *Engine) Sleep() engine.AwakeOff { return d.endAwake() }

func (d *Engine) Refresh(pages []string) error {
	if err := d.CheckPages(pages); err != nil {
		return err
	}
	d.hub.refreshed(pages)
	return nil
}
