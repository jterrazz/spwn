package calls

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
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
	// value: who the owner shares it with; empty keeps its audience, else
	// its rights'. Refused inside a VM: a guest is never the owner.
	To catalog.Audience
}

const AudienceAbility = "ports.share.to"

func (in ShareIn) Asks() []string {
	// invariant: an older binary decodes To as absent and shares with the
	// rights' audience, so an audience is asked only of one that lists it.
	if in.To == "" {
		return nil
	}
	return []string{AudienceAbility}
}

var Share = call.Call[Deps, ShareIn, ports.SharedPort]{
	// invariant: a share reserves the port on the VM's Mac now, before
	// anything listens, and prints its URL; it takes back a stop of sharing.
	Name: "ports.share", Section: section.Ports, Where: call.InEngine, Applies: machine.Anywhere, Needs: ports.ShareNeeds(),
	Abilities: []string{AudienceAbility},
	Run: func(ctx context.Context, d Deps, in ShareIn, _ call.Progress) (ports.SharedPort, error) {
		port, err := d.Ports.CheckPort(in.Port)
		if err != nil {
			return ports.SharedPort{}, err
		}
		if in.To != "" && !slices.Contains(in.To.Enum(), string(in.To)) {
			return ports.SharedPort{}, ports.UnknownAudience(string(in.To))
		}
		vm, inside := d.Ports.Placed()
		if inside && in.To != "" {
			return ports.SharedPort{}, ports.AudienceInside(vm, port, string(in.To))
		}
		if !inside {
			if vm = in.VM; vm == "" {
				return ports.SharedPort{}, fail.New(fail.Refused, ports.CodeNeedsVM, in.Port, nil)
			}
			if mac, err := d.Ports.CheckVM(ctx, vm); err != nil || mac != "" {
				return ports.SharedPort{}, elsewhere(vm, mac, err)
			}
		}
		return d.Ports.Share(ctx, vm, port, in.To)
	},
}

func elsewhere(vm, mac string, err error) error {
	if err != nil {
		return err
	}
	return fail.New(fail.Refused, ports.CodeElsewhere, vm, map[string]string{"mac": mac})
}
