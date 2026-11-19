package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
)

func activityWorlds() []World {
	pro, mini := "machine."+proName, "machine."+miniMac
	return []World{
		{Name: "machines.history", Edit: macs().then(busyProcesses), Pages: []string{pro, mini, "machine.linux-1", "machine.infra"}},
	}
}

// value: how each machine's CPU and network went over the day the story
// tells; mac-mini ran linux-3's tests at 03:10 and stayed above 90 % for
// half an hour.
func profiles() map[string]activity.Profile {
	day := storyNow.Truncate(24 * time.Hour)
	tests := [2]time.Time{day.Add(2*time.Hour + 58*time.Minute), day.Add(3*time.Hour + 32*time.Minute)}
	return map[string]activity.Profile{
		proName:      {Base: 22, Swing: 4, Seed: 3, Bumps: []activity.Bump{{Ago: 25 * time.Minute, Width: 4 * time.Minute, Height: 24}, {Ago: 50 * time.Minute, Width: 5 * time.Minute, Height: 8}}, In: 0.5, Out: 0.45},
		miniMac:      {Base: 41, Swing: 4, Seed: 11, Bumps: []activity.Bump{{Ago: 30 * time.Minute, Width: 3 * time.Minute, Height: 14}}, Plateau: &tests, In: 0.3, Out: 0.25},
		"linux-1":    {Base: 34, Swing: 6, Seed: 5, Bumps: []activity.Bump{{Ago: 15 * time.Minute, Width: 4 * time.Minute, Height: 36}, {Ago: 52 * time.Minute, Width: 4 * time.Minute, Height: 10}}, In: 0.45, Out: 0.3},
		"agent-7f":   {Base: 4, Swing: 1, Seed: 8, In: 0.05, Out: 0.02},
		"infra":      {Base: 15, Swing: 3, Seed: 13, Bumps: []activity.Bump{{Ago: 38 * time.Minute, Width: 4 * time.Minute, Height: 26}}, In: 0.55, Out: 0.2},
		"linux-3":    {Base: 50, Swing: 8, Seed: 17, In: 0.2, Out: 0.1},
		"review-bot": {Base: 3, Swing: 1, Seed: 19, In: 0.02, Out: 0.01},
	}
}

// value: each machine's activity as its engine keeps it: the day at a
// minute, the last 6 min at 2 s.
func sceneHistory() map[string]activity.Report {
	out := map[string]activity.Report{}
	for name, p := range profiles() {
		out[name] = activity.Report{Source: activity.FromEngine, History: p.Points(storyNow, 24*time.Hour, activity.CoarseEvery), Graph: p.Points(storyNow, 6*time.Minute, activity.FineEvery)}
	}
	return out
}

// value: the story's busiest processes, and each machine's live readings
// as the end of its kept 6 min, so the value now is where its chart ends.
func busyProcesses(f *Scene) {
	top := map[string][]activity.Process{
		proName:   {{PID: 501, Name: "Xcode", Value: 9}, {PID: 502, Name: "Safari", Value: 4}, {PID: 812, Name: "station", Value: 1}},
		miniMac:   {{PID: 601, Name: "mds_stores", Value: 2}, {PID: 812, Name: "station", Value: 1}},
		"linux-1": {{PID: 701, Name: "claude", Value: 12}, {PID: 702, Name: "node (vite)", Value: 9}, {PID: 703, Name: "station", Value: 1}},
		"infra":   {{PID: 801, Name: "k3s-server", Value: 9}, {PID: 802, Name: "traefik", Value: 4}, {PID: 803, Name: "station", Value: 1}},
	}
	share := map[string]float64{"linux-1": 12, "agent-7f": 1, "linux-3": 22, "infra": 9, "review-bot": 1}
	for name, readings := range f.Samples {
		kept := f.History[name].Graph
		for i := range readings {
			r := &readings[i]
			if k := len(kept) - len(readings) + i; k >= 0 {
				memory := r.Sample
				r.Sample = kept[k]
				r.Sample.MemUsed, r.Sample.MemTotal, r.Sample.DiskFree, r.Sample.DiskTotal = memory.MemUsed, memory.MemTotal, memory.DiskFree, memory.DiskTotal
			}
			if t, ok := top[name]; ok {
				r.Top.CPU = t
			}
			for j := range r.VMs {
				r.VMs[j].CPU = share[r.VMs[j].VM]
			}
		}
	}
}
