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
	here   = machine.Ref{Name: "macbook-pro"}
	linux1 = machine.Ref{Name: "linux-1", On: "macbook-pro"}
	infra  = machine.Ref{Name: "infra", On: "mac-mini"}
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
	if got := activity.Findings(here, machine.Computer, s, nil); len(got) != 0 {
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
		{infra, machine.Server, 4e9, []line{{finding.Broken, "disk", ""}}},
		{here, machine.Computer, 11e9, []line{}},
	}
	for _, c := range cases {
		if got := lines(activity.Findings(c.m, c.kind, free(c.free), nil)); !slices.Equal(got, c.want) {
			t.Errorf("%s with %d bytes free of 100 GB: Findings() = %v; want %v", c.m.Name, c.free, got, c.want)
		}
	}
}

func TestADiskAlmostFullOfAMacIsAFilledLineForTheTerminalAndAServersIsASentence(t *testing.T) {
	full := series(2*time.Second, 1, func(_ int, s *activity.Sample) { s.DiskFree, s.DiskTotal = 4e9, 100e9 })
	if got := activity.Findings(here, machine.Computer, full, nil); len(got) != 1 || !got[0].Fix.InTerminal || got[0].Fix.Command != "station disk" {
		t.Errorf("a Mac's disk: %+v; want its command filled for the terminal", got)
	}
	if got := activity.Findings(infra, machine.Server, full, nil); len(got) != 1 || got[0].Fix.Call+got[0].Fix.Command+got[0].Fix.Path != "" {
		t.Errorf("a server's disk: %+v; want a sentence with no fix", got)
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

	if got := lines(activity.Findings(here, machine.Computer, held, nil)); !slices.Equal(got, []line{{finding.Broken, "memory", "station activity"}}) {
		t.Errorf("critical for 60 s: Findings() = %v; want memory broken", got)
	}
	if got := activity.Findings(here, machine.Computer, brief, nil); len(got) != 0 {
		t.Errorf("critical for 58 s: Findings() = %v; want none", got)
	}
}

func TestSwapGrowingForTenMinutesStraightNeedsALook(t *testing.T) {
	growingFor := func(minutes int) *activity.Series {
		return series(time.Minute, 12, func(i int, s *activity.Sample) { s.SwapUsed = int64(max(i, 11-minutes)) * 100e6 })
	}

	if got := lines(activity.Findings(linux1, machine.Workspace, growingFor(10), nil)); !slices.Equal(got, []line{{finding.NeedsALook, "swap", "station activity --on linux-1"}}) {
		t.Errorf("swap grew 10 min: Findings() = %v; want swap growing", got)
	}
	if got := activity.Findings(linux1, machine.Workspace, growingFor(9), nil); len(got) != 0 {
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
		if got := lines(activity.Findings(here, machine.Computer, s, nil)); !slices.Equal(got, want) {
			t.Errorf("thermal %d: Findings() = %v; want %v", level, got, want)
		}
	}
}

func TestABatteryAtTwentyPercentOnBatteryNeedsALook(t *testing.T) {
	low := series(2*time.Second, 1, func(_ int, s *activity.Sample) { s.Battery = &activity.Battery{Percent: 20, OnBattery: true} })
	charging := series(2*time.Second, 1, func(_ int, s *activity.Sample) { s.Battery = &activity.Battery{Percent: 12, Charging: true} })

	if got := lines(activity.Findings(here, machine.Computer, low, nil)); !slices.Equal(got, []line{{finding.NeedsALook, "battery", ""}}) {
		t.Errorf("20 %% on battery: Findings() = %v; want battery low", got)
	}
	if got := activity.Findings(here, machine.Computer, low, nil); got[0].Detail != "Plug in "+here.Name {
		t.Errorf("battery detail %q; want the sentence that says to plug in", got[0].Detail)
	}
	if got := activity.Findings(here, machine.Computer, charging, nil); len(got) != 0 {
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
			s.CPUBusy, s.CPUTotal, s.CPUs = busy, uint64(i)*100, 4 //nolint:gosec // value: i counts up from zero.
		})
	}

	got := activity.Findings(infra, machine.Server, busyFor(96, 10), []activity.Process{{Name: "go test", Value: 97}})
	if !slices.Equal(lines(got), []line{{finding.NeedsALook, "cpu", "station activity --on infra"}}) || got[0].Text != "infra busy: 96 % for 10 min" {
		t.Errorf("server at 96 %% for 10 min: Findings() = %+v; want infra busy: 96 %% for 10 min", got)
	}
	if got[0].Detail != "go test holds 97 % of its 4 CPUs" {
		t.Errorf("detail = %q; want the busiest process named with its share of the CPUs", got[0].Detail)
	}
	if quiet := activity.Findings(infra, machine.Server, busyFor(96, 10), nil); quiet[0].Detail != "" {
		t.Errorf("detail = %q with no process read; want none", quiet[0].Detail)
	}
	for _, c := range []struct {
		percent uint64
		minutes int
	}{{94, 10}, {96, 9}} {
		if got := activity.Findings(infra, machine.Server, busyFor(c.percent, c.minutes), nil); len(got) != 0 {
			t.Errorf("server at %d %% for %d min: Findings() = %v; want none", c.percent, c.minutes, got)
		}
	}
	if got := activity.Findings(linux1, machine.Workspace, busyFor(96, 11), nil); len(got) != 0 {
		t.Errorf("workspace at 96 %% for 11 min: Findings() = %v; want none", got)
	}
}
