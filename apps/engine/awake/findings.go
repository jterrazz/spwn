package awake

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

func Findings(r Report, m machine.Ref) []finding.Finding {
	// invariant: a guard's end needs a look for StoppedFor; a keep-awake on
	// battery near its end is said before the guard ends it, on a Mac that
	// reports a battery.
	f := finding.Finding{Machine: m, Section: section.Awake, Severity: finding.NeedsALook, Subject: "awake"}
	switch r.Mode() {
	case StoppedByGuard:
		f.Text = "Awake stopped: " + r.LastEnd.Detail
		f.Fix = finding.Fix{Command: "station awake"}
		return []finding.Finding{f}
	case AwakeUntil:
		rd := r.Reading
		if r.Caps.Battery && rd.OnBattery() && rd.Percent <= BatteryWarns && rd.Percent > BatteryEnds {
			f.Text = "Awake: battery " + strconv.Itoa(rd.Percent) + " %, it stops at " + strconv.Itoa(BatteryEnds) + " %"
			return []finding.Finding{f}
		}
	case NeverSleeps, SleepsNormally:
	}
	return nil
}
