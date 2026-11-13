package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type ListIn struct {
	// value: every Mac's ports, each asked through its engine.
	AllMachines bool
	// value: the one machine whose ports are kept, a Mac or a VM, read
	// through every Mac; empty keeps all.
	Of string
}

var List = call.Call[Deps, ListIn, ports.Listing]{
	Name: "ports.list", Section: section.Ports, Where: call.AtTarget, Applies: ports.Applies,
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) (ports.Listing, error) {
		allMachines, err := d.Ports.List(ctx, !in.AllMachines && in.Of == "")
		if _, inside := d.Ports.Placed(); err != nil || inside {
			return ports.Listing{AllMachines: allMachines}, err
		}
		node, err := d.Tailnet.Status(ctx)
		l := ports.Listing{AllMachines: ports.OwnDevices(allMachines, node.Peers), Tailnet: node}
		if in.Of != "" {
			l.AllMachines = ports.Of(l.AllMachines, in.Of)
			if len(l.AllMachines.Macs) == 0 || !l.AllMachines.Macs[0].ThisMac {
				l.Tailnet = ports.Status{}
			}
		}
		return l, err
	},
}
