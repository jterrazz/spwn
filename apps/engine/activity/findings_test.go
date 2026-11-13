package activity_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

var (
	start  = time.Unix(1_800_000_000, 0)
	here   = machine.Ref{Name: "jterrazz-macbook-pro"}
	linux1 = machine.Ref{Name: "linux-1", On: "jterrazz-macbook-pro"}
	infra  = machine.Ref{Name: "infra", On: "jterrazz-mac-mini"}
)

func calm(at time.Duration) activity.Sample {
	return activity.Sample{At: start.Add(at), Pressure: activity.PressureNormal, Thermal: 0, DiskFree: 388e9, DiskTotal: 994e9}
}

func series(every time.Duration, n int, change func(i int, s *activity.Sample)) *activity.Series {
	s := &activity.Series{}
	for i := range n {
		x := calm(time.Duration(i) * every)
		change(i, &x)
		s.Add(x)
	}
	return s
}

type line struct {
	Severity     finding.Severity
	Subject, Fix string
}

func lines(fs []finding.Finding) []line {
	out := make([]line, 0, len(fs))
	for _, f := range fs {
		out = append(out, line{f.Severity, f.Subject, f.Fix.Command})
	}
	return out
}

func TestACalmMachineNeedsNothing(t *testing.T) {
	s := series(2*time.Second, 400, func(int, *activity.Sample) {})
	if got := activity.Findings(here, machine.Computer, s); len(got) != 0 {
		t.Errorf("Findings() = %+v; want none", got)
	}
}

func TestADiskAlmostFullNeedsALookUnderTenPercentAndIsBrokenUnderFive(t *testing.T) {
	free := func(bytes int64) *activity.Series {
		return series(2*time.Second, 1, func(_ int, s *activity.Sample) { s.DiskFree, s.DiskTotal = bytes, 100e9 })
	}
	cases := []struct {
		m    machine.Ref
		kind machine.Kind
		free int64
		want []line
	}{
		{here, machine.Computer, 8e9, []line{{finding.NeedsALook, "disk", "station disk"}}},
		{linux1, machine.Workspace, 4e9, []line{{finding.Broken, "disk", "station disk --on linux-1"}}},
		{infra, machine.Server, 4e9, []line{{finding.Broken, "disk", "station machines edit infra --disk <size> --on jterrazz-mac-mini"}}},
		{here, machine.Computer, 11e9, []line{}},
	}
	for _, c := range cases {
		if got := lines(activity.Findings(c.m, c.kind, free(c.free))); !slices.Equal(got, c.want) {
			t.Errorf("%s with %d bytes free of 100 GB: Findings() = %v; want %v", c.m.Name, c.free, got, c.want)
		}
	}
}

func TestMemoryPressureIsBrokenOnceCriticalForAMinute(t *testing.T) {
	critical := func(from int) func(int, *activity.Sample) {
		return func(i int, s *activity.Sample) {
			if i >= from {
				s.Pressure = activity.PressureCritical
			}
		}
	}
	held := series(2*time.Second, 100, critical(69))
	brief := series(2*time.Second, 100, critical(70))

	if got := lines(activity.Findings(here, machine.Computer, held)); !slices.Equal(got, []line{{finding.Broken, "memory", "station activity"}}) {
		t.Errorf("critical for 60 s: Findings() = %v; want memory broken", got)
	}
	if got := activity.Findings(here, machine.Computer, brief); len(got) != 0 {
		t.Errorf("critical for 58 s: Findings() = %v; want none", got)
	}
}

func TestSwapGrowingForTenMinutesStraightNeedsALook(t *testing.T) {
	growingFor := func(minutes int) *activity.Series {
		return series(time.Minute, 12, func(i int, s *activity.Sample) { s.SwapUsed = int64(max(i, 11-minutes)) * 100e6 })
	}

	if got := lines(activity.Findings(linux1, machine.Workspace, growingFor(10))); !slices.Equal(got, []line{{finding.NeedsALook, "swap", "station activity --on linux-1"}}) {
		t.Errorf("swap grew 10 min: Findings() = %v; want swap growing", got)
	}
	if got := activity.Findings(linux1, machine.Workspace, growingFor(9)); len(got) != 0 {
		t.Errorf("swap grew 9 min: Findings() = %v; want none", got)
	}
}

func TestRunningHotNeedsALookWhenSeriousAndIsBrokenWhenCritical(t *testing.T) {
	for level, want := range map[int][]line{
		1:                        {},
		activity.ThermalSerious:  {{finding.NeedsALook, "thermal", ""}},
		activity.ThermalCritical: {{finding.Broken, "thermal", ""}},
		activity.ThermalUnknown:  {},
	} {
		s := series(2*time.Second, 1, func(_ int, s *activity.Sample) { s.Thermal = level })
		if got := lines(activity.Findings(here, machine.Computer, s)); !slices.Equal(got, want) {
			t.Errorf("thermal %d: Findings() = %v; want %v", level, got, want)
		}
	}
}

func TestABatteryAtTwentyPercentOnBatteryNeedsALook(t *testing.T) {
	low := series(2*time.Second, 1, func(_ int, s *activity.Sample) { s.Battery = &activity.Battery{Percent: 20, OnBattery: true} })
	charging := series(2*time.Second, 1, func(_ int, s *activity.Sample) { s.Battery = &activity.Battery{Percent: 12, Charging: true} })

	if got := lines(activity.Findings(here, machine.Computer, low)); !slices.Equal(got, []line{{finding.NeedsALook, "battery", "plug in"}}) {
		t.Errorf("20 %% on battery: Findings() = %v; want battery low", got)
	}
	if got := activity.Findings(here, machine.Computer, charging); len(got) != 0 {
		t.Errorf("12 %% charging: Findings() = %v; want none", got)
	}
}

func TestAServerBusyForTenMinutesNeedsALookAWorkspaceNever(t *testing.T) {
	busyFor := func(percent uint64, minutes int) *activity.Series {
		var busy uint64
		return series(time.Minute, 12, func(i int, s *activity.Sample) {
			if i > 0 {
				busy += 10
				if i > 11-minutes {
					busy += percent - 10
				}
			}
			s.CPUBusy, s.CPUTotal = busy, uint64(i)*100 //nolint:gosec // value: i counts up from zero.
		})
	}

	got := activity.Findings(infra, machine.Server, busyFor(96, 10))
	if !slices.Equal(lines(got), []line{{finding.NeedsALook, "cpu", "station activity --on infra"}}) || got[0].Text != "infra busy: 96 % for 10 min" {
		t.Errorf("server at 96 %% for 10 min: Findings() = %+v; want infra busy: 96 %% for 10 min", got)
	}
	for _, c := range []struct {
		percent uint64
		minutes int
	}{{94, 10}, {96, 9}} {
		if got := activity.Findings(infra, machine.Server, busyFor(c.percent, c.minutes)); len(got) != 0 {
			t.Errorf("server at %d %% for %d min: Findings() = %v; want none", c.percent, c.minutes, got)
		}
	}
	if got := activity.Findings(linux1, machine.Workspace, busyFor(96, 11)); len(got) != 0 {
		t.Errorf("workspace at 96 %% for 11 min: Findings() = %v; want none", got)
	}
}

func TestTheHistoryKeepsSixMinutesFineAndADayCoarse(t *testing.T) {
	s := series(2*time.Second, 3*60*60/2, func(int, *activity.Sample) {})
	fine, coarse := s.Fine(), s.Coarse()

	if len(fine) != activity.FinePoints || fine[len(fine)-1].At.Sub(fine[0].At) != 358*time.Second {
		t.Errorf("Fine() holds %d points; want the last %d, 2 s apart", len(fine), activity.FinePoints)
	}
	if len(coarse) != 180 || coarse[1].At.Sub(coarse[0].At) != time.Minute {
		t.Errorf("Coarse() holds %d points; want one a minute over 3 h", len(coarse))
	}
	day := series(time.Minute, 2000, func(int, *activity.Sample) {})
	if got := day.Coarse(); len(got) != activity.CoarsePoints || got[0].At != start.Add(560*time.Minute) {
		t.Errorf("Coarse() over 2000 min holds %d points from %v; want the last %d", len(got), got[0].At, activity.CoarsePoints)
	}
	if got := day.Fine(); len(got) != 7 {
		t.Errorf("Fine() unwatched holds %d points; want only the minutes of the last 6 min", len(got))
	}
}
