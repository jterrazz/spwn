package world_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
)

const withPast = `{"now": "2026-11-07T10:30:00Z", "this_mac": "mbp", "macs": {
	"mbp": {
		"engine": {"started_at": "2026-11-04T10:30:00Z", "past": {
			"mbp": {"base": 22, "swing": 4, "seed": 3, "in": 0.5, "out": 0.45},
			"vm1": {"base": 34, "swing": 6, "seed": 5, "in": 0.45, "out": 0.3},
			"gone": {"base": 4, "swing": 1, "seed": 8}}},
		"meter": {
			"sample": {"at": "2026-11-07T10:29:50Z", "cpu_busy": 300000000, "cpu_total": 1000000000, "cpus": 16, "mem_used": 32000000000, "mem_total": 48000000000, "net_in": 1000000000000, "net_out": 1000000000000, "boot": "2026-11-04T10:30:00Z"},
			"per_second": {"cpu_busy": 300, "cpu_total": 1000, "net_in": 500000, "net_out": 450000},
			"top": {"cpu": [{"pid": 501, "name": "Xcode", "value": 4}, {"pid": 1001, "name": "Virtualization", "value": 9}], "memory": []},
			"processes": {"1001": {"cpu": 9, "memory": 5000000000}}},
		"vms": {"vm1": {"disk": true, "pid": 1001}, "api": {"disk": true}}},
	"vm1": {"meter": {"sample": {"at": "2026-11-07T10:30:00Z", "cpu_busy": 1000000, "cpu_total": 3000000, "cpus": 4, "boot": "2026-11-07T08:30:00Z"}}}}}`

func pastOf(t *testing.T, doc, machine string) activity.Kept {
	t.Helper()
	k, ok := seeded(t, doc).Macs["mbp"].Engine.Activity[machine]
	if !ok {
		t.Fatalf("the engine keeps no activity of %s: its past was not unfolded", machine)
	}
	return k
}

func TestAnEnginesPastEndsOnTheSampleItsMeterAnswersNow(t *testing.T) {
	w := seeded(t, withPast)
	k, ok := w.Macs["mbp"].Engine.Activity["mbp"]
	if !ok {
		t.Fatal("the engine keeps no activity of mbp: its past was not unfolded")
	}
	now := w.Macs["mbp"].Meter.Answers(w.Now)
	for ring, points := range map[string][]activity.Sample{"coarse": k.Series.Coarse(), "fine": k.Series.Fine()} {
		newest := points[len(points)-1]
		if newest != now {
			t.Errorf("the %s ring ends on %+v, want the meter's answer now %+v", ring, newest, now)
		}
	}
	if k.Last.Sample != now {
		t.Errorf("the last reading is %+v, want the meter's answer now", k.Last.Sample)
	}
	next := w.Macs["mbp"].Meter.Answers(w.Now.Add(activity.FineEvery))
	if r := activity.Between(k.Series.Fine()[len(k.Series.Fine())-1], next); r.CPU <= 0 || r.CPU > 100 || r.NetIn <= 0 {
		t.Errorf("the first live sample after the past reads %+v, want the meter's own rates", r)
	}
}

func TestAnEnginesPastIsADayAtAMinuteAndSixMinutesAtTwoSeconds(t *testing.T) {
	k := pastOf(t, withPast, "mbp")
	coarse, fine := k.Series.Coarse(), k.Series.Fine()
	if len(coarse) != activity.CoarsePoints || len(fine) != activity.FinePoints {
		t.Fatalf("the past holds %d coarse and %d fine points, want %d and %d", len(coarse), len(fine), activity.CoarsePoints, activity.FinePoints)
	}
	for ring, points := range map[string][]activity.Sample{"coarse": coarse, "fine": fine} {
		for i := 1; i < len(points); i++ {
			a, b := points[i-1], points[i]
			if b.CPUBusy < a.CPUBusy || b.CPUTotal <= a.CPUTotal || b.NetIn < a.NetIn || b.NetOut < a.NetOut {
				t.Fatalf("the %s ring's counters go back at %v: %+v then %+v", ring, b.At, a, b)
			}
		}
	}
	if peak := activity.Fold(k.Series.Graph(24*time.Hour, coarse[len(coarse)-1].At), 24*time.Hour, coarse[len(coarse)-1].At); peak.Peak < 0 || peak.PeakAt.IsZero() {
		t.Errorf("a day of the past charts no peak: %+v", peak)
	}
}

func TestAMacsPastReadsTheVMsItRunsFromTheirProcesses(t *testing.T) {
	k := pastOf(t, withPast, "mbp")
	if len(k.Last.VMs) != 1 || k.Last.VMs[0].VM != "vm1" || k.Last.VMs[0].CPU != 9 {
		t.Errorf("the Mac's last reading holds the VMs %+v, want vm1 at 9 %%", k.Last.VMs)
	}
	if top := k.Last.Top.CPU; len(top) != 2 || top[0].PID != 1001 || top[0].VM != "vm1" {
		t.Errorf("the Mac's busiest are %+v, want vm1's process first, named after it", top)
	}
}

func TestAPastNeverStartsBeforeTheMachineCounted(t *testing.T) {
	k := pastOf(t, withPast, "vm1")
	coarse := k.Series.Coarse()
	boot := time.Date(2026, 11, 7, 8, 30, 0, 0, time.UTC)
	if coarse[0].At.Before(boot) {
		t.Errorf("the VM's past starts at %v, before it booted at %v", coarse[0].At, boot)
	}
	if len(coarse) >= 120 {
		t.Errorf("the VM's past holds %d minutes; its counters hold less than its two hours since boot", len(coarse))
	}
	for i := 1; i < len(coarse); i++ {
		if coarse[i].CPUTotal <= coarse[i-1].CPUTotal {
			t.Fatalf("the VM's counters wrap at %v: %d then %d", coarse[i].At, coarse[i-1].CPUTotal, coarse[i].CPUTotal)
		}
	}
	if newest := coarse[len(coarse)-1]; newest.CPUTotal != 3_000_000 {
		t.Errorf("the VM's past ends at %d ticks, want its meter's 3000000", newest.CPUTotal)
	}
}

func TestAPastIsSpentAndNeverReplacesWhatTheEngineKeeps(t *testing.T) {
	doc := strings.Replace(withPast, `"started_at": "2026-11-04T10:30:00Z",`, `"started_at": "2026-11-04T10:30:00Z", "activity": {"mbp": {"last": {"sample": {"at": "2026-11-07T10:29:00Z", "cpu_total": 7}}, "series": {"fine": [], "coarse": []}}},`, 1)
	w := seeded(t, doc)
	e := w.Macs["mbp"].Engine
	if got := e.Activity["mbp"].Last.Sample.CPUTotal; got != 7 {
		t.Errorf("the activity the engine kept was replaced by its past: %d ticks", got)
	}
	if _, ok := e.Activity["gone"]; ok {
		t.Error("a machine with no meter got a past")
	}
	data, err := w.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"past"`) {
		t.Error("the world writes its past back after it was unfolded")
	}
	again, written := seeded(t, string(data)).Macs["mbp"].Engine.Activity["vm1"], e.Activity["vm1"]
	if len(again.Series.Coarse()) != len(written.Series.Coarse()) {
		t.Error("the past unfolded reads back other than it was written")
	}
}
