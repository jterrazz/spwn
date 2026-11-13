package pages

import (
	"math"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
)

func activityWorlds() []World {
	pro, mini := "machine."+proName, "machine."+miniMac
	return []World{
		{Name: "machines.history", Edit: macs().then(busyProcesses), Pages: []string{pro, mini, "machine.linux-1", "machine.infra"}},
	}
}

type bump struct {
	ago, width time.Duration
	height     float64
}

// value: how a machine's CPU and network went over the day the story
// tells: a level, its slow swing, the bumps it had; mac-mini ran linux-3's
// tests at 03:10 and stayed above 90 % for half an hour.
type profile struct {
	base, swing, seed float64
	bumps             []bump
	plateau           *[2]time.Time
	in, out           float64
}

func profiles() map[string]profile {
	day := storyNow.Truncate(24 * time.Hour)
	tests := [2]time.Time{day.Add(2*time.Hour + 58*time.Minute), day.Add(3*time.Hour + 32*time.Minute)}
	return map[string]profile{
		proName:      {base: 22, swing: 4, seed: 3, bumps: []bump{{25 * time.Minute, 4 * time.Minute, 24}, {50 * time.Minute, 5 * time.Minute, 8}}, in: 0.5, out: 0.45},
		miniMac:      {base: 41, swing: 4, seed: 11, bumps: []bump{{30 * time.Minute, 3 * time.Minute, 14}}, plateau: &tests, in: 0.3, out: 0.25},
		"linux-1":    {base: 34, swing: 6, seed: 5, bumps: []bump{{15 * time.Minute, 4 * time.Minute, 36}, {52 * time.Minute, 4 * time.Minute, 10}}, in: 0.45, out: 0.3},
		"agent-7f":   {base: 4, swing: 1, seed: 8, in: 0.05, out: 0.02},
		"infra":      {base: 15, swing: 3, seed: 13, bumps: []bump{{38 * time.Minute, 4 * time.Minute, 26}}, in: 0.55, out: 0.2},
		"linux-3":    {base: 50, swing: 8, seed: 17, in: 0.2, out: 0.1},
		"review-bot": {base: 3, swing: 1, seed: 19, in: 0.02, out: 0.01},
	}
}

func (p profile) cpu(t time.Time) float64 {
	if p.plateau != nil && !t.Before(p.plateau[0]) && !t.After(p.plateau[1]) {
		if t.Equal(p.plateau[0].Add(12 * time.Minute)) {
			return 94.4
		}
		return 94
	}
	m := t.Sub(storyNow).Minutes()
	sec := t.Sub(storyNow).Seconds()
	v := p.base + p.swing*math.Sin(m/7+p.seed) + p.swing/2*math.Sin(m/2.3+2*p.seed) + p.swing/2*math.Sin(sec/23+p.seed) + p.swing/3*math.Sin(sec/9+2*p.seed)
	for _, b := range p.bumps {
		d := t.Sub(storyNow.Add(-b.ago)).Minutes() / b.width.Minutes()
		v += b.height * math.Exp(-d*d)
	}
	return min(max(v, 1), 99)
}

// value: the network's rates in bytes per second: a steady level and a
// burst received every 7 min, sent every 5.
func (p profile) rates(t time.Time) (in, out float64) {
	m := t.Sub(storyNow).Minutes()
	burst := func(every float64) float64 {
		d := math.Mod(math.Abs(m)+every, every) - every/2
		return math.Exp(-d * d * 4)
	}
	sway := func(seed float64) float64 { return 1 + 0.2*math.Sin(m/3.7+seed) + 0.12*math.Sin(m*1.3+2*seed) }
	return p.in * (sway(p.seed) + 1.2*burst(7)) * MB, p.out * (sway(p.seed+1) + 0.8*burst(5)) * MB
}

const MB = 1e6

// value: the points a profile gives every `every` over `span` up to now,
// oldest first, its counters accumulating each interval's share and rates.
func (p profile) points(span, every time.Duration) []activity.Sample {
	n := int(span / every)
	out := make([]activity.Sample, 0, n+1)
	s := activity.Sample{At: storyNow.Add(-span), CPUTotal: 1_000_000}
	out = append(out, s)
	for i := 1; i <= n; i++ {
		at := storyNow.Add(-span + time.Duration(i)*every)
		seconds := every.Seconds()
		in, outRate := p.rates(at)
		s = activity.Sample{At: at, CPUTotal: s.CPUTotal + uint64(seconds*1000), CPUBusy: s.CPUBusy + uint64(p.cpu(at)*seconds*10),
			NetIn: s.NetIn + uint64(in*seconds), NetOut: s.NetOut + uint64(outRate*seconds)}
		out = append(out, s)
	}
	return out
}

// value: each machine's activity as its engine keeps it: the day at a
// minute, the last 6 min at 2 s.
func sceneHistory() map[string]activity.Report {
	out := map[string]activity.Report{}
	for name, p := range profiles() {
		out[name] = activity.Report{Source: activity.FromEngine, History: p.points(24*time.Hour, activity.CoarseEvery), Graph: p.points(6*time.Minute, activity.FineEvery)}
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
