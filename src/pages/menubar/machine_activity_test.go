package menubar

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
)

// value: a day of minutes and 6 min of 2 s points at a steady CPU share,
// the share `at` gives where it says one.
func kept(steady float64, at func(time.Time) (float64, bool)) *activity.Report {
	series := func(span, every time.Duration) []activity.Sample {
		s := activity.Sample{At: macNow.Add(-span), CPUTotal: 1_000_000}
		out := []activity.Sample{s}
		for i := 1; i <= int(span/every); i++ {
			when := macNow.Add(-span + time.Duration(i)*every)
			share := steady
			if v, ok := at(when); ok {
				share = v
			}
			s = activity.Sample{At: when, CPUTotal: s.CPUTotal + uint64(every.Seconds()*1000), CPUBusy: s.CPUBusy + uint64(share*every.Seconds()*10),
				NetIn: s.NetIn + uint64(every.Seconds()*400_000), NetOut: s.NetOut + uint64(every.Seconds()*200_000)}
			out = append(out, s)
		}
		return out
	}
	return &activity.Report{History: series(24*time.Hour, activity.CoarseEvery), Graph: series(6*time.Minute, activity.FineEvery)}
}

func chartsOf(t *testing.T, s *page.Section) (cpu, net page.Chart) {
	t.Helper()
	var charts []page.Chart
	for _, it := range s.Items {
		if it.Chart != nil {
			charts = append(charts, *it.Chart)
		}
	}
	if len(charts) != 2 {
		t.Fatalf("section %s charts %d, want its CPU then its network", s.ID, len(charts))
	}
	return charts[0], charts[1]
}

// invariant: SPEC A4.1, B3: a machine's Activity carries its three windows
// at their points, each with the picker that opens on the hour, so the
// window picked is the app's own and asks the engine nothing.
func TestActivityCarriesItsThreeWindowsSoPickingOneAsksNothing(t *testing.T) {
	p := macPageOf(t, miniMac, func(m *Machine) { m.Activity = kept(40, func(time.Time) (float64, bool) { return 0, false }) })
	want := []struct {
		flag  string
		kind  page.ChartKind
		n     int
		start string
	}{{"6m", page.ChartLine, 72, "6 min ago"}, {"1h", page.ChartBars, 60, "1 hr ago"}, {"24h", page.ChartBars, 48, "yesterday 10:30"}}
	for _, w := range want {
		s := sectionOf(p, SectionWindow+w.flag)
		if s == nil || s.Segment != SegmentActivity {
			t.Fatalf("no window %s in the Activity segment", w.flag)
		}
		var picked []string
		for _, a := range s.Picker {
			if a.Picked {
				picked = append(picked, a.ID)
			}
		}
		if len(s.Picker) != 3 || !slices.Equal(picked, []string{ActionWindow + WindowDefault}) {
			t.Fatalf("window %s picker %+v, want 6 min, 1 hr, 24 hr, the hour picked", w.flag, s.Picker)
		}
		cpu, net := chartsOf(t, s)
		if cpu.Kind != w.kind || len(cpu.Series) != 1 || len(cpu.Series[0].Points) != w.n || !slices.Equal(cpu.Ends, []string{w.start, "now"}) {
			t.Fatalf("window %s CPU %s of %d points ends %q, want %s of %d from %q", w.flag, cpu.Kind, len(cpu.Series[0].Points), cpu.Ends, w.kind, w.n, w.start)
		}
		if net.Kind != page.ChartMirror || len(net.Series) != 2 || net.Series[0].Key != "sent" || net.Series[1].Points[w.n-1] != 0.4 {
			t.Fatalf("window %s network %+v, want sent over received, 0.4 MB/s received now", w.flag, net)
		}
		if len(cpu.At) != w.n || !slices.Equal(net.At, cpu.At) {
			t.Fatalf("window %s says %d times, the network %d, want one per point of both", w.flag, len(cpu.At), len(net.At))
		}
	}
}

// invariant: SPEC A4.1: a scrubbed point says when it was, as its window
// says time: how long ago within the hour, the time of day over the day.
func TestAScrubbedPointSaysWhenItWas(t *testing.T) {
	p := macPageOf(t, miniMac, func(m *Machine) { m.Activity = kept(40, func(time.Time) (float64, bool) { return 0, false }) })
	want := []struct {
		flag  string
		index int
		at    string
	}{{"6m", 71, "5 s ago"}, {"6m", 41, "2 min 35 s ago"}, {"6m", 60, "1 min ago"}, {"1h", 36, "24 min ago"}, {"24h", 0, "10:30"}, {"24h", 47, "10:00"}}
	for _, w := range want {
		cpu, _ := chartsOf(t, sectionOf(p, SectionWindow+w.flag))
		if got := cpu.At[w.index]; got != w.at {
			t.Errorf("window %s point %d says %q, want %q", w.flag, w.index, got, w.at)
		}
	}
}

// invariant: SPEC A4.1: an anomaly is the whole run at or above 90 %
// around the peak, on both of its sides, 90 % itself included.
func TestAnAnomalyIsTheWholeRunAroundItsPeak(t *testing.T) {
	p := macPageOf(t, miniMac, func(m *Machine) {
		m.Activity = kept(30, func(at time.Time) (float64, bool) {
			switch {
			case at.Equal(macNow.Add(-35 * time.Minute)):
				return 95, true
			case !at.Before(macNow.Add(-40*time.Minute)) && at.Before(macNow.Add(-30*time.Minute)):
				return 90, true
			}
			return 0, false
		})
	})
	cpu, _ := chartsOf(t, sectionOf(p, SectionWindow+"1h"))
	if cpu.Peak == nil || cpu.Peak.Index != 25 {
		t.Fatalf("peak %+v, want the bar 35 min ago", cpu.Peak)
	}
	if a := cpu.Anomaly; a == nil || a.From != 20 || a.To != 29 || a.Sentence != "Peaked 95% at 09:55" {
		t.Fatalf("anomaly %+v, want the ten bars at 90 %% and over, 40 to 30 min ago, Peaked 95%% at 09:55", a)
	}
}

// invariant: SPEC A4.1: a window whose CPU stayed above 90 % shades the
// bars it spans and says it: when it peaked once it is over, how long it
// has lasted while it still holds.
func TestAHighCPUIsShadedAndSaidOnItsWindow(t *testing.T) {
	tests := time.Date(2026, 11, 7, 3, 0, 0, 0, time.UTC)
	p := macPageOf(t, miniMac, func(m *Machine) {
		m.Activity = kept(30, func(at time.Time) (float64, bool) {
			if !at.Before(tests) && at.Before(tests.Add(30*time.Minute)) {
				return 94, true
			}
			return 0, false
		})
	})
	cpu, _ := chartsOf(t, sectionOf(p, SectionWindow+"24h"))
	if cpu.Peak == nil || cpu.Peak.Value.N != 940 || cpu.Peak.At != "03:00" {
		t.Fatalf("peak %+v, want 94 %% at 03:00", cpu.Peak)
	}
	if a := cpu.Anomaly; a == nil || a.From != cpu.Peak.Index || a.To != cpu.Peak.Index || a.Sentence != "Peaked 94% at 03:00" {
		t.Fatalf("anomaly %+v, want the 03:00 bar shaded, Peaked 94%% at 03:00", a)
	}
	calm, _ := chartsOf(t, sectionOf(p, SectionWindow+"1h"))
	if calm.Anomaly != nil {
		t.Fatalf("the hour's anomaly %+v, want none: it stayed at 30 %%", calm.Anomaly)
	}

	p = macPageOf(t, miniMac, func(m *Machine) {
		m.Activity = kept(30, func(at time.Time) (float64, bool) {
			if at.After(macNow.Add(-13 * time.Minute)) {
				return 96, true
			}
			return 0, false
		})
	})
	cpu, _ = chartsOf(t, sectionOf(p, SectionWindow+"1h"))
	if a := cpu.Anomaly; a == nil || a.To != 59 || a.Sentence != "CPU on mac-mini has been above 90 % for 12 min" {
		t.Fatalf("anomaly %+v, want the last 12 bars, still above 90 %%", a)
	}
}

// invariant: SPEC A4.1: a Mac's busiest processes are its share of the
// CPU, each VM it runs among them by its name, opening the VM's page.
func TestAMacsBusiestProcessesHoldItsVMsWhichOpenTheirPages(t *testing.T) {
	p := macPageOf(t, miniMac, func(m *Machine) {
		m.Samples = []activity.Reading{{Sample: activity.Sample{At: macNow, MemTotal: 64 << 30, MemUsed: 50 << 30},
			Top: activity.Busiest{CPU: []activity.Process{{PID: 9, Name: "Virtualization", Value: 21, VM: "linux-3"}, {PID: 7, Name: "mds_stores", Value: 2}}},
			VMs: []activity.VMUsage{{VM: "linux-3", Usage: activity.Usage{CPU: 21}}, {VM: "infra", Usage: activity.Usage{CPU: 9}}}}}
	})
	s := sectionOf(p, SectionProcesses)
	if got := labels(s.Items); !slices.Equal(got, []string{"linux-3", "infra", "mds_stores"}) {
		t.Fatalf("processes %q, want linux-3 once, by its name, then infra, then mds_stores", got)
	}
	if r := s.Items[0].Row; r.Nav != machinePage("linux-3") || r.Glyph.Kind != "agent" || r.Cells[0].Live != "cpuh:linux-3" {
		t.Fatalf("linux-3 %+v, want the agent mark, its share of the Mac live, opening its page", r)
	}
	memory := sectionOf(p, SectionMemory).Items[0].Gauge
	parts := memory.Bar.Items
	if memory.Value.Text != "14 GB free" || parts[0].Label != "macOS" || parts[0].Text != "14 GB" || parts[1].Label != "VMs" || parts[1].Text != "36 GB" || len(parts[1].Items) != 3 {
		t.Fatalf("memory %+v, want 14 GB free, macOS 14 GB, the three VMs' 36 GB", memory)
	}
}

// invariant: SPEC A4: a running VM's page has its own header and segments,
// a server's Overview and Activity, a workspace's four; its Activity says
// where its figures come from, and a workspace's upkeep there runs Update
// inside it.
func TestAVMsActivityIsMeasuredInsideItAndUpdatesItsGuest(t *testing.T) {
	server := macPageOf(t, "infra", func(m *Machine) {
		m.Summary = &overview.Summary{Head: overview.Head{Name: "infra"}, Needs: []overview.Need{{Machine: "infra", Section: "apps", Subject: "updates", Text: "2 updates", Call: CallUpdate}}}
	})
	if server.Header == nil || server.Header.Back != "mac-mini" || server.Header.Glyph.Kind != "server" || len(server.Segments) != 2 {
		t.Fatalf("infra header %+v segments %+v, want back to the mini, the server glyph, Overview and Activity", server.Header, server.Segments)
	}
	if note := sectionOf(server, SectionMeasured); note == nil || note.Items[0].Note.Text != "Measured inside the guest. If infra stops answering, this shows what mac-mini sees of it." {
		t.Fatalf("infra's footnote %+v", note)
	}
	if sectionOf(server, SectionGuestUpkeep) != nil {
		t.Fatal("a server keeps no upkeep of its own here")
	}
	for _, s := range server.Sections {
		if s.Segment == "" && s.ID != SectionMachine && s.ID != SectionSummary {
			t.Errorf("section %s has no segment: every segment would draw it", s.ID)
		}
	}

	ws := macPageOf(t, "linux-3", func(m *Machine) {
		m.Summary = &overview.Summary{Head: overview.Head{Name: "linux-3"}, Needs: []overview.Need{{Machine: "linux-3", Section: "apps", Subject: "updates", Text: "2 app updates", Call: CallUpdate}}}
	})
	if len(ws.Segments) != 4 || ws.Header.Glyph.Kind != "agent" {
		t.Fatalf("linux-3 segments %+v glyph %+v, want four segments under the agent mark", ws.Segments, ws.Header.Glyph)
	}
	upkeep := sectionOf(ws, SectionGuestUpkeep)
	if upkeep == nil || upkeep.Items[0].Row.Label != "2 app updates" || upkeep.Items[0].Row.Verb != ActionUpdateGuest {
		t.Fatalf("upkeep %+v, want 2 app updates with Update", upkeep)
	}
	i := slices.IndexFunc(ws.Actions, func(a page.Action) bool { return a.ID == ActionUpdateGuest })
	if i < 0 || ws.Actions[i].Run.Call != CallUpdate || *ws.Actions[i].Run.Target != (page.Target{Name: "linux-3", On: miniMac}) {
		t.Fatalf("Update %+v, want apps.update inside linux-3 on the mini", ws.Actions)
	}

	stopped := macPageOf(t, "linux-3", func(m *Machine) { m.All.Macs[1].VMs[0].State = model.Stopped })
	if stopped.Header != nil || len(stopped.Segments) != 0 {
		t.Fatalf("a stopped VM's header %+v segments %+v, want none", stopped.Header, stopped.Segments)
	}
}
