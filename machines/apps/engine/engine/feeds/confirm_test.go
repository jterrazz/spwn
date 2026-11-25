package feeds

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
)

// value: the level a page's question draws: none, the name typed, or yes
// or no.
func drawn(c *page.Confirm) confirm.Level {
	switch {
	case c == nil:
		return confirm.None
	case c.Type != "":
		return confirm.TypeName
	}
	return confirm.YesNo
}

func TestTheMenuBarAsksEachGestureOnAVMAtTheLevelItsCallAsks(t *testing.T) {
	// invariant: ADR-097: one grading for every surface. The menu bar's
	// question on a VM is the one its call asks of the same VM, and the
	// page never answers a typed question for the owner.
	gestures := map[string]func(machines.VM) confirm.Action{
		pagemenubar.ActionDelete: func(machines.VM) confirm.Action { return confirm.Delete },
		pagemenubar.ActionStop:   func(machines.VM) confirm.Action { return confirm.Stop },
		pagemenubar.ActionRestart: func(vm machines.VM) confirm.Action {
			if vm.Lifetime == machines.AlwaysOn {
				return confirm.RestartServer
			}
			return confirm.Restart
		},
	}
	for _, vm := range []machines.VM{
		{Name: "linux-1", Kind: catalog.Linux, Lifetime: machines.Ephemeral, State: machines.Running},
		{Name: "linux-2", Kind: catalog.Linux, Lifetime: machines.Ephemeral, State: machines.Stopped},
		{Name: "agent-7f", Kind: catalog.Linux, Lifetime: machines.Ephemeral, State: machines.Running, Creator: "studio-os-43"},
		{Name: "api", Kind: catalog.Linux, Lifetime: machines.Persistent, State: machines.Running},
		{Name: "infra", Kind: catalog.Linux, Lifetime: machines.AlwaysOn, State: machines.Running},
	} {
		all := machines.Judged(machines.AllMachines{Macs: []machines.MacView{{Name: thisMac, ThisMac: true, Reachable: true, VMs: []machines.VM{vm}}}, Read: at})
		p := pagemenubar.MachinePage(pagemenubar.Machine{Name: vm.Name, All: &all, Asks: levels, Now: at})
		judged := all.Macs[0].VMs[0]
		for _, a := range p.Actions {
			action, graded := gestures[a.ID]
			if !graded {
				continue
			}
			want := confirm.Of(action(judged), judged.Lifetime.Kind(), machines.FactsOf(judged)...)
			if got := drawn(a.Confirm); got != want {
				t.Errorf("%s of %s draws %s, its call asks %s", a.ID, vm.Name, got, want)
			}
			if _, fixed := a.Run.In["Confirm"]; want == confirm.TypeName && fixed {
				t.Errorf("%s of %s answers the name typed for the owner: %s", a.ID, vm.Name, a.Run.In["Confirm"])
			}
		}
	}
}
