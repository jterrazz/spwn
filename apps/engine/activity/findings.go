package activity

import (
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

const (
	PressureFor = time.Minute
	SwapFor     = 10 * time.Minute
	BusyFor     = 10 * time.Minute
	BusyCPU     = 95
	LowBattery  = 20
	SubjectDisk = "disk"
)

func DiskFreeTold(name string, last Sample) string {
	// value: a disk under its floor in the words told beyond its machine's
	// page (SPEC §10.4).
	return name + ": " + strconv.Itoa(int(float64(last.DiskFree)/float64(last.DiskTotal)*100)) + " % of its disk is free"
}

func Findings(m machine.Ref, kind machine.Kind, s *Series) []finding.Finding {
	// invariant: SPEC §6.5, read from the series alone: what holds now is
	// judged on the last sample, what must last on the points that span it.
	last, ok := s.Last()
	if !ok {
		return nil
	}
	var out []finding.Finding
	add := func(sev finding.Severity, subject, text, fix string) {
		out = append(out, finding.Finding{Machine: m, Section: section.Activity, Severity: sev, Subject: subject, Text: text, Fix: finding.Fix{Command: fix}})
	}
	on := ""
	if m.On != "" {
		on = " --on " + m.Name
	}
	if last.DiskTotal > 0 {
		free := float64(last.DiskFree) / float64(last.DiskTotal)
		fix := "station disk" + on
		if kind == machine.Server {
			fix = "station machines edit " + m.Name + " --disk <size>"
			if m.On != "" {
				fix += " --on " + m.On
			}
		}
		text := "Disk almost full: " + strconv.Itoa(int(free*100)) + " % free"
		switch {
		case free < 0.05:
			add(finding.Broken, SubjectDisk, text, fix)
		case free < 0.10:
			add(finding.NeedsALook, SubjectDisk, text, fix)
		}
	}
	points := merged(s)
	if heldFor(points, func(prev, x Sample) bool { return prev.Pressure == PressureCritical && x.Pressure == PressureCritical }) >= PressureFor {
		add(finding.Broken, "memory", "Memory pressure critical", "station activity"+on)
	}
	coarse := s.Coarse()
	if heldFor(coarse, func(prev, x Sample) bool { return x.SwapUsed > prev.SwapUsed }) >= SwapFor {
		add(finding.NeedsALook, "swap", "Swap growing", "station activity"+on)
	}
	switch {
	case last.Thermal >= ThermalCritical:
		add(finding.Broken, "thermal", "Running hot: Critical", "")
	case last.Thermal == ThermalSerious:
		add(finding.NeedsALook, "thermal", "Running hot: Serious", "")
	}
	if b := last.Battery; b != nil && b.OnBattery && b.Percent <= LowBattery {
		add(finding.NeedsALook, "battery", "Battery low: "+strconv.Itoa(b.Percent)+" %", "plug in")
	}
	busy := func(prev, x Sample) bool { return Between(prev, x).CPU >= BusyCPU }
	if held := heldFor(coarse, busy); kind == machine.Server && held >= BusyFor {
		newest := coarse[len(coarse)-1]
		first := newest
		for _, x := range coarse {
			if !x.At.Before(newest.At.Add(-held)) {
				first = x
				break
			}
		}
		mean := Between(first, newest).CPU
		add(finding.NeedsALook, "cpu", m.Name+" busy: "+strconv.Itoa(int(mean+0.5))+" % for "+strconv.Itoa(int(held.Minutes()))+" min", "station activity"+on)
	}
	return out
}

func merged(s *Series) []Sample {
	all := append(s.Coarse(), s.Fine()...)
	slices.SortStableFunc(all, func(a, b Sample) int { return a.At.Compare(b.At) })
	return slices.CompactFunc(all, func(a, b Sample) bool { return a.At.Equal(b.At) })
}

func heldFor(points []Sample, holds func(prev, x Sample) bool) time.Duration {
	// value: how long, back from the newest point, every point held; a
	// point is judged against the one before it.
	if len(points) < 2 {
		return 0
	}
	newest := points[len(points)-1]
	since := newest.At
	for i := len(points) - 1; i > 0 && holds(points[i-1], points[i]); i-- {
		since = points[i-1].At
	}
	return newest.At.Sub(since)
}

func Persisting(now, kept []finding.Finding) []finding.Finding {
	// invariant: what a machine reads now judges each subject it raises;
	// what must last (a CPU busy for 10 min) is judged on the series an
	// engine keeps.
	for _, f := range kept {
		if !slices.ContainsFunc(now, func(g finding.Finding) bool { return g.Subject == f.Subject }) {
			now = append(now, f)
		}
	}
	return now
}
