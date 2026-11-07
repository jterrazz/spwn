package calls

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/core/machine"
)

type processes map[int]activity.Usage

func (processes) Sample(context.Context) (activity.Sample, error) { return activity.Sample{}, nil }
func (p processes) Top(context.Context) (activity.Busiest, error) {
	b := activity.Busiest{CPU: []activity.Process{}, Memory: []activity.Process{}}
	for pid, u := range p {
		b.CPU = append(b.CPU, activity.Process{PID: pid, Name: "com.apple.Virtualization.Virtual", Value: u.CPU})
	}
	return b, nil
}
func (p processes) Process(_ context.Context, pid int) (activity.Usage, error) {
	if u, ok := p[pid]; ok {
		return u, nil
	}
	return activity.Usage{}, errors.New("no such process")
}

func TestAVMsRunIsMeasuredOnItsMacByTheProcessThatHoldsItsGuest(t *testing.T) {
	now := time.Date(2026, 11, 6, 4, 0, 0, 0, time.UTC)
	d := Deps{
		Meter: processes{4100: {CPU: 0, Memory: 1 << 20}, 4242: {CPU: 18, Memory: 15 << 30}},
		Name:  "jterrazz-mac-mini", Kind: machine.Mac, Now: func() time.Time { return now },
		Runs: func(context.Context) ([]activity.VMRun, error) {
			return []activity.VMRun{{VM: "infra", Holder: 4242}}, nil
		},
	}
	r := d.read(t.Context(), ReadIn{Of: "infra", Machine: machine.Ref{Name: "infra", On: "jterrazz-mac-mini"}, Kind: machine.Server, Run: true})
	if r.Source != activity.FromMacProcess || r.Last.Run == nil || r.Last.Run.Memory != 15<<30 {
		t.Errorf("infra's run read on its Mac is %+v; want the 15 GB of the process that holds its guest", r)
	}
	if r = d.read(t.Context(), ReadIn{Of: "stopped", Run: true}); r.Last.Run != nil {
		t.Errorf("a VM no process holds read %+v; want no run", r.Last.Run)
	}
}

func TestAMacsSampleCarriesEachVMItRunsAsTheProcessOfItsGuest(t *testing.T) {
	d := Deps{
		Meter: processes{4242: {CPU: 18, Memory: 15 << 30}},
		Name:  "jterrazz-mac-mini", Kind: machine.Mac,
		Runs: func(context.Context) ([]activity.VMRun, error) {
			return []activity.VMRun{{VM: "infra", Holder: 4242}, {VM: "booting"}, {VM: "gone", Holder: 9}}, nil
		},
	}
	r, err := compose(t.Context(), d)
	if err != nil {
		t.Fatal(err)
	}
	if want := []activity.VMUsage{{VM: "infra", Usage: activity.Usage{CPU: 18, Memory: 15 << 30}}}; !slices.Equal(r.VMs, want) {
		t.Errorf("the Mac's reading carries VMs %+v; want only infra's, 18 %% of the Mac", r.VMs)
	}
	if len(r.Top.CPU) != 1 || r.Top.CPU[0].VM != "infra" {
		t.Errorf("the busiest by CPU are %+v; want infra's process named after it", r.Top.CPU)
	}
}

func TestTheDaysChartIsFoldedFromTheHistoryItsEngineKeeps(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	points := make([]activity.Sample, 0, 120)
	for i := range 120 {
		points = append(points, activity.Sample{At: now.Add(time.Duration(i-120) * time.Minute), CPUBusy: uint64(i * 300), CPUTotal: uint64(i * 1000)})
	}
	d := Deps{
		Meter: processes{}, Name: "jterrazz-mac-mini", Kind: machine.Mac, Now: func() time.Time { return now },
		Kept: func(context.Context, string) (activity.Kept, time.Time, bool, error) {
			return activity.Kept{Series: activity.SeriesOf(nil, points)}, now.Add(-3 * time.Hour), true, nil
		},
	}
	r := d.read(t.Context(), ReadIn{Graph: 24 * time.Hour})
	if r.Chart == nil || len(r.Chart.Buckets) != 48 || r.Chart.Buckets[47].Points != 30 || r.Chart.Buckets[47].Mean != 30 {
		t.Fatalf("the day's chart is %+v; want 48 bars, the last of 30 minutes at 30 %%", r.Chart)
	}
	if r.Chart.Buckets[0].Points != 0 {
		t.Errorf("the day's first bar is %+v; want a gap: the engine kept 2 h", r.Chart.Buckets[0])
	}
}
