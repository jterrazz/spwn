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
		busy := by(s.CPUBusy, rate.CPUBusy)
		if m.Load != nil {
			busy = s.CPUBusy + m.Load.busy(s.At, now, rate.CPUTotal)
		}
		s.CPUBusy, s.CPUTotal = busy, by(s.CPUTotal, rate.CPUTotal)
		s.DiskRead, s.DiskWritten = by(s.DiskRead, rate.DiskRead), by(s.DiskWritten, rate.DiskWritten)
		s.NetIn, s.NetOut = by(s.NetIn, rate.NetIn), by(s.NetOut, rate.NetOut)
		s.SwapUsed += int64(float64(rate.SwapUsed) * seconds)
	}
	s.At = now
	s.Thermal, s.Battery, s.DiskFree, s.DiskTotal = activity.ThermalUnknown, nil, 0, 0
	return s
}

func (l Load) level(slot int64) uint64 {
	// invariant: a slot's level is a pure function of the seed and the slot,
	// whole-number arithmetic alone, so every host draws the same curve.
	z := l.Seed + uint64(slot)*0x9e3779b97f4a7c15 //nolint:gosec // value: a slot is after 1970.
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	floor := max(l.Floor, 0)
	span := max(l.Ceil-floor, 0) + 1
	return uint64(floor) + z%uint64(span) //nolint:gosec // value: span is at least one.
}

func (l Load) busy(from, to time.Time, perSecond uint64) uint64 {
	// value: the busy counts between two times: the rate of CPUTotal at each
	// slot's level, over the part of the slot that lies between them.
	slot := int64(60_000)
	if l.SlotSeconds > 0 {
		slot = int64(l.SlotSeconds) * 1000
	}
	end := to.UnixMilli()
	var weighted uint64
	for t := from.UnixMilli(); t < end; {
		next := min((t/slot+1)*slot, end)
		weighted += l.level(t/slot) * uint64(next-t) //nolint:gosec // value: next is after t.
		t = next
	}
	return perSecond * weighted / 1_000_000
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
			e.Activity[name] = w.past(p, *on.Meter, on.Disk, e.StartedAt, w.runsOf(mac, name))
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

func (w *World) past(p activity.Profile, meter Meter, disk *Disk, started time.Time, runs map[int]string) activity.Kept {
	// invariant: never before the engine started nor the machine booted,
	// and its newest point is the very sample the meter answers at now, so
	// the engine's first live sample after it reads no reset nor jump.
	now := w.Now
	answer := meter.Answers(now)
	// invariant: a kept point is what the machine's activity.sample
	// answered, its free space composed in from its disk as the Disk family
	// reads it, a server's as a workspace's.
	if disk != nil {
		answer.DiskTotal, answer.DiskFree = disk.Total, disk.Free
	}
	from := now.Add(-24*time.Hour + activity.IdleEvery)
	for _, t := range []time.Time{started, answer.Boot} {
		if t.After(from) {
			from = t
		}
	}
	span := func(end time.Time, every time.Duration, points int) time.Duration {
		return min(max(end.Sub(from), 0), every*time.Duration(points-1)).Truncate(every)
	}
	// invariant: the day unwatched, a sample each whole minute, up to the
	// last before its last 6 min watched, one every 2 s, as the engine
	// would have sampled them: the minutes read the profile's curve as it
	// is at now, so a peak the profile puts 25 min ago is 25 min ago, and
	// their counters run on into the first watched sample.
	watched := aligned(p.Points(now, span(now, activity.WatchedEvery, 180), activity.WatchedEvery), answer)
	points := watched
	if len(watched) > 0 {
		first := watched[0]
		end := first.At.Truncate(activity.IdleEvery)
		if gap := first.At.Sub(end); gap > 0 {
			if bridge := aligned(p.PointsTo(first.At, now, gap, gap), first); len(bridge) == 2 {
				minutes := aligned(p.PointsTo(end, now, span(end, activity.IdleEvery, 1440), activity.IdleEvery), bridge[0])
				points = slices.Concat(minutes[:max(len(minutes)-1, 0)], watched)
			}
		} else {
			points = append(aligned(p.PointsTo(end, now, span(end, activity.IdleEvery, 1440), activity.IdleEvery), first), watched[1:]...)
		}
	}
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
	return activity.Kept{Last: last, Series: activity.SeriesOf(points)}
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
