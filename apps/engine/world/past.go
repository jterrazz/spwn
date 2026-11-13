package world

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
)

func (m Meter) Answers(now time.Time) activity.Sample {
	// value: the sample the machine's kernel answers at now: Sample's
	// counters grown by PerSecond since Sample's time, and none of what
	// another family composes in (thermal, battery, free space).
	s := m.Sample
	if m.PerSecond != nil && !s.At.IsZero() {
		seconds := max(now.Sub(s.At).Seconds(), 0)
		by := func(n, per uint64) uint64 { return n + uint64(float64(per)*seconds) }
		rate := *m.PerSecond
		s.CPUBusy, s.CPUTotal = by(s.CPUBusy, rate.CPUBusy), by(s.CPUTotal, rate.CPUTotal)
		s.DiskRead, s.DiskWritten = by(s.DiskRead, rate.DiskRead), by(s.DiskWritten, rate.DiskWritten)
		s.NetIn, s.NetOut = by(s.NetIn, rate.NetIn), by(s.NetOut, rate.NetOut)
		s.SwapUsed += int64(float64(rate.SwapUsed) * seconds)
	}
	s.At = now
	s.Thermal, s.Battery, s.DiskFree, s.DiskTotal = activity.ThermalUnknown, nil, 0, 0
	return s
}

func (w *World) unfold() {
	// invariant: an engine's Past becomes the activity it keeps, once, at
	// the world's now: a machine it already keeps is left as it is, and the
	// profiles are spent, so a successor after a restart holds none.
	if w.Now.IsZero() {
		return
	}
	for mac, m := range w.Macs {
		e := m.Engine
		if e == nil || e.Past == nil {
			continue
		}
		for name, p := range e.Past {
			if _, kept := e.Activity[name]; kept {
				continue
			}
			on := w.Macs[name]
			if on == nil || on.Meter == nil {
				continue
			}
			if e.Activity == nil {
				e.Activity = map[string]activity.Kept{}
			}
			e.Activity[name] = w.past(p, *on.Meter, e.StartedAt, w.runsOf(mac, name))
		}
		e.Past = nil
	}
}

func (w *World) runsOf(mac, name string) map[int]string {
	// value: the VMs a Mac runs, by the pid that holds each; none for a VM.
	if mac != name {
		return nil
	}
	out := map[int]string{}
	for vm, v := range w.Macs[mac].VMs {
		if v.Running() {
			out[v.PID] = vm
		}
	}
	return out
}

func (w *World) past(p activity.Profile, meter Meter, started time.Time, runs map[int]string) activity.Kept {
	// invariant: never before the engine started nor the machine booted,
	// and its newest point is the very sample the meter answers at now, so
	// the engine's first live sample after it reads no reset nor jump.
	now := w.Now
	answer := meter.Answers(now)
	from := now.Add(-activity.CoarseEvery * (activity.CoarsePoints - 1))
	for _, t := range []time.Time{started, answer.Boot} {
		if t.After(from) {
			from = t
		}
	}
	span := func(every time.Duration, points int) time.Duration {
		return min(max(now.Sub(from), 0), every*time.Duration(points-1)).Truncate(every)
	}
	coarse := aligned(p.Points(now, span(activity.CoarseEvery, activity.CoarsePoints), activity.CoarseEvery), answer)
	fine := aligned(p.Points(now, span(activity.FineEvery, activity.FinePoints), activity.FineEvery), answer)
	last := activity.Reading{Sample: answer, Top: activity.Busiest{CPU: activity.Ranked(meter.Top.CPU), Memory: activity.Ranked(meter.Top.Memory)}}
	pids := make([]int, 0, len(runs))
	for pid := range runs {
		pids = append(pids, pid)
	}
	slices.Sort(pids)
	for _, pid := range pids {
		if u, ok := meter.Processes[pid]; ok {
			last.VMs = append(last.VMs, activity.VMUsage{VM: runs[pid], Usage: u})
		}
	}
	for _, list := range [][]activity.Process{last.Top.CPU, last.Top.Memory} {
		for i, proc := range list {
			if vm, ok := runs[proc.PID]; ok {
				list[i].VM = vm
			}
		}
	}
	return activity.Kept{Last: last, Series: activity.SeriesOf(fine, coarse)}
}

func aligned(points []activity.Sample, answer activity.Sample) []activity.Sample {
	// invariant: each point is the answer at its own time, its counters the
	// answer's less what the profile counted since: a counter that would go
	// below zero is before the machine counted, and those points are dropped.
	if len(points) == 0 {
		return nil
	}
	newest := points[len(points)-1]
	out := make([]activity.Sample, len(points))
	for i := len(points) - 1; i >= 0; i-- {
		x, ok := answer, true
		back := func(at, newer, older uint64) uint64 {
			counted := newer - older
			if at < counted {
				ok = false
				return 0
			}
			return at - counted
		}
		x.At = points[i].At
		x.CPUBusy = back(answer.CPUBusy, newest.CPUBusy, points[i].CPUBusy)
		x.CPUTotal = back(answer.CPUTotal, newest.CPUTotal, points[i].CPUTotal)
		x.DiskRead = back(answer.DiskRead, newest.DiskRead, points[i].DiskRead)
		x.DiskWritten = back(answer.DiskWritten, newest.DiskWritten, points[i].DiskWritten)
		x.NetIn = back(answer.NetIn, newest.NetIn, points[i].NetIn)
		x.NetOut = back(answer.NetOut, newest.NetOut, points[i].NetOut)
		if !ok {
			return out[i+1:]
		}
		out[i] = x
	}
	return out
}
