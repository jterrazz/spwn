package calls_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/awake/calls"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract"
)

type clock struct{ At time.Time }

func (c *clock) Now() time.Time { return c.At }

type port struct {
	r     awake.Reading
	calls []string
}

func (p *port) Sleeping() bool { return false }

func (p *port) Capabilities() contract.PowerCaps {
	return contract.PowerCaps{Battery: p.r.HasBattery, Lid: p.r.Lid != "", Thermal: true}
}

func (p *port) Hold(name string) (func(), error) {
	p.calls = append(p.calls, "assert "+name)
	return func() { p.calls = append(p.calls, "release") }, nil
}

func (p *port) Clamshell(disabled bool) error {
	if disabled {
		p.calls = append(p.calls, "clamshell disabled")
	} else {
		p.calls = append(p.calls, "clamshell normal")
	}
	return nil
}

func (p *port) Read() (awake.Reading, error) { return p.r, nil }

func (p *port) Watch(context.Context) <-chan awake.Event { return nil }

func (p *port) DisplaySleepNow(context.Context) error {
	p.calls = append(p.calls, "display sleep")
	return nil
}

func (p *port) count(call string) int {
	n := 0
	for _, c := range p.calls {
		if c == call {
			n++
		}
	}
	return n
}

type world struct {
	t       *testing.T
	port    *port
	clock   *clock
	m       *calls.Keeper
	notices []awake.End
}

func laptop() awake.Reading {
	return awake.Reading{Lid: "open", Source: "ac", HasBattery: true, Percent: 80}
}

func newWorld(t *testing.T, r awake.Reading) *world {
	w := &world{t: t, port: &port{r: r}, clock: &clock{At: time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)}}
	w.m = calls.NewKeeper(w.port, w.clock, calls.KeeperOptions{Notice: func(e awake.End) { w.notices = append(w.notices, e) }})
	return w
}

func (w *world) awake(req awake.Request) {
	w.t.Helper()
	if req.By == "" {
		req.By = "station power awake"
	}
	if _, _, err := w.m.Awake(req); err != nil {
		w.t.Fatalf("awake: %v", err)
	}
}

func (w *world) pass(d time.Duration) {
	for step := time.Duration(0); step < d; step += time.Second {
		w.clock.At = w.clock.At.Add(time.Second)
		w.m.Tick()
	}
}

func (w *world) event(kind string) { w.m.Observe(context.Background(), awake.Event{Kind: kind}) }

func (w *world) ended(reason string) {
	w.t.Helper()
	v := w.m.View()
	if v.Session != nil {
		w.t.Fatalf("the session still holds; want it ended by %s", reason)
	}
	if v.LastEnd == nil || v.LastEnd.Reason != reason {
		w.t.Fatalf("last end = %+v, want reason %s", v.LastEnd, reason)
	}
	if last := w.port.calls[len(w.port.calls)-1]; last != "clamshell normal" {
		w.t.Fatalf("the last call is %q, want the clamshell reset to normal: %v", last, w.port.calls)
	}
	if w.port.count("release") != 1 {
		w.t.Fatalf("the assertion was released %d times: %v", w.port.count("release"), w.port.calls)
	}
}

func (w *world) held() {
	w.t.Helper()
	if !w.m.Held() {
		w.t.Fatalf("the session ended (%+v); want it held", w.m.View().LastEnd)
	}
}

func TestEachGuardPastItsThresholdRefusesTheStart(t *testing.T) {
	for name, c := range map[string]struct {
		r     func(*awake.Reading)
		req   awake.Request
		guard string
	}{
		"battery":   {func(r *awake.Reading) { r.Source, r.Percent = "battery", 18 }, awake.Request{}, awake.ReasonBattery},
		"low power": {func(r *awake.Reading) { r.LowPower = true }, awake.Request{}, awake.ReasonLowPower},
		"thermal":   {func(r *awake.Reading) { r.Thermal = 3 }, awake.Request{}, awake.ReasonThermal},
		"lid":       {func(r *awake.Reading) { r.Source, r.Lid = "battery", "closed" }, awake.Request{Lid: true}, awake.ReasonLid},
	} {
		t.Run(name, func(t *testing.T) {
			r := laptop()
			c.r(&r)
			w := newWorld(t, r)
			_, _, err := w.m.Awake(c.req)
			var g *awake.GuardError
			if !errors.As(err, &g) || g.Guard != c.guard {
				t.Fatalf("awake = %v, want the %s guard's refusal", err, c.guard)
			}
			if len(w.port.calls) != 0 {
				t.Fatalf("a refused start called the port: %v", w.port.calls)
			}
		})
	}
}

func TestBatteryLetsTheLidCloseOnBattery(t *testing.T) {
	r := laptop()
	r.Source, r.Lid = "battery", "closed"
	w := newWorld(t, r)
	w.awake(awake.Request{Lid: true, Battery: true})
	w.event("lid")
	w.held()
	if g := awake.GuardStates(w.m.View().Reading, w.m.View().Session); g[3].State != awake.GuardNear {
		t.Fatalf("the lid guard reads %+v, want near while allowed", g[3])
	}
}

func TestARefreshKeepsOneAssertionAndOnlyTheLidTouchesTheClamshell(t *testing.T) {
	w := newWorld(t, laptop())
	w.awake(awake.Request{})
	if w.port.count("clamshell disabled") != 0 {
		t.Fatalf("a session without --lid disabled the clamshell: %v", w.port.calls)
	}
	_, refreshed, err := w.m.Awake(awake.Request{Lid: true, For: time.Hour, By: "station power awake --lid --for 1h"})
	if err != nil || !refreshed {
		t.Fatalf("the second awake = %v, refreshed %v", err, refreshed)
	}
	if w.port.count("assert station: station power awake") != 1 || w.port.count("clamshell disabled") != 1 {
		t.Fatalf("a refresh took a second assertion or no clamshell: %v", w.port.calls)
	}
	if s := w.m.View().Session; !s.Lid || s.Until.Sub(w.clock.At) != time.Hour {
		t.Fatalf("the refreshed session is %+v", s)
	}
}

func TestEveryEndResetsTheClamshellAndRecordsWhy(t *testing.T) {
	for reason, end := range map[string]func(w *world){
		awake.ReasonUntil: func(w *world) { w.pass(time.Hour) },
		awake.ReasonSleep: func(w *world) {
			if s := w.m.Sleep(); !s.Ended || s.After != 0 {
				t.Fatalf("sleep = %+v", s)
			}
		},
		awake.ReasonStop: func(w *world) { w.m.Stop() },
		awake.ReasonQuit: func(w *world) { w.m.Quit() },
		awake.ReasonBattery: func(w *world) {
			w.port.r.Source, w.port.r.Percent = "battery", 20
			w.event("source")
		},
		awake.ReasonLowPower: func(w *world) {
			w.port.r.LowPower = true
			w.pass(29 * time.Second)
			w.held()
			w.pass(time.Second)
		},
		awake.ReasonThermal: func(w *world) {
			w.port.r.Thermal = 3
			w.event("thermal")
		},
	} {
		t.Run(reason, func(t *testing.T) {
			w := newWorld(t, laptop())
			w.awake(awake.Request{Lid: true, Battery: true, For: time.Hour})
			end(w)
			w.ended(reason)
			if notified := len(w.notices) == 1; notified == (reason == awake.ReasonSleep) {
				t.Fatalf("notices %v after %s: power sleep alone ends silently", w.notices, reason)
			}
		})
	}
}

func TestHeavyThermalEndsOnlyAfterFiveContinuousMinutesLidClosedOnBattery(t *testing.T) {
	r := laptop()
	r.Lid, r.Source, r.Thermal = "closed", "battery", 2
	w := newWorld(t, r)
	w.awake(awake.Request{Lid: true, Battery: true})
	w.event("thermal")
	w.pass(4*time.Minute + 59*time.Second)
	w.held()
	w.port.r.Lid = "open"
	w.event("lid")
	w.port.r.Lid = "closed"
	w.event("lid")
	w.pass(4*time.Minute + 59*time.Second)
	w.held()
	w.pass(31 * time.Second)
	w.ended(awake.ReasonThermal)
}

func TestHeavyThermalOnACNeverEnds(t *testing.T) {
	r := laptop()
	r.Lid, r.Thermal = "closed", 2
	w := newWorld(t, r)
	w.awake(awake.Request{Lid: true, For: time.Hour})
	w.pass(30 * time.Minute)
	w.held()
}

func TestTheLidGuardTripsInEitherOrder(t *testing.T) {
	for name, order := range map[string][2]func(*awake.Reading){
		"lid then battery": {func(r *awake.Reading) { r.Lid = "closed" }, func(r *awake.Reading) { r.Source = "battery" }},
		"battery then lid": {func(r *awake.Reading) { r.Source = "battery" }, func(r *awake.Reading) { r.Lid = "closed" }},
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t, laptop())
			w.awake(awake.Request{Lid: true})
			order[0](&w.port.r)
			w.event("lid")
			w.held()
			order[1](&w.port.r)
			w.event("source")
			w.ended(awake.ReasonLid)
		})
	}
}

func TestTheMaskIsSetAgainOnEverySourceAndLidChangeWhileTheLidHolds(t *testing.T) {
	w := newWorld(t, laptop())
	w.awake(awake.Request{Lid: true, Battery: true})
	for _, kind := range []string{"source", "lid", "thermal", "lid"} {
		w.event(kind)
	}
	if n := w.port.count("clamshell disabled"); n != 4 {
		t.Fatalf("the clamshell was disabled %d times, want 1 at the start and 3 again: %v", n, w.port.calls)
	}
}

func TestTheDisplaySleepsOnLidCloseUnderTheLidOnly(t *testing.T) {
	for name, lid := range map[string]bool{"--lid": true, "without --lid": false} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t, laptop())
			w.awake(awake.Request{Lid: lid})
			w.port.r.Lid = "closed"
			w.event("lid")
			if slept := w.port.count("display sleep") == 1; slept != lid {
				t.Fatalf("the display slept = %v, want %v: %v", slept, lid, w.port.calls)
			}
		})
	}
}

func TestAHeldSessionLogsItsHeartbeatEvery30Seconds(t *testing.T) {
	w := newWorld(t, laptop())
	var beats []string
	w.m = calls.NewKeeper(w.port, w.clock, calls.KeeperOptions{Log: func(line string) {
		if strings.HasPrefix(line, "held ") {
			beats = append(beats, line)
		}
	}})
	w.awake(awake.Request{})
	w.pass(65 * time.Second)
	want := []string{"held lid=open source=ac battery=80% thermal=0", "held lid=open source=ac battery=80% thermal=0"}
	if !slices.Equal(beats, want) {
		t.Fatalf("heartbeats over 65s = %q, want %q", beats, want)
	}
	w.m.Sleep()
	w.pass(time.Minute)
	if len(beats) != 2 {
		t.Fatalf("a heartbeat was logged with no session held: %q", beats)
	}
}

// invariant: a keep-awake asked until an hour ends at that hour, however
// long after the page that offered it was read: Until 17:00 holds until
// 17:00, never 17:01.
func TestAKeepAwakeUntilAnHourEndsAtThatHour(t *testing.T) {
	w := newWorld(t, laptop())
	at := time.Date(2026, 11, 3, 17, 0, 0, 0, time.UTC)
	w.clock.At = time.Date(2026, 11, 3, 10, 0, 47, 0, time.UTC)
	w.awake(awake.Request{For: 7 * time.Hour, Until: at})
	if got := w.m.View().Session.Until; !got.Equal(at) {
		t.Fatalf("until %s, want %s", got, at)
	}
	if _, _, err := w.m.Awake(awake.Request{Until: w.clock.At.Add(-time.Minute)}); err == nil {
		t.Fatalf("an hour already past is taken")
	}
}

// invariant: a switch flipped on a held session keeps its end and the term
// it was asked for: 4 h asked, then Lid closed, is still the 4 h one.
func TestASwitchKeepsTheTermTheSessionWasAskedFor(t *testing.T) {
	w := newWorld(t, laptop())
	w.awake(awake.Request{For: 4 * time.Hour})
	end := w.m.View().Session.Until
	w.clock.At = w.clock.At.Add(time.Minute)
	if _, _, err := w.m.Awake(awake.Request{Lid: true, Until: end}); err != nil {
		t.Fatalf("Lid closed on the held session = %v", err)
	}
	if s := w.m.View().Session; !s.Until.Equal(end) || s.For != 4*time.Hour || !s.Lid {
		t.Fatalf("after Lid closed the session is %+v, want until %s, asked 4 h, lid", s, end)
	}
}
