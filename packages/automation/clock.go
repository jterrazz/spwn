package automation

import (
	"sort"
	"sync"
	"time"
)

// Clock is the engine's view of wall time. Production uses
// RealClock (which reads time.Now / time.After); tests inject a
// FakeClock to drive cron evaluation deterministically.
//
// The interface is small on purpose: the engine only needs Now and
// After. Everything else (Sleep, AfterFunc, etc.) can be expressed
// in terms of those two.
type Clock interface {
	// Now returns the current wall time.
	Now() time.Time
	// After returns a channel that fires once at d duration from
	// the call. With RealClock this delegates to time.After. With
	// FakeClock the channel fires when Advance crosses the deadline.
	After(d time.Duration) <-chan time.Time
}

// RealClock is the production Clock: time.Now + time.After.
type RealClock struct{}

// Now returns the current time per the system clock.
func (RealClock) Now() time.Time { return time.Now() }

// After delegates to time.After.
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// FakeClock is a manually-driven Clock for tests. Time only advances
// when Advance is called — Now never moves on its own. Pending
// After channels fire (with the synthetic deadline as the value)
// the moment the fake time crosses their deadline.
//
// Safe for concurrent use: multiple goroutines may call Now / After /
// Advance from any goroutine.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	pending []*pendingTimer
}

type pendingTimer struct {
	deadline time.Time
	ch       chan time.Time
	done     bool // set when fired so a re-Advance doesn't double-send
}

// NewFakeClock constructs a FakeClock anchored at the given time.
// Convention: tests parse a fixed RFC3339 string so failures print a
// stable, human-readable timestamp.
func NewFakeClock(t time.Time) *FakeClock {
	return &FakeClock{now: t}
}

// Now returns the fake clock's current time. Does not advance.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// After registers a synthetic timer firing at Now+d. Returns a buffered
// channel of size 1 — Advance pushes a value when the deadline elapses,
// then closes the channel so receivers don't block on extra reads.
func (c *FakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &pendingTimer{
		deadline: c.now.Add(d),
		ch:       make(chan time.Time, 1),
	}
	c.pending = append(c.pending, t)
	if d <= 0 {
		// Zero-or-negative duration: fire immediately, exactly as
		// time.After does.
		t.ch <- t.deadline
		close(t.ch)
		t.done = true
	}
	return t.ch
}

// Advance moves the fake clock forward by d and fires every pending
// timer whose deadline is at or before the new time. Timers fire in
// deadline order so consumers see causally consistent values when
// they read sequentially.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	now := c.now
	due := make([]*pendingTimer, 0, len(c.pending))
	keep := make([]*pendingTimer, 0, len(c.pending))
	for _, t := range c.pending {
		if t.done {
			continue
		}
		if !t.deadline.After(now) {
			due = append(due, t)
		} else {
			keep = append(keep, t)
		}
	}
	c.pending = keep
	c.mu.Unlock()
	// Fire in deadline order. Sort outside the lock to avoid holding
	// it while channels send (the receiver may run synchronously).
	sort.Slice(due, func(i, j int) bool { return due[i].deadline.Before(due[j].deadline) })
	for _, t := range due {
		t.ch <- t.deadline
		close(t.ch)
		t.done = true
	}
}

// AdvanceTo moves the fake clock to the absolute time t. Panics if
// t is before the current time — moving backwards in tests is almost
// always a bug.
func (c *FakeClock) AdvanceTo(t time.Time) {
	c.mu.Lock()
	cur := c.now
	c.mu.Unlock()
	if t.Before(cur) {
		panic("FakeClock.AdvanceTo: target is before current time")
	}
	c.Advance(t.Sub(cur))
}

// Pending reports the number of timers waiting to fire. Test helper
// only — useful when asserting "no goroutine has registered a timer
// yet" or "every loop iteration registered exactly one new wait".
func (c *FakeClock) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, t := range c.pending {
		if !t.done {
			n++
		}
	}
	return n
}
