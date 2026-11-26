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

func pressureHeld(prev, x *bucket) bool {
	critical := pressureRank(PressureCritical)
	return prev.floor() == critical && x.floor() == critical
}

func swapGrowing(prev, x *bucket) bool { return x.swap.mean(x.n()) > prev.swap.mean(prev.n()) }

func busy(_, x *bucket) bool { return x.span > 0 && x.cpu.mean >= BusyCPU }

func Findings(m machine.Ref, kind machine.Kind, s *Series, busiest []Process) []finding.Finding {
	// invariant: SPEC §6.5, read from the series alone: what holds now is
	// judged on the last sample, what must last on the tier's buckets that
	// span it: memory pressure on T0's seconds, swap and a busy CPU on T2's
	// minutes; a busy CPU names the busiest process of the reading beside it.
	last, ok := s.Last()
	if !ok {
		return nil
	}
	var out []finding.Finding
	add := func(sev finding.Severity, subject, text string, fix finding.Fix) {
		out = append(out, finding.Finding{Machine: m, Section: section.Activity, Severity: sev, Subject: subject, Text: text, Fix: fix})
	}
	on := ""
	if m.On != "" {
		on = " --on " + m.Name
	}
	if last.DiskTotal > 0 {
		free := float64(last.DiskFree) / float64(last.DiskTotal)
		fix := finding.Fix{Command: "station disk" + on, InTerminal: true}
		if kind == machine.Server {
			fix = finding.Fix{}
		}
		text := "Disk almost full: " + strconv.Itoa(int(free*100)) + " % free"
		switch {
		case free < 0.05:
			add(finding.Broken, SubjectDisk, text, fix)
		case free < 0.10:
			add(finding.NeedsALook, SubjectDisk, text, fix)
		}
	}
	if _, held := s.held(T0, pressureHeld); held >= PressureFor {
		add(finding.Broken, "memory", "Memory pressure critical", activityFix(on))
	}
	if _, held := s.held(T2, swapGrowing); held >= SwapFor {
		add(finding.NeedsALook, "swap", "Swap growing", activityFix(on))
	}
	switch {
	case last.Thermal >= ThermalCritical:
		add(finding.Broken, "thermal", "Running hot: Critical", finding.Fix{})
	case last.Thermal == ThermalSerious:
		add(finding.NeedsALook, "thermal", "Running hot: Serious", finding.Fix{})
	}
	if b := last.Battery; b != nil && b.OnBattery && b.Percent <= LowBattery {
		add(finding.NeedsALook, "battery", "Battery low: "+strconv.Itoa(b.Percent)+" %", finding.Fix{})
		out[len(out)-1].Detail = "Plug in " + m.Name
	}
	if run, held := s.held(T2, busy); kind == machine.Server && held >= BusyFor {
		var sum, span float64
		for _, b := range run {
			sum += float64(b.cpu.mean) * float64(b.span)
			span += float64(b.span)
		}
		mean := sum / span
		newest := last
		add(finding.NeedsALook, "cpu", m.Name+" busy: "+strconv.Itoa(int(mean+0.5))+" % for "+strconv.Itoa(int(held.Minutes()))+" min", activityFix(on))
		if len(busiest) > 0 {
			top := busiest[0]
			holds := top.Name + " holds " + strconv.Itoa(int(top.Value+0.5)) + " % of "
			if newest.CPUs > 0 {
				holds += "its " + strconv.Itoa(newest.CPUs) + " CPUs"
			} else {
				holds += "its CPU"
			}
			out[len(out)-1].Detail = holds
		}
	}
	return out
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

func activityFix(on string) finding.Fix {
	return finding.Fix{Command: "station activity" + on, InTerminal: true}
}
