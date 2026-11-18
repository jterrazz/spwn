package pages

import (
	"time"

	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

// value: the home stories' worlds: the default machines (macbook-air asleep,
// each Mac saying what it has), then each state of the hero over it.
func homeWorlds() []World {
	home := func(name string, edits ...func(*Scene)) World {
		return World{Name: name, Edit: func(f *Scene) {
			homeScene(f)
			for _, e := range edits {
				e(f)
			}
		}, Pages: []string{"menubar"}}
	}
	return []World{
		home("machines.home"),
		home("machines.home.down", infraDown),
		home("machines.home.restarting-2", infraRestarting),
		home("machines.home.recovered", infraRecovered),
		home("machines.home.exposed", airPlayExposed),
		home("machines.home.down-exposed", infraDown, airPlayExposed),
		home("machines.home.fine", nothingNeedsYou),
		home("machines.home.cleanup", ephemeralsStopped),
		home("machines.home.empty", thisMacAlone),
		home("machines.home.loading", miniSilent),
	}
}

func homeScene(f *Scene) {
	for i, h := range f.All.Macs {
		switch h.Name {
		case proName:
			f.All.Macs[i].Hardware, f.All.Macs[i].Laptop = coremachine.Hardware{Model: "Mac16,6", Chip: "Apple M4 Max", Memory: 48 * gb, CPUs: 16}, true
		case miniMac:
			f.All.Macs[i].Hardware = coremachine.Hardware{Model: "Mac16,11", Chip: "Apple M4 Pro", Memory: 64 * gb, CPUs: 14}
		case airName:
			f.All.Macs[i].Asleep, f.All.Macs[i].Laptop = true, true
		}
	}
}

func infraRecovered(f *Scene) {
	infra := &f.All.Macs[1].VMs[0]
	infra.BootAt = storyNow.Add(-time.Minute)
	infra.Restarts = model.Restarts{Count: 1, Last: storyNow.Add(-2 * time.Minute)}
}

func airPlayExposed(f *Scene) {
	pro := &f.Ports.Macs[0]
	pro.Ports = append(pro.Ports, ports.Port{Machine: coremachine.Ref{Name: proName}, Port: 5000, Process: "ControlCenter", Binds: []string{"0.0.0.0"}, Reach: ports.Outside, Answer: ports.Answers})
}

// value: Clean up's world: agent-7f stopped on This Mac and linux-3 on
// mac-mini, linux-1 still running.
func ephemeralsStopped(f *Scene) {
	for i := range f.All.Macs {
		for j, vm := range f.All.Macs[i].VMs {
			if vm.Name == "agent-7f" || vm.Name == "linux-3" {
				f.All.Macs[i].VMs[j].State = model.Stopped
			}
		}
	}
}

func miniSilent(f *Scene) {
	mini := &f.All.Macs[1]
	mini.Reachable, mini.Answered = false, storyNow.Add(-40*time.Second)
	f.Away = nil
}
