package automation

import (
	"sync"
	"testing"
	"time"
)

const testEpoch = "2026-05-01T00:00:00Z"

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	got, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %s: %v", s, err)
	}
	return got
}

// ── FakeClock — basic Now/Advance ───────────────────────────────────

func TestFakeClock_NowReflectsAdvance(t *testing.T) {
	c := NewFakeClock(mustParse(t, testEpoch))
	if got := c.Now(); !got.Equal(mustParse(t, testEpoch)) {
		t.Errorf("initial Now = %s, want %s", got, testEpoch)
	}
	c.Advance(1 * time.Hour)
	if got := c.Now(); !got.Equal(mustParse(t, "2026-05-01T01:00:00Z")) {
		t.Errorf("after Advance Now = %s, want 01:00", got)
	}
}

func TestFakeClock_AdvanceTo(t *testing.T) {
	c := NewFakeClock(mustParse(t, testEpoch))
	c.AdvanceTo(mustParse(t, "2026-05-02T06:00:00Z"))
	if got := c.Now(); !got.Equal(mustParse(t, "2026-05-02T06:00:00Z")) {
		t.Errorf("Now = %s", got)
	}
}

func TestFakeClock_AdvanceToBackwardsPanics(t *testing.T) {
	c := NewFakeClock(mustParse(t, "2026-05-02T00:00:00Z"))
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on backwards AdvanceTo")
		}
	}()
	c.AdvanceTo(mustParse(t, "2026-05-01T00:00:00Z"))
}

// ── FakeClock — After fires deterministically ───────────────────────

func TestFakeClock_AfterFiresOnAdvance(t *testing.T) {
	c := NewFakeClock(mustParse(t, testEpoch))
	ch := c.After(1 * time.Hour)

	// Advancing less than the deadline must NOT fire.
	c.Advance(30 * time.Minute)
	select {
	case got := <-ch:
		t.Fatalf("After fired early at %s", got)
	default:
	}

	// Advancing past the deadline fires.
	c.Advance(31 * time.Minute)
	select {
	case got := <-ch:
		want := mustParse(t, "2026-05-01T01:00:00Z")
		if !got.Equal(want) {
			t.Errorf("After value = %s, want %s", got, want)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("After did not fire after deadline crossed")
	}
}

func TestFakeClock_AfterZeroDurationFiresImmediately(t *testing.T) {
	// time.After(0) fires immediately; FakeClock matches that so
	// engine code that computes a negative-or-zero next-tick (e.g.
	// the "we missed it" case) doesn't deadlock.
	c := NewFakeClock(mustParse(t, testEpoch))
	ch := c.After(0)
	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		t.Fatal("After(0) did not fire")
	}
}

func TestFakeClock_AfterMultipleTimers_FireInDeadlineOrder(t *testing.T) {
	c := NewFakeClock(mustParse(t, testEpoch))
	ch3h := c.After(3 * time.Hour)
	ch1h := c.After(1 * time.Hour)
	ch2h := c.After(2 * time.Hour)

	// Advance past all three. Receive in the order they should fire.
	c.Advance(4 * time.Hour)

	want := []time.Duration{1 * time.Hour, 2 * time.Hour, 3 * time.Hour}
	chans := []<-chan time.Time{ch1h, ch2h, ch3h}
	for i, ch := range chans {
		select {
		case got := <-ch:
			expected := mustParse(t, testEpoch).Add(want[i])
			if !got.Equal(expected) {
				t.Errorf("timer #%d fired at %s, want %s", i, got, expected)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timer #%d did not fire", i)
		}
	}
}

func TestFakeClock_PendingCountAccurate(t *testing.T) {
	c := NewFakeClock(mustParse(t, testEpoch))
	if got := c.Pending(); got != 0 {
		t.Errorf("initial Pending = %d, want 0", got)
	}
	_ = c.After(1 * time.Hour)
	_ = c.After(2 * time.Hour)
	if got := c.Pending(); got != 2 {
		t.Errorf("after 2 After(): Pending = %d, want 2", got)
	}
	c.Advance(1 * time.Hour)
	if got := c.Pending(); got != 1 {
		t.Errorf("after firing one: Pending = %d, want 1", got)
	}
}

// ── FakeClock — concurrency safety ──────────────────────────────────

func TestFakeClock_ConcurrentSafe(t *testing.T) {
	// Stress test: many goroutines registering After() while another
	// goroutine advances. We don't assert specific firings — only
	// that the package never deadlocks or races (run with -race).
	c := NewFakeClock(mustParse(t, testEpoch))
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(d time.Duration) {
			defer wg.Done()
			ch := c.After(d)
			select {
			case <-ch:
			case <-time.After(2 * time.Second):
				t.Errorf("timer at d=%s never fired", d)
			}
		}(time.Duration(i+1) * time.Millisecond)
	}

	// Give registrations a moment, then sweep.
	time.Sleep(20 * time.Millisecond)
	c.Advance(1 * time.Second)
	wg.Wait()
}

// ── RealClock — sanity check ────────────────────────────────────────

func TestRealClock_NowMatchesSystem(t *testing.T) {
	r := RealClock{}
	before := time.Now()
	got := r.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Errorf("RealClock.Now %s outside [%s, %s]", got, before, after)
	}
}

func TestRealClock_AfterFires(t *testing.T) {
	r := RealClock{}
	ch := r.After(10 * time.Millisecond)
	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		t.Fatal("RealClock.After(10ms) did not fire within 1s")
	}
}
