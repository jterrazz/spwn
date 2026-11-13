package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type ShareIn struct {
	// value: the VM whose port it is; empty inside a VM, which shares its own.
	VM   string
	Port string
}

var Share = call.Call[Deps, ShareIn, ports.SharedPort]{
	// invariant: a share reserves the port on the VM's Mac now, before
	// anything listens, and prints its URL; it takes back a stop of sharing.
	Name: "ports.share", Section: section.Ports, Where: call.InEngine, Applies: machine.Anywhere, Needs: ports.ShareNeeds(),
	Run: func(ctx context.Context, d Deps, in ShareIn, _ call.Progress) (ports.SharedPort, error) {
		port, err := d.Ports.CheckPort(in.Port)
		if err != nil {
			return ports.SharedPort{}, err
		}
		vm, inside := d.Ports.Placed()
		if !inside {
			if vm = in.VM; vm == "" {
				return ports.SharedPort{}, fail.New(fail.Refused, ports.CodeNeedsVM, in.Port, nil)
			}
			if mac, err := d.Ports.CheckVM(ctx, vm); err != nil || mac != "" {
				return ports.SharedPort{}, elsewhere(vm, mac, err)
			}
		}
		return d.Ports.Share(ctx, vm, port)
	},
}

func elsewhere(vm, mac string, err error) error {
	if err != nil {
		return err
	}
	return fail.New(fail.Refused, ports.CodeElsewhere, vm, map[string]string{"mac": mac})
}
