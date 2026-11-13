package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func macWorlds() []World {
	pro, mini, air := "machine."+proName, "machine."+miniMac, "machine."+airName
	return []World{
		{Name: "machines.macs", Edit: macIdentities, Pages: []string{pro, mini, air}},
		{Name: "machines.macs.down", Edit: macs().then(infraDown), Pages: []string{mini}},
		{Name: "machines.macs.four-needs", Edit: macs().then(infraDown, sharesEnding), Pages: []string{mini}},
		{Name: "machines.macs.exposed", Edit: macs().then(airPlayOnEvery), Pages: []string{pro}},
		{Name: "machines.macs.empty", Edit: edits{thisMacAlone}.then(macIdentities), Pages: []string{pro}},
		{Name: "machines.macs.checking", Edit: macs().then(miniSilentFor(40 * time.Second)), Pages: []string{mini}},
		{Name: "machines.macs.stale", Edit: macs().then(miniSilentFor(3 * time.Minute)), Pages: []string{mini}},
		{Name: "machines.macs.creating", Edit: macs().then(linux4Creating), Pages: []string{mini}},
		{Name: "machines.macs.created", Edit: macs().then(linux4Ready), Pages: []string{mini}},
	}
}

type edits []func(*Scene)

func (e edits) then(more ...func(*Scene)) func(*Scene) {
	all := append(append(edits{}, e...), more...)
	return func(f *Scene) {
		for _, edit := range all {
			edit(f)
		}
	}
}

func macs() edits { return edits{macIdentities} }

// value: SPEC B5.12's shared fixture: each Mac as it reads itself, the
// air asleep since it last answered on battery with its lid closed, and
// each running VM's share of its Mac's CPU.
func macIdentities(f *Scene) {
	hardware := map[string]coremachine.Hardware{
		proName: {Model: "Mac16,5", Chip: "Apple M4 Max", Memory: 48 * gb, CPUs: 16},
		miniMac: {Model: "Mac16,11", Chip: "Apple M4 Pro", Memory: 64 * gb, CPUs: 14},
		airName: {Model: "Mac15,12", Chip: "Apple M3", Memory: 24 * gb, CPUs: 8},
	}
	for i, h := range f.All.Macs {
		f.All.Macs[i].Hardware = hardware[h.Name]
		f.All.Macs[i].Laptop = h.Name != miniMac
		f.All.Macs[i].Asleep = h.Name == airName
	}
	share := map[string]float64{"linux-1": 40, "agent-7f": 1, "infra": 17, "linux-3": 70, "review-bot": 2, "linux-4": 9}
	for _, mac := range []string{proName, miniMac} {
		readings := f.Samples[mac]
		if len(readings) == 0 {
			continue
		}
		last := &readings[len(readings)-1]
		last.VMs = nil
		for _, h := range f.All.Macs {
			if h.Name != mac {
				continue
			}
			for _, vm := range h.VMs {
				if vm.State == model.Running {
					last.VMs = append(last.VMs, activity.VMUsage{VM: vm.Name, Usage: activity.Usage{CPU: share[vm.Name]}})
				}
			}
		}
	}
}

func sharesEnding(f *Scene) {
	for i, m := range f.Accounts.Machines {
		if m.Name == "linux-3" {
			f.Accounts.Machines[i].Accounts = []accounts.Account{
				loan(catalog.ProviderClaude, catalog.ClaudeTokenName, 12*time.Minute),
				loan(catalog.ProviderGitHub, "github", 12*time.Minute),
			}
		}
	}
}

func airPlayOnEvery(f *Scene) {
	f.Ports.Macs[0].Ports = append(f.Ports.Macs[0].Ports, ports.Port{Machine: coremachine.Ref{Name: proName}, Port: 5000, Process: "ControlCenter", Binds: []string{"*"}, Reach: ports.Outside})
}

func miniSilentFor(d time.Duration) func(*Scene) {
	return func(f *Scene) {
		f.All.Macs[1].Reachable = false
		f.All.Macs[1].Answered = storyNow.Add(-d)
	}
}

func linux4() model.VM {
	vm := sceneVM("linux-4", miniMac, model.Ephemeral, model.Creating, 4, 8, "")
	vm.Agent = &model.AgentView{State: "working", Task: "fix the flaky e2e test"}
	return vm
}

func linux4Creating(f *Scene) {
	vm := linux4()
	vm.Birth = model.BirthAt(model.BornBooting, false, storyNow.Add(-20*time.Second))
	f.All.Macs[1].VMs = append(f.All.Macs[1].VMs, vm)
}

func linux4Ready(f *Scene) {
	vm := linux4()
	vm.State, vm.ReadyAt, vm.BootAt = model.Running, storyNow.Add(-2*time.Second), storyNow.Add(-time.Minute)
	f.All.Macs[1].VMs = append(f.All.Macs[1].VMs, vm)
}
