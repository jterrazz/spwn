package pages

import (
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func workspaceWorlds() []World {
	return []World{
		{Name: "machines.started", Edit: apiStarted, Pages: []string{"machine.api"}},
	}
}

func ago(d time.Duration) time.Time { return storyNow.Add(-d) }

func apiStarted(f *Scene) {
	for i, vm := range f.All.Macs[0].VMs {
		if vm.Name == "api" {
			f.All.Macs[0].VMs[i].State, f.All.Macs[0].VMs[i].BootAt, f.All.Macs[0].VMs[i].Guest = model.Running, ago(10*time.Second), "connected"
		}
	}
	f.Samples["api"] = readingsAt(storyNow, 8*gb, 40e9, 36e9)
}
