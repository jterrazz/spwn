package process

import (
	"context"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

// invariant: a VM's rights are read from its Machinefile at each of its
// runs, and when the engine adopted it running at its first read (its
// hello, the first port it reports, the serve reconciliation), so they
// hold for a run as its env does; a Machinefile that cannot be read opens
// nothing, its rows saying why. Each read judges the VM's reservations
// again (judgeLocked).
func (d *Engine) loadRights(ctx context.Context, vm string) ports.Rights {
	d.mu.Lock()
	server := false
	if v := d.vms[vm]; v != nil {
		server = v.rec.Lifetime == model.AlwaysOn
	}
	d.mu.Unlock()
	r := ports.RightsOf(nil, server)
	if d.o.Rights != nil {
		said, err := d.o.Rights(ctx, vm)
		switch {
		case err != nil:
			r = ports.Rights{Refused: err.Error()}
		case said != nil:
			r = ports.RightsOf(said, server)
		}
	}
	mode := d.policy().HTTPS
	d.mu.Lock()
	d.rights[vm] = r
	d.judgeLocked(ctx, vm, r, mode)
	d.mu.Unlock()
	return r
}

func (d *Engine) rightsOf(ctx context.Context, vm string) ports.Rights {
	d.mu.Lock()
	r, ok := d.rights[vm]
	d.mu.Unlock()
	if ok {
		return r
	}
	return d.loadRights(ctx, vm)
}
