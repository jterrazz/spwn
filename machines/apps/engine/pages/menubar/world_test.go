package menubar

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	heroPro  = "macbook-pro"
	heroMini = "mac-mini"
)

var heroNow = time.Date(2026, 11, 7, 14, 30, 0, 0, time.UTC)

type heroWorld struct {
	r Reading
}

func newHeroWorld() *heroWorld {
	infra := model.VM{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running, BootAt: heroNow.Add(-12 * 24 * time.Hour)}
	linux1 := model.VM{Name: "linux-1", Lifetime: model.Ephemeral, State: model.Running}
	return &heroWorld{r: Reading{Now: heroNow, Zone: time.UTC, Ports: &ports.AllMachines{},
		All: &model.AllMachines{Read: heroNow, Macs: []model.MacView{
			{Name: heroPro, ThisMac: true, Reachable: true, Answered: heroNow, Hardware: hardware(48), VMs: []model.VM{linux1}},
			{Name: heroMini, Reachable: true, Answered: heroNow, Hardware: hardware(64), VMs: []model.VM{infra}},
		}}}}
}

func (w *heroWorld) mini() *model.MacView { return &w.r.All.Macs[1] }

// value: the world with one of mac-mini's ports reachable from outside.
func (w *heroWorld) exposedOnMini(port int) *heroWorld {
	w.r.Ports.Macs = append(w.r.Ports.Macs, ports.MacPorts{Name: heroMini, Ports: []ports.Port{
		{Machine: machine.Ref{Name: heroMini}, Port: port, Process: "node", Reach: ports.Outside, Answer: ports.Answers},
	}})
	return w
}

func (w *heroWorld) granted(t *testing.T, vm string) *heroWorld {
	t.Helper()
	w.r.SignIns = &SignIns{Machines: []MachineSignIns{{Name: vm, Mac: heroPro, ThisMac: true, VM: true, Rows: []accounts.ProgramRow{grantedRow(t, catalog.ProviderClaude, "pw-1")}}}}
	return w
}

func (w *heroWorld) down() *heroWorld {
	infra := &w.mini().VMs[0]
	infra.State, infra.Cause, infra.BootAt = model.Stopped, "out of memory", time.Time{}
	infra.Restarts = model.Restarts{Looping: true}
	infra.CheckStatus = model.CheckStatus{State: model.CheckFailing, Since: heroNow.Add(-2 * time.Minute)}
	return w
}

func (w *heroWorld) exposed() *heroWorld {
	w.r.Ports.Macs = append(w.r.Ports.Macs, ports.MacPorts{Name: heroPro, ThisMac: true, Ports: []ports.Port{
		{Machine: machine.Ref{Name: heroPro}, Port: 5000, Process: "ControlCenter", Reach: ports.Outside, Answer: ports.Answers},
	}})
	return w
}

func hardware(gigabytes int64) machine.Hardware {
	return machine.Hardware{Memory: gigabytes << 30, CPUs: 12}
}
