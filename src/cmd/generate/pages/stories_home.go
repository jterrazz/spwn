package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
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
		home("machines.home.one-wait", sharesLong),
		home("machines.home.share-ending", reviewMerged),
		home("machines.home.many", manyNeeds),
		home("machines.home.fine", nothingNeedsYou),
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

func sharesLong(f *Scene) {
	for i, m := range f.Accounts.Machines {
		for j, a := range m.Accounts {
			if a.Shared != nil {
				f.Accounts.Machines[i].Accounts[j] = loan(a.Provider, a.Shared.Name, 6*time.Hour)
			}
		}
	}
}

func reviewMerged(f *Scene) {
	bot := &f.All.Macs[1].VMs[2]
	bot.Agent = &model.AgentView{State: model.AgentFinished, Task: bot.Agent.Task, Ended: storyNow.Add(-3 * time.Minute), Reason: "merged PR #42"}
}

func manyNeeds(f *Scene) {
	linux3 := &f.All.Macs[1].VMs[1]
	linux3.Agent = &model.AgentView{State: "waiting", Task: linux3.Agent.Task, Ask: "asks which token store to use", Since: storyNow.Add(-2 * time.Minute)}
	for i, m := range f.Accounts.Machines {
		switch m.Name {
		case "linux-3":
			f.Accounts.Machines[i].Accounts[0] = loan(catalog.ProviderGitHub, "github", 12*time.Minute)
		case "review-bot":
			f.Accounts.Machines[i].Accounts[0] = loan(catalog.ProviderClaude, catalog.ClaudeTokenName, 17*time.Minute)
		}
	}
}

func miniSilent(f *Scene) {
	mini := &f.All.Macs[1]
	mini.Reachable, mini.Answered = false, storyNow.Add(-40*time.Second)
	f.Away = nil
}
