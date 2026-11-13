package lanes

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type manualClock struct {
	mu      sync.Mutex
	now     time.Duration
	waiters []waiter
}

type waiter struct {
	at   time.Duration
	wake chan struct{}
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Unix(0, 0).Add(c.now)
}

func (c *manualClock) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	w := waiter{at: c.now + d, wake: make(chan struct{})}
	c.waiters = append(c.waiters, w)
	c.mu.Unlock()
	select {
	case <-w.wake:
		return nil
	case <-ctx.Done():
		c.mu.Lock()
		defer c.mu.Unlock()
		c.waiters = slices.DeleteFunc(c.waiters, func(o waiter) bool { return o.wake == w.wake })
		return ctx.Err()
	}
}

func (c *manualClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now += d
	kept := c.waiters[:0]
	for _, w := range c.waiters {
		if w.at <= c.now {
			close(w.wake)
		} else {
			kept = append(kept, w)
		}
	}
	c.waiters = kept
}

func (c *manualClock) sleepers() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.waiters)
}

func eventually(t *testing.T, ok func() bool) {
	t.Helper()
	for i := 0; !ok(); i++ {
		if i > 10_000_000 {
			t.Fatal("the condition never held")
		}
		runtime.Gosched()
	}
}

type report struct{ parts []string }

func clone(r report) report { return report{parts: slices.Clone(r.parts)} }

func set(i int, v string) func(*report) { return func(r *report) { r.parts[i] = v } }

func TestTheFrameComesFirstAndTheLastValueIsComplete(t *testing.T) {
	p := NewPool(context.Background(), &manualClock{}, 4, 1)
	release := make(chan struct{})
	jobs := []Job[report]{
		{Key: "a", Do: func(context.Context) func(*report) { return set(0, "fast") }},
		{Key: "b", Lane: Inline, Do: func(context.Context) func(*report) { <-release; return set(1, "slow") }},
	}
	ch := Run(context.Background(), p, report{parts: []string{"-", "-"}}, jobs, clone)
	if first := <-ch; !slices.Equal(first.parts, []string{"-", "-"}) {
		t.Fatalf("the first value is %v, want the frame", first.parts)
	}
	close(release)
	if last := Last(ch); !slices.Equal(last.parts, []string{"fast", "slow"}) {
		t.Fatalf("the last value is %v, want every answer", last.parts)
	}
}

func TestValuesAreCoalescedAndTheLastAlwaysSent(t *testing.T) {
	p := NewPool(context.Background(), &manualClock{}, 64, 1)
	jobs := make([]Job[report], 50)
	for i := range jobs {
		jobs[i] = Job[report]{Do: func(context.Context) func(*report) { return set(i, "x") }}
	}
	frame := report{parts: make([]string, 50)}
	values := 0
	var last report
	for v := range Run(context.Background(), p, frame, jobs, clone) {
		values++
		last = v
	}
	if values > 5 {
		t.Errorf("%d values for 50 instant answers, want them coalesced", values)
	}
	for i, part := range last.parts {
		if part != "x" {
			t.Fatalf("part %d of the last value is %q", i, part)
		}
	}
}

func TestBudgetStartsWhenTheJobStarts(t *testing.T) {
	clock := &manualClock{}
	p := NewPool(context.Background(), clock, 1, 1)
	hold := make(chan struct{})
	answer := make(chan struct{})
	jobs := []Job[report]{
		{Key: "hold", Budget: 10 * time.Second, Do: func(context.Context) func(*report) { <-hold; return set(0, "held") }},
		{Key: "queued", Budget: 2 * time.Second, Do: func(context.Context) func(*report) { <-answer; return set(1, "answered") }, Late: set(1, "late")},
	}
	ch := Run(context.Background(), p, report{parts: []string{"", ""}}, jobs, clone)
	eventually(t, func() bool { return clock.sleepers() == 1 })
	clock.advance(3 * time.Second)
	close(hold)
	eventually(t, func() bool { return clock.sleepers() == 1 })
	clock.advance(time.Second)
	close(answer)
	if last := Last(ch); last.parts[1] != "answered" {
		t.Fatalf("the queued job answered %q: its budget counted its time in the queue", last.parts[1])
	}
}

func TestALateJobAnswersItsLatePart(t *testing.T) {
	clock := &manualClock{}
	p := NewPool(context.Background(), clock, 1, 1)
	jobs := []Job[report]{{Key: "stuck", Budget: 2 * time.Second, Do: func(ctx context.Context) func(*report) {
		<-ctx.Done()
		return set(0, "answered")
	}, Late: set(0, "late")}}
	ch := Run(context.Background(), p, report{parts: []string{""}}, jobs, clone)
	eventually(t, func() bool { return clock.sleepers() == 1 })
	clock.advance(2 * time.Second)
	if last := Last(ch); last.parts[0] != "late" {
		t.Fatalf("a job past its budget answered %q", last.parts[0])
	}
}

func TestPreferredPrefixRunsFirst(t *testing.T) {
	p := NewPool(context.Background(), &manualClock{}, 1, 1)
	hold := make(chan struct{})
	var order []string
	var mu sync.Mutex
	record := func(name string) func(context.Context) func(*report) {
		return func(context.Context) func(*report) {
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			return func(*report) {}
		}
	}
	busy := Run(context.Background(), p, report{}, []Job[report]{{Key: "x/hold", Budget: time.Hour, Do: func(context.Context) func(*report) {
		<-hold
		return func(*report) {}
	}}}, clone)
	<-busy
	other := Run(context.Background(), p, report{}, []Job[report]{{Key: "storage/a", Budget: time.Hour, Do: record("storage")}}, clone)
	<-other
	preferred := Run(context.Background(), p, report{}, []Job[report]{{Key: "status/a", Budget: time.Hour, Do: record("status")}}, clone)
	<-preferred
	p.Prefer("status/")
	close(hold)
	Last(busy)
	Last(other)
	Last(preferred)
	if !slices.Equal(order, []string{"status", "storage"}) {
		t.Fatalf("the jobs started in the order %v, want the preferred prefix first", order)
	}
}

func TestASharedKeyRunsOnce(t *testing.T) {
	p := NewPool(context.Background(), &manualClock{}, 4, 1)
	var runs atomic.Int32
	job := Job[report]{Key: "apps/list/formulae", Do: func(context.Context) func(*report) {
		runs.Add(1)
		return set(0, "listed")
	}}
	a := Run(context.Background(), p, report{parts: []string{""}}, []Job[report]{job}, clone)
	b := Run(context.Background(), p, report{parts: []string{""}}, []Job[report]{job}, clone)
	if Last(a).parts[0] != "listed" || Last(b).parts[0] != "listed" {
		t.Fatal("a reader of the shared key missed its answer")
	}
	if n := runs.Load(); n != 1 {
		t.Fatalf("the shared key ran %d times, want once", n)
	}
	p.Forget("apps/")
	Last(Run(context.Background(), p, report{parts: []string{""}}, []Job[report]{job}, clone))
	if n := runs.Load(); n != 2 {
		t.Fatalf("after Forget the key ran %d times in all, want a second run", n)
	}
}

func TestAStreamIsSharedAndEveryReaderSeesItsLastValue(t *testing.T) {
	p := NewPool(context.Background(), &manualClock{}, 4, 1)
	var starts atomic.Int32
	start := func(context.Context) <-chan int {
		starts.Add(1)
		ch := make(chan int, 3)
		ch <- 1
		ch <- 2
		ch <- 3
		close(ch)
		return ch
	}
	if a, b := Last(Stream(context.Background(), p, "storage", start)), Last(Stream(context.Background(), p, "storage", start)); a != 3 || b != 3 {
		t.Fatalf("the readers ended on %d and %d, want 3", a, b)
	}
	if n := starts.Load(); n != 1 {
		t.Fatalf("the stream started %d times, want once", n)
	}
}

func TestCloseEndsASharedJobAndWaitsForIt(t *testing.T) {
	p := NewPool(context.Background(), &manualClock{}, 4, 1)
	started := make(chan struct{})
	var reaped atomic.Bool
	job := Job[report]{Key: "disk/walk", Budget: time.Hour, Do: func(ctx context.Context) func(*report) {
		close(started)
		<-ctx.Done()
		reaped.Store(true)
		return set(0, "killed")
	}}
	ch := Run(context.Background(), p, report{parts: []string{""}}, []Job[report]{job}, clone)
	<-started
	p.Close()
	if !reaped.Load() {
		t.Fatal("Close returned while a shared job still ran: its command outlives station")
	}
	Last(ch)
	Last(Run(context.Background(), p, report{parts: []string{""}}, []Job[report]{{Key: "after", Do: func(context.Context) func(*report) {
		t.Error("a job started on a closed pool")
		return set(0, "ran")
	}}}, clone))
}
