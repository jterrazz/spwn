package pages

import (
	"time"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

// value: the server stories' worlds: infra as its page reads it (its
// check on the site it serves, an hour of check runs, a month of record
// with two restarts), then down, restarting at step 2 and at step 4,
// healthy again, and stopped by its owner.
func serverWorlds() []World {
	page := []string{"machine.infra"}
	return []World{
		{Name: "machines.server", Edit: infraServes, Pages: page},
		{Name: "machines.server.down", Edit: edits{infraServes}.then(infraDown, infraSeenAt(4*time.Minute), infraDownToday), Pages: page},
		{Name: "machines.server.restarting-2", Edit: edits{infraServes}.then(infraDown, infraDownToday, infraWalking(2, time.Second)), Pages: page},
		{Name: "machines.server.restarting-4", Edit: edits{infraServes}.then(infraDown, infraDownToday, infraWalking(4, 3*time.Second)), Pages: page},
		{Name: "machines.server.recovered", Edit: edits{infraServes}.then(infraRecovered, infraBackToday), Pages: page},
		{Name: "machines.server.stopped", Edit: edits{infraServes}.then(infraStoppedByYou), Pages: page},
	}
}

func infra(f *Scene) *model.VM { return &f.All.Macs[1].VMs[0] }

// value: an hour of check runs a minute apart, 34 to 42 ms with one
// peak of 98 ms 34 min ago, and a record kept for two months that holds
// two restarts of 4 min each, 20 and 8 days ago.
func infraServes(f *Scene) {
	vm := infra(f)
	vm.Check = "https://jterrazz.com/healthz"
	vm.Latency = nil
	bump := map[int]time.Duration{24: 64, 25: 98, 26: 91, 27: 58}
	for i := range model.LatencyPoints {
		took := time.Duration(34+(i*7)%9) * time.Millisecond
		if ms, ok := bump[i]; ok {
			took = ms * time.Millisecond
		}
		vm.Latency = append(vm.Latency, model.LatencyPoint{At: storyNow.Add(time.Duration(i-model.LatencyPoints+1) * time.Minute), Took: took})
	}
	u := model.Uptime{Since: storyNow.AddDate(0, 0, -60)}
	for _, ago := range []int{20, 8} {
		from := storyNow.AddDate(0, 0, -ago).Add(-6 * time.Hour)
		u, _ = u.Marked(model.UptimeMark{Server: vm.Name, Change: model.WentDown, At: from, Cause: "out of memory"})
		u, _ = u.Marked(model.UptimeMark{Server: vm.Name, Change: model.CameBack, At: from.Add(4 * time.Minute), Restarted: true})
	}
	vm.Uptime = u
}

func infraDownToday(f *Scene) {
	vm := infra(f)
	vm.Latency = nil
	vm.Uptime, _ = vm.Uptime.Marked(model.UptimeMark{Server: vm.Name, Change: model.WentDown, At: vm.CheckStatus.Since, Cause: vm.Cause})
}

func infraSeenAt(ago time.Duration) func(*Scene) {
	return func(f *Scene) { f.Samples["infra"] = readingsAt(storyNow.Add(-ago), 24*gb, 80e9, 44e9) }
}

func infraWalking(step int, since time.Duration) func(*Scene) {
	return func(f *Scene) {
		vm := infra(f)
		vm.State, vm.IP = model.Running, "192.168.64.7"
		vm.Restart = &model.RestartWalk{Step: step, At: storyNow.Add(-since)}
		vm.BootAt = storyNow.Add(-20 * time.Second)
	}
}

func infraBackToday(f *Scene) {
	vm := infra(f)
	from := storyNow.Add(-2 * time.Minute)
	vm.Uptime, _ = vm.Uptime.Marked(model.UptimeMark{Server: vm.Name, Change: model.WentDown, At: from, Cause: "out of memory"})
	vm.Uptime, _ = vm.Uptime.Marked(model.UptimeMark{Server: vm.Name, Change: model.CameBack, At: from.Add(time.Minute), Restarted: true})
	vm.CheckStatus.At = storyNow.Add(-time.Second)
}

func infraStoppedByYou(f *Scene) {
	vm := infra(f)
	vm.State, vm.Want, vm.BootAt, vm.IP, vm.SSH, vm.Guest = model.Stopped, "stopped", time.Time{}, "", false, ""
	vm.Latency = nil
	for i, h := range f.Ports.Macs {
		for j, p := range h.Ports {
			if p.Machine.Name == vm.Name {
				f.Ports.Macs[i].Ports[j].Answer = ports.Unreachable
			}
		}
	}
}
