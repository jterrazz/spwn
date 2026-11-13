package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
)

// value: the worlds of Home's sections and of the Needs you page: the home
// scene with more agents, an agent being born, Awake off; the fleet's needs
// many, then its updates installed.
func needsWorlds() []World {
	home := func(name string, edits ...func(*Scene)) World {
		return World{Name: name, Edit: func(f *Scene) {
			homeScene(f)
			for _, e := range edits {
				e(f)
			}
		}, Pages: []string{"menubar"}}
	}
	needs := func(name string, edit func(*Scene)) World {
		return World{Name: name, Edit: edit, Pages: []string{"needs"}}
	}
	return []World{
		home("machines.home.four-agents", twoMoreAgents),
		home("machines.home.creating", linux4Creating),
		home("machines.home.born", linux4Ready, linux4Reads),
		home("machines.home.awake-off", awakeOff),
		needs("machines.needs.five", sharesEndingSoon),
		needs("machines.needs.updated", updatesInstalled),
		needs("machines.needs.catalog-refused", catalogRefused),
	}
}

func twoMoreAgents(f *Scene) {
	mini := &f.All.Macs[1]
	for _, w := range []struct{ name, task, ip string }{{"linux-5", "migrates the db schema", "192.168.64.10"}, {"linux-6", "writes the changelog", "192.168.64.11"}} {
		vm := sceneVM(w.name, miniMac, model.Ephemeral, model.Running, 2, 4, w.ip)
		vm.Agent = &model.AgentView{State: "working", Task: w.task}
		mini.VMs = append(mini.VMs, vm)
	}
}

func linux4Reads(f *Scene) {
	vms := f.All.Macs[1].VMs
	vms[len(vms)-1].Agent.Step = []string{"reading the repository"}
}

func awakeOff(f *Scene) {
	f.Awake.Session = nil
}

func sharesEndingSoon(f *Scene) {
	for i, m := range f.Accounts.Machines {
		switch m.Name {
		case "linux-1":
			f.Accounts.Machines[i].Accounts = []accounts.Account{loan(catalog.ProviderGitHub, "github", 12*time.Minute), loan(catalog.ProviderClaude, catalog.ClaudeTokenName, 12*time.Minute)}
		case "linux-3":
			f.Accounts.Machines[i].Accounts = []accounts.Account{loan(catalog.ProviderClaude, catalog.ClaudeTokenName, 12*time.Minute), loan(catalog.ProviderGitHub, "github", 12*time.Minute)}
		}
	}
}

// value: the owner's package file refused: the run reads the built-in packages,
// and Needs you names the file and its line.
func catalogRefused(f *Scene) {
	f.Home.Needs = append(f.Home.Needs, overview.Need{Machine: f.Home.Head.Name, Section: "overview", Broken: true, Severity: "broken",
		Subject: overview.RefusedSubject, Text: "~/.config/station/catalog/apps/jq.yaml:3: unknown key `vesion`", Path: "~/.config/station/catalog/apps/jq.yaml"})
}

func updatesInstalled(f *Scene) {
	var kept []overview.Need
	for _, n := range f.Home.Needs {
		if n.Call != "apps.update" {
			kept = append(kept, n)
		}
	}
	f.Home.Needs = kept
}
