package pages

import (
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func shellWorlds() []World {
	return []World{
		{Name: "machines.restarting", Edit: infraRestarting, Pages: []string{"menubar", "find"}},
	}
}

func infraRestarting(f *Scene) {
	infra := &f.All.Macs[1].VMs[0]
	infra.Restart = &model.RestartWalk{Step: 2, At: storyNow.Add(-8 * time.Second)}
	infra.BootAt = storyNow.Add(-20 * time.Second)
}
