package menubar

import (
	"strconv"

	"github.com/jterrazz/studio-station/src/accounts"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

// invariant: SPEC A3, B5.7: the status item counts what needs you and never
// the upkeep: a server down, a port exposed, a copy shared for less than
// 30 min more. A server down or a port exposed tints it red; a server
// restarting turns the busy mark in place of its red and is not counted.
// Its label says the one signal it shows. nil until the machines are read.
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
			case model.Down(vm):
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
	b := page.Badge{Needs: len(reds) + sharesEnding(r.Accounts), Down: len(reds) > 0}
	switch {
	case b.Down:
		b.Label = "station, " + reds[0]
	case restarting != nil:
		b.Restarting = true
		b.Label = restarting.Name + " restarting, " + restarting.Restart.Counted() + " checks"
	case b.Needs == 1:
		b.Label = "station, 1 thing needs you"
	case b.Needs > 1:
		b.Label = "station, " + strconv.Itoa(b.Needs) + " things need you"
	default:
		b.Label = "all machines are fine"
	}
	return &b
}

func sharesEnding(a *accounts.AllMachines) int {
	if a == nil {
		return 0
	}
	n := 0
	for _, c := range a.Read().Shared {
		if c.Short {
			n++
		}
	}
	return n
}
