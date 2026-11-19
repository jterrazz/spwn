package menubar

import (
	"strconv"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

// invariant: SPEC A3, B5.7: the status item says one signal and counts
// nothing: a server down or a port exposed slashes the mark; a server
// restarting lights its dots in a wave in place of the slash. Its label
// says the one signal it shows. nil until the machines are read.
func badge(r Reading) *page.Badge {
	if r.All == nil {
		return nil
	}
	var reds []string
	var restarting *model.VM
	for _, h := range r.All.Macs {
		for i, vm := range h.VMs {
			switch {
			case vm.Lifetime == model.AlwaysOn && vm.Restart != nil:
				if restarting == nil {
					restarting = &h.VMs[i]
				}
			case vm.Standing == model.StandingDown:
				reds = append(reds, vm.Name+" down")
			}
		}
	}
	if r.Ports != nil {
		for _, h := range r.Ports.Macs {
			for _, p := range h.Ports {
				if p.Red() {
					reds = append(reds, "port "+strconv.Itoa(p.Port)+" exposed")
				}
			}
		}
	}
	b := page.Badge{Down: len(reds) > 0}
	switch {
	case b.Down:
		b.Label = "Spwn Menu, " + reds[0]
	case restarting != nil:
		b.Restarting = true
		b.Label = restarting.Name + " restarting, " + restarting.Restart.Counted() + " checks"
	default:
		b.Label = "all machines are fine"
	}
	return &b
}
