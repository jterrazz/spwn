package automation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"spwn.sh/packages/project"
)

// engineFixture bundles the four collaborators tests almost always
// want fresh: a FakeClock anchored at testEpoch, a MockDispatcher,
// in-memory receipts + state. Plus the assembled Engine.
type engineFixture struct {
	clock      *FakeClock
	dispatcher *MockDispatcher
	receipts   *MemoryReceiptWriter
	state      *MemoryStateStore
	engine     *Engine
}

func newEngineFixture(t *testing.T) *engineFixture {
	t.Helper()
	clock := NewFakeClock(mustParse(t, testEpoch))
	disp := NewMockDispatcher()
	rec := NewMemoryReceiptWriter()
	state := NewMemoryStateStore()
	eng, err := New(Config{
		Clock:      clock,
		Dispatcher: disp,
		Receipts:   rec,
		State:      state,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return &engineFixture{clock, disp, rec, state, eng}
}

func cronAuto(expr, agent, prompt string) project.Automation {
	return project.Automation{
		On:      project.Trigger{Cron: expr},
		Agent:   agent,
		Prompt:  prompt,
		Catchup: "collapse",
	}
}

// waitForReceipts polls until the receipt count reaches n or the
// deadline elapses. Lets test goroutines race the engine's dispatch
// goroutine without sleep loops in every test.
func (f *engineFixture) waitForReceipts(t *testing.T, n int, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if len(f.receipts.Receipts()) >= n {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("expected %d receipts within %s; got %d", n, within, len(f.receipts.Receipts()))
}

// ── Lifecycle ───────────────────────────────────────────────────────

func TestEngine_New_RejectsNilCollaborators(t *testing.T) {
	cases := map[string]Config{
		"missing clock":      {Dispatcher: NewMockDispatcher(), Receipts: NewMemoryReceiptWriter(), State: NewMemoryStateStore()},
		"missing dispatcher": {Clock: NewFakeClock(time.Now()), Receipts: NewMemoryReceiptWriter(), State: NewMemoryStateStore()},
		"missing receipts":   {Clock: NewFakeClock(time.Now()), Dispatcher: NewMockDispatcher(), State: NewMemoryStateStore()},
		"missing state":      {Clock: NewFakeClock(time.Now()), Dispatcher: NewMockDispatcher(), Receipts: NewMemoryReceiptWriter()},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New(cfg); err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestEngine_RegisterAfterStartRejected(t *testing.T) {
	f := newEngineFixture(t)
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	err := f.engine.Register("brain", map[string]project.Automation{
		"x": cronAuto("0 6 * * *", "neo", "go"),
	})
	if err == nil {
		t.Error("Register after Start should error")
	}
}

func TestEngine_StartTwiceRejected(t *testing.T) {
	f := newEngineFixture(t)
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()
	if err := f.engine.Start(context.Background()); err == nil {
		t.Error("second Start should error")
	}
}

func TestEngine_StopBeforeStartIsNoop(t *testing.T) {
	f := newEngineFixture(t)
	// Should not panic, should not block.
	f.engine.Stop()
}

// ── On-time cron fires ──────────────────────────────────────────────

func TestEngine_CronFiresAtScheduledTime(t *testing.T) {
	f := newEngineFixture(t) // anchored at testEpoch = 2026-05-01T00:00:00Z
	must(t, f.engine.Register("brain", map[string]project.Automation{
		"morning-brief": cronAuto("0 6 * * *", "editor", "Brief."),
	}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	// Give the runCron goroutine a moment to register its first After().
	waitForPending(t, f.clock, 1, 200*time.Millisecond)

	// Advance to 6:00 — the timer fires, dispatch records, receipt
	// commits (asynchronously).
	f.clock.AdvanceTo(mustParse(t, "2026-05-01T06:00:00Z"))
	f.waitForReceipts(t, 1, 500*time.Millisecond)

	rec := f.receipts.Receipts()[0]
	if rec.World != "brain" || rec.Automation != "morning-brief" {
		t.Errorf("receipt = %+v", rec)
	}
	if rec.Reason != "on-time" {
		t.Errorf("reason = %q, want on-time", rec.Reason)
	}
	if !rec.OK {
		t.Errorf("ok = false, error = %q", rec.Error)
	}
	if !rec.Scheduled.Equal(mustParse(t, "2026-05-01T06:00:00Z")) {
		t.Errorf("scheduled = %s", rec.Scheduled)
	}
	if reqs := f.dispatcher.Requests(); len(reqs) != 1 || reqs[0].Prompt != "Brief." {
		t.Errorf("dispatch requests = %+v", reqs)
	}
}

func TestEngine_CronFiresMultipleTimes(t *testing.T) {
	f := newEngineFixture(t)
	must(t, f.engine.Register("brain", map[string]project.Automation{
		"hourly": cronAuto("0 * * * *", "editor", "tick"),
	}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	waitForPending(t, f.clock, 1, 200*time.Millisecond)

	for hour := 1; hour <= 3; hour++ {
		f.clock.AdvanceTo(mustParse(t, "2026-05-01T0"+string(rune('0'+hour))+":00:00Z"))
		f.waitForReceipts(t, hour, 500*time.Millisecond)
	}
	if got := f.dispatcher.Count(); got != 3 {
		t.Errorf("dispatch count = %d, want 3", got)
	}
}

// ── Catch-up ────────────────────────────────────────────────────────

func TestEngine_CatchupCollapseFiresOnceForMissedSlots(t *testing.T) {
	f := newEngineFixture(t)
	// Pre-seed state: last fired Sunday 2026-04-26 at 06:00.
	last := mustParse(t, "2026-04-26T06:00:00Z")
	must(t, f.state.RecordFire("brain", "morning-brief", last))

	// Anchor the clock at 2026-04-29 08:14 — three 6am slots have
	// passed since the last fire (Mon, Tue, Wed).
	f.clock = NewFakeClock(mustParse(t, "2026-04-29T08:14:00Z"))
	f.engine, _ = New(Config{
		Clock:      f.clock,
		Dispatcher: f.dispatcher,
		Receipts:   f.receipts,
		State:      f.state,
	})

	must(t, f.engine.Register("brain", map[string]project.Automation{
		"morning-brief": cronAuto("0 6 * * *", "editor", "go"),
	}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	// Catch-up runs synchronously inside Start, so the receipt is
	// already committed by the time Start returns.
	recs := f.receipts.Receipts()
	if len(recs) != 1 {
		t.Fatalf("catch-up should fire exactly once; got %d receipts", len(recs))
	}
	got := recs[0]
	if got.Reason != "catchup" {
		t.Errorf("reason = %q, want catchup", got.Reason)
	}
	if got.Missed != 3 {
		t.Errorf("missed = %d, want 3 (Mon/Tue/Wed)", got.Missed)
	}
	if !got.LastFired.Equal(last) {
		t.Errorf("last_fired = %s, want %s", got.LastFired, last)
	}
	// Scheduled should be the most recent missed slot, not the first.
	if !got.Scheduled.Equal(mustParse(t, "2026-04-29T06:00:00Z")) {
		t.Errorf("scheduled = %s, want 2026-04-29T06:00 (latest missed)", got.Scheduled)
	}
}

func TestEngine_CatchupSkipFiresZero(t *testing.T) {
	f := newEngineFixture(t)
	// Same setup, skip mode.
	must(t, f.state.RecordFire("brain", "morning-brief", mustParse(t, "2026-04-26T06:00:00Z")))
	f.clock = NewFakeClock(mustParse(t, "2026-04-29T08:14:00Z"))
	f.engine, _ = New(Config{
		Clock:      f.clock,
		Dispatcher: f.dispatcher,
		Receipts:   f.receipts,
		State:      f.state,
	})

	auto := cronAuto("0 6 * * *", "editor", "go")
	auto.Catchup = "skip"
	must(t, f.engine.Register("brain", map[string]project.Automation{"morning-brief": auto}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	if got := len(f.receipts.Receipts()); got != 0 {
		t.Errorf("skip mode produced %d catch-up receipts, want 0", got)
	}

	// Sanity: schedule still resumes — advance to next 6am, fire normally.
	waitForPending(t, f.clock, 1, 200*time.Millisecond)
	f.clock.AdvanceTo(mustParse(t, "2026-04-30T06:00:00Z"))
	f.waitForReceipts(t, 1, 500*time.Millisecond)
	if got := f.receipts.Receipts()[0].Reason; got != "on-time" {
		t.Errorf("post-skip resume = %q, want on-time", got)
	}
}

func TestEngine_CatchupStack_FiresOncePerMissedSlot(t *testing.T) {
	// catchup: stack — one fire per missed cron slot, in order.
	// Setup: cursor at Mon 06:00, boot at Wed 08:00, 6am cron.
	// Should produce 2 catch-up receipts (Tue + Wed slots) plus
	// schedule resumes for Thu onwards.
	f := newEngineFixture(t)
	last := mustParse(t, "2026-04-27T06:00:00Z") // Mon
	must(t, f.state.RecordFire("brain", "x", last))
	f.clock = NewFakeClock(mustParse(t, "2026-04-29T08:00:00Z")) // Wed 08:00
	f.engine, _ = New(Config{
		Clock: f.clock, Dispatcher: f.dispatcher, Receipts: f.receipts, State: f.state,
	})

	auto := cronAuto("0 6 * * *", "editor", "go")
	auto.Catchup = "stack"
	must(t, f.engine.Register("brain", map[string]project.Automation{"x": auto}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	recs := f.receipts.Receipts()
	if len(recs) != 2 {
		t.Fatalf("stack mode should produce 2 catch-up receipts (Tue + Wed); got %d", len(recs))
	}
	if !recs[0].Scheduled.Equal(mustParse(t, "2026-04-28T06:00:00Z")) {
		t.Errorf("first catchup Scheduled = %s, want Tue 06:00", recs[0].Scheduled)
	}
	if !recs[1].Scheduled.Equal(mustParse(t, "2026-04-29T06:00:00Z")) {
		t.Errorf("second catchup Scheduled = %s, want Wed 06:00", recs[1].Scheduled)
	}
	for _, rec := range recs {
		if rec.Missed != 1 {
			t.Errorf("stack-mode receipt should have Missed=1, got %d", rec.Missed)
		}
		if rec.Reason != "catchup" {
			t.Errorf("Reason = %q, want catchup", rec.Reason)
		}
	}
}

func TestEngine_CatchupStack_RespectsCap(t *testing.T) {
	// 200 missed minutely slots → cap caps at 100. Verify we don't
	// blast 200 receipts.
	f := newEngineFixture(t)
	last := mustParse(t, "2026-05-01T00:00:00Z")
	must(t, f.state.RecordFire("brain", "x", last))
	// Advance ~3.5 hours = 210 minutely slots; cap should clamp.
	f.clock = NewFakeClock(mustParse(t, "2026-05-01T03:30:00Z"))
	f.engine, _ = New(Config{
		Clock: f.clock, Dispatcher: f.dispatcher, Receipts: f.receipts, State: f.state,
	})

	auto := cronAuto("* * * * *", "editor", "go")
	auto.Catchup = "stack"
	must(t, f.engine.Register("brain", map[string]project.Automation{"x": auto}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	got := len(f.receipts.Receipts())
	if got != catchUpStackCap {
		t.Errorf("stack cap test: got %d receipts, want %d (cap)", got, catchUpStackCap)
	}
}

func TestEngine_CatchupNoPriorFireSkipsCatchup(t *testing.T) {
	// First-ever boot — engine has nothing in state. Even with N
	// "missed" slots in the abstract, we don't fire a catch-up
	// because the engine has no view of "what should have happened
	// before I existed".
	f := newEngineFixture(t)
	f.clock = NewFakeClock(mustParse(t, "2026-04-29T08:14:00Z"))
	f.engine, _ = New(Config{
		Clock:      f.clock,
		Dispatcher: f.dispatcher,
		Receipts:   f.receipts,
		State:      f.state,
	})
	must(t, f.engine.Register("brain", map[string]project.Automation{
		"morning-brief": cronAuto("0 6 * * *", "editor", "go"),
	}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	if got := len(f.receipts.Receipts()); got != 0 {
		t.Errorf("first-boot produced %d catch-up receipts, want 0", got)
	}
}

func TestEngine_CatchupAdvancesStateCursor(t *testing.T) {
	// The catch-up fire records the SCHEDULED time of the slot it
	// covered, not the wall-clock fired time. Otherwise the next
	// catch-up would re-detect the same slots (or skip the next
	// real one).
	f := newEngineFixture(t)
	last := mustParse(t, "2026-04-26T06:00:00Z")
	must(t, f.state.RecordFire("brain", "x", last))

	f.clock = NewFakeClock(mustParse(t, "2026-04-29T08:14:00Z"))
	f.engine, _ = New(Config{
		Clock:      f.clock,
		Dispatcher: f.dispatcher,
		Receipts:   f.receipts,
		State:      f.state,
	})
	must(t, f.engine.Register("brain", map[string]project.Automation{
		"x": cronAuto("0 6 * * *", "editor", "go"),
	}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	// State should advance to the most recent missed slot
	// (2026-04-29T06:00), NOT to "now" (08:14) and NOT to 2026-04-26.
	got, ok := f.state.LastFired("brain", "x")
	if !ok {
		t.Fatal("state should record after catch-up fire")
	}
	want := mustParse(t, "2026-04-29T06:00:00Z")
	if !got.Equal(want) {
		t.Errorf("state cursor = %s, want %s", got, want)
	}
}

// ── Per-agent serialisation ─────────────────────────────────────────

func TestEngine_PerAgentSerialisation(t *testing.T) {
	// Two automations targeting the SAME agent fire at the same
	// instant. The dispatcher is held until we release; we should
	// only see ONE in-flight at a time. Verifies the agentLocks
	// mutex.
	f := newEngineFixture(t)
	hold := make(chan struct{})
	f.dispatcher.Hold = hold

	must(t, f.engine.Register("brain", map[string]project.Automation{
		"a": cronAuto("0 6 * * *", "editor", "p1"),
		"b": cronAuto("0 6 * * *", "editor", "p2"),
	}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	waitForPending(t, f.clock, 2, 200*time.Millisecond)
	f.clock.AdvanceTo(mustParse(t, "2026-05-01T06:00:00Z"))

	// Wait for the FIRST dispatch to be in flight (waiting on hold).
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(f.dispatcher.Requests()) > 0 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if got := f.dispatcher.Count(); got != 1 {
		t.Errorf("after first hold: count = %d, want exactly 1 in flight", got)
	}

	// Release first — second should now proceed.
	hold <- struct{}{}
	hold <- struct{}{}
	f.waitForReceipts(t, 2, 500*time.Millisecond)
}

func TestEngine_CrossAgentParallel(t *testing.T) {
	// Two automations targeting DIFFERENT agents fire at the same
	// instant. They should both reach Dispatch concurrently.
	f := newEngineFixture(t)
	hold := make(chan struct{})
	f.dispatcher.Hold = hold

	must(t, f.engine.Register("brain", map[string]project.Automation{
		"a": cronAuto("0 6 * * *", "editor", "p"),
		"b": cronAuto("0 6 * * *", "curator", "p"),
	}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	waitForPending(t, f.clock, 2, 200*time.Millisecond)
	f.clock.AdvanceTo(mustParse(t, "2026-05-01T06:00:00Z"))

	// Both should reach Dispatch quickly even with hold blocking.
	// "Reach Dispatch" = the request slice has both entries (each
	// goroutine appends BEFORE waiting on hold in MockDispatcher? —
	// no, the mock waits on hold first, THEN records. So we observe
	// "two goroutines blocked on hold" via concurrent receipt
	// counting after we release.).
	close(hold) // release everything
	f.waitForReceipts(t, 2, 500*time.Millisecond)
}

// ── Dispatch errors ─────────────────────────────────────────────────

func TestEngine_DispatchErrorRecordedInReceipt(t *testing.T) {
	f := newEngineFixture(t)
	f.dispatcher.Err = errors.New("world cold-start failed: docker daemon down")

	must(t, f.engine.Register("brain", map[string]project.Automation{
		"brief": cronAuto("0 6 * * *", "editor", "go"),
	}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	waitForPending(t, f.clock, 1, 200*time.Millisecond)
	f.clock.AdvanceTo(mustParse(t, "2026-05-01T06:00:00Z"))
	f.waitForReceipts(t, 1, 500*time.Millisecond)

	rec := f.receipts.Receipts()[0]
	if rec.OK {
		t.Errorf("OK should be false on dispatch error")
	}
	if rec.Error == "" {
		t.Error("Error should be populated")
	}
}

func TestEngine_DispatchErrorDoesNotAdvanceState(t *testing.T) {
	// Failed dispatches must NOT advance the state cursor — otherwise
	// a transient failure would silently swallow that scheduled slot
	// from the next catch-up's view, and the user would notice their
	// agent missed a brief without any receipt-level reason.
	f := newEngineFixture(t)
	f.dispatcher.Err = errors.New("boom")

	must(t, f.engine.Register("brain", map[string]project.Automation{
		"x": cronAuto("0 6 * * *", "editor", "go"),
	}))
	must(t, f.engine.Start(context.Background()))
	defer f.engine.Stop()

	waitForPending(t, f.clock, 1, 200*time.Millisecond)
	f.clock.AdvanceTo(mustParse(t, "2026-05-01T06:00:00Z"))
	f.waitForReceipts(t, 1, 500*time.Millisecond)

	if _, ok := f.state.LastFired("brain", "x"); ok {
		t.Errorf("state should not record fire after dispatch error")
	}
}

// ── FS trigger end-to-end through the engine ───────────────────────

func TestEngine_FSFiresAfterDebounce(t *testing.T) {
	source := NewFakeFSSource()
	clock := NewFakeClock(mustParse(t, testEpoch))
	fsWatcher := NewFSWatcher(source, clock)
	disp := NewMockDispatcher()
	rec := NewMemoryReceiptWriter()
	state := NewMemoryStateStore()
	eng, err := New(Config{
		Clock:      clock,
		Dispatcher: disp,
		Receipts:   rec,
		State:      state,
		FS:         fsWatcher,
	})
	must(t, err)

	must(t, eng.Register("brain", map[string]project.Automation{
		"inbox": {
			On: project.Trigger{
				FS: &project.FSTrigger{
					Path:     "/inbox",
					Events:   []string{"create"},
					Debounce: project.Duration(100 * time.Millisecond),
				},
			},
			Agent:  "curator",
			Prompt: "new file: {{ .Event.Name }}",
		},
	}))
	must(t, eng.Start(context.Background()))
	defer eng.Stop()

	// Drive the watcher synchronously through its handle path so we
	// don't race the source pump.
	fsWatcher.handle(context.Background(), RawFSEvent{Path: "/inbox/foo.md", Kind: "create"})
	clock.Advance(100 * time.Millisecond)

	// The fire path is async (per-agent lock + dispatch); poll.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if disp.Count() >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	if got := disp.Count(); got != 1 {
		t.Fatalf("dispatch count = %d, want 1", got)
	}
	got := disp.Requests()[0]
	if got.World != "brain" || got.Agent != "curator" {
		t.Errorf("request = %+v", got)
	}
	if got.Prompt != "new file: foo.md" {
		t.Errorf("prompt = %q, want \"new file: foo.md\"", got.Prompt)
	}

	recs := rec.Receipts()
	if len(recs) != 1 || recs[0].Trigger != "fs" {
		t.Errorf("receipts = %+v", recs)
	}
	if recs[0].Reason != "create:foo.md" {
		t.Errorf("reason = %q, want create:foo.md", recs[0].Reason)
	}
}

// ── CommandResolver wiring ──────────────────────────────────────────

func TestEngine_CommandResolverLoadsBody(t *testing.T) {
	clock := NewFakeClock(mustParse(t, testEpoch))
	disp := NewMockDispatcher()
	rec := NewMemoryReceiptWriter()
	state := NewMemoryStateStore()

	// Fake resolver returns a templated body from the "filesystem".
	resolver := CommandResolverFunc(func(ref string) (string, error) {
		if ref == "command/morning-brief" {
			return "Brief for {{ .Now | date \"2006-01-02\" }}.", nil
		}
		return "", errors.New("not found")
	})

	eng, err := New(Config{
		Clock:      clock,
		Dispatcher: disp,
		Receipts:   rec,
		State:      state,
		Commands:   resolver,
	})
	must(t, err)

	must(t, eng.Register("brain", map[string]project.Automation{
		"brief": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Command: "command/morning-brief",
			Catchup: "collapse",
		},
	}))
	must(t, eng.Start(context.Background()))
	defer eng.Stop()

	// Wait for the runCron goroutine to register its first timer.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if clock.Pending() >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	clock.AdvanceTo(mustParse(t, "2026-05-01T06:00:00Z"))

	// Wait for dispatch.
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if disp.Count() >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	if disp.Count() != 1 {
		t.Fatalf("dispatch count = %d, want 1", disp.Count())
	}
	if got := disp.Requests()[0].Prompt; got != "Brief for 2026-05-01." {
		t.Errorf("prompt = %q", got)
	}
}

func TestEngine_CommandResolverErrorWritesFailedReceipt(t *testing.T) {
	clock := NewFakeClock(mustParse(t, testEpoch))
	disp := NewMockDispatcher()
	rec := NewMemoryReceiptWriter()
	state := NewMemoryStateStore()

	resolver := CommandResolverFunc(func(ref string) (string, error) {
		return "", errors.New("file not found")
	})

	eng, err := New(Config{
		Clock:      clock,
		Dispatcher: disp,
		Receipts:   rec,
		State:      state,
		Commands:   resolver,
	})
	must(t, err)
	must(t, eng.Register("brain", map[string]project.Automation{
		"brief": {
			On:      project.Trigger{Cron: "0 6 * * *"},
			Agent:   "editor",
			Command: "command/missing",
			Catchup: "collapse",
		},
	}))
	must(t, eng.Start(context.Background()))
	defer eng.Stop()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if clock.Pending() >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	clock.AdvanceTo(mustParse(t, "2026-05-01T06:00:00Z"))

	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(rec.Receipts()) >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	recs := rec.Receipts()
	if len(recs) != 1 {
		t.Fatalf("receipts = %d, want 1", len(recs))
	}
	if recs[0].OK {
		t.Errorf("receipt should be OK=false on resolver error")
	}
	if disp.Count() != 0 {
		t.Errorf("dispatch should not be called when resolver fails")
	}
}

// ── FS replay-on-startup ─────────────────────────────────────────

func TestEngine_FSReplay_FiresForNewerFiles(t *testing.T) {
	// Setup: last_fired cursor points at T0; two files exist —
	// "old.md" with mtime T-1h, "new.md" with mtime T+1h. On Start,
	// only new.md should be replayed.
	source := NewFakeFSSource()
	clock := NewFakeClock(mustParse(t, testEpoch))
	fsWatcher := NewFSWatcher(source, clock)
	disp := NewMockDispatcher()
	rec := NewMemoryReceiptWriter()
	state := NewMemoryStateStore()

	dir := t.TempDir()
	last := mustParse(t, testEpoch)
	must(t, state.RecordFire("brain", "inbox", last))

	// Write old + new files with crafted mtimes.
	oldPath := filepath.Join(dir, "old.md")
	newPath := filepath.Join(dir, "new.md")
	must(t, os.WriteFile(oldPath, []byte("old"), 0o644))
	must(t, os.WriteFile(newPath, []byte("new"), 0o644))
	mustChtimes(t, oldPath, last.Add(-1*time.Hour))
	mustChtimes(t, newPath, last.Add(1*time.Hour))

	eng, err := New(Config{
		Clock:      clock,
		Dispatcher: disp,
		Receipts:   rec,
		State:      state,
		FS:         fsWatcher,
	})
	must(t, err)
	must(t, eng.Register("brain", map[string]project.Automation{
		"inbox": {
			On: project.Trigger{
				FS: &project.FSTrigger{
					Path:     dir,
					Events:   []string{"create"},
					Debounce: project.Duration(100 * time.Millisecond),
				},
			},
			Agent:  "curator",
			Prompt: "{{ range .Event.Paths }}{{ . }}\n{{ end }}",
		},
	}))
	must(t, eng.Start(context.Background()))
	defer eng.Stop()

	// Replay runs synchronously inside Start, so the receipt
	// should already exist.
	recs := rec.Receipts()
	if len(recs) != 1 {
		t.Fatalf("got %d replay receipts, want 1", len(recs))
	}
	got := recs[0]
	if len(got.EventPaths) != 1 || got.EventPaths[0] != newPath {
		t.Errorf("EventPaths = %v, want [%s]", got.EventPaths, newPath)
	}
	if got.Reason != "replay:new.md" {
		t.Errorf("Reason = %q, want replay:new.md", got.Reason)
	}
}

func TestEngine_FSReplay_SkippedOnFirstBoot(t *testing.T) {
	// No state cursor — engine has no baseline to compare against.
	// Replay must skip even if files exist with future mtimes.
	source := NewFakeFSSource()
	clock := NewFakeClock(mustParse(t, testEpoch))
	fsWatcher := NewFSWatcher(source, clock)
	disp := NewMockDispatcher()
	rec := NewMemoryReceiptWriter()
	state := NewMemoryStateStore()

	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "b.md"), []byte("b"), 0o644))

	eng, err := New(Config{
		Clock: clock, Dispatcher: disp, Receipts: rec, State: state, FS: fsWatcher,
	})
	must(t, err)
	must(t, eng.Register("brain", map[string]project.Automation{
		"inbox": {
			On: project.Trigger{FS: &project.FSTrigger{
				Path: dir, Events: []string{"create"}, Debounce: project.Duration(100 * time.Millisecond),
			}},
			Agent:  "curator",
			Prompt: "p",
		},
	}))
	must(t, eng.Start(context.Background()))
	defer eng.Stop()

	if got := len(rec.Receipts()); got != 0 {
		t.Errorf("first-boot replay produced %d receipts, want 0", got)
	}
}

func TestEngine_FSReplay_SkipModeDisablesReplay(t *testing.T) {
	source := NewFakeFSSource()
	clock := NewFakeClock(mustParse(t, testEpoch))
	fsWatcher := NewFSWatcher(source, clock)
	disp := NewMockDispatcher()
	rec := NewMemoryReceiptWriter()
	state := NewMemoryStateStore()

	dir := t.TempDir()
	last := mustParse(t, testEpoch)
	must(t, state.RecordFire("brain", "inbox", last))

	newer := filepath.Join(dir, "new.md")
	must(t, os.WriteFile(newer, []byte("n"), 0o644))
	mustChtimes(t, newer, last.Add(1*time.Hour))

	eng, err := New(Config{
		Clock: clock, Dispatcher: disp, Receipts: rec, State: state, FS: fsWatcher,
	})
	must(t, err)
	must(t, eng.Register("brain", map[string]project.Automation{
		"inbox": {
			On: project.Trigger{FS: &project.FSTrigger{
				Path: dir, Events: []string{"create"}, Debounce: project.Duration(100 * time.Millisecond),
			}},
			Agent:   "curator",
			Prompt:  "p",
			Catchup: "skip",
		},
	}))
	must(t, eng.Start(context.Background()))
	defer eng.Stop()

	if got := len(rec.Receipts()); got != 0 {
		t.Errorf("catchup:skip replay produced %d receipts, want 0", got)
	}
}

// mustChtimes sets atime + mtime on the given path or fails the test.
func mustChtimes(t *testing.T, path string, when time.Time) {
	t.Helper()
	must(t, os.Chtimes(path, when, when))
}

func TestEngine_FSWithoutWatcherInConfigRejects(t *testing.T) {
	// Defensive: if the engine is built without an FSWatcher, fs
	// automations must error at Register so a programmer doesn't
	// silently lose their triggers.
	clock := NewFakeClock(mustParse(t, testEpoch))
	eng, err := New(Config{
		Clock:      clock,
		Dispatcher: NewMockDispatcher(),
		Receipts:   NewMemoryReceiptWriter(),
		State:      NewMemoryStateStore(),
		// FS intentionally omitted
	})
	must(t, err)

	err = eng.Register("brain", map[string]project.Automation{
		"x": {
			On:     project.Trigger{FS: &project.FSTrigger{Path: "/x"}},
			Agent:  "curator",
			Prompt: "p",
		},
	})
	if err == nil {
		t.Error("expected error registering fs trigger without FSWatcher")
	}
}

// ── Error logging surfaces ──────────────────────────────────────────

// captureLogger is a Logger that records Warnf calls. Used to assert
// that swallowed errors are surfaced rather than silently dropped.
type captureLogger struct {
	mu   sync.Mutex
	msgs []string
}

func (c *captureLogger) Warnf(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, format)
}

func (c *captureLogger) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.msgs))
	copy(out, c.msgs)
	return out
}

// failingReceiptWriter returns a stub error from every Write so we
// can confirm the engine logs the failure rather than swallowing it.
type failingReceiptWriter struct{}

func (failingReceiptWriter) Write(_ Receipt) error { return errors.New("disk full") }

func TestEngine_ReceiptWriteFailureIsLogged(t *testing.T) {
	clock := NewFakeClock(mustParse(t, testEpoch))
	logger := &captureLogger{}
	disp := NewMockDispatcher()
	state := NewMemoryStateStore()

	eng, err := New(Config{
		Clock:      clock,
		Dispatcher: disp,
		Receipts:   failingReceiptWriter{},
		State:      state,
		Logger:     logger,
	})
	must(t, err)
	must(t, eng.Register("brain", map[string]project.Automation{
		"brief": cronAuto("0 6 * * *", "editor", "go"),
	}))
	must(t, eng.Start(context.Background()))
	defer eng.Stop()

	// Wait for the cron loop to register its timer, then advance
	// past the slot.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if clock.Pending() >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	clock.AdvanceTo(mustParse(t, "2026-05-01T06:00:00Z"))

	// Wait for the dispatcher to be hit (which means fire ran +
	// receipt write was attempted).
	deadline = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if disp.Count() >= 1 && len(logger.snapshot()) >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	got := logger.snapshot()
	if len(got) == 0 {
		t.Fatal("expected at least one Warnf call from the failed receipt write")
	}
	found := false
	for _, msg := range got {
		if strings.Contains(msg, "receipt write failed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'receipt write failed' in logger messages, got: %v", got)
	}
}

// ── countMissed / lastScheduled (white-box) ─────────────────────────

func TestCountMissed_OnExactSlot(t *testing.T) {
	// Edge case: `last` exactly on a cron boundary. Schedule.Next
	// returns the NEXT slot strictly after, so the matching one
	// itself should not double-count.
	f := newEngineFixture(t)
	_ = f
	parser := f.engine.cronParser
	sched, err := parser.Parse("0 6 * * *")
	if err != nil {
		t.Fatal(err)
	}
	last := mustParse(t, "2026-05-01T06:00:00Z")
	now := mustParse(t, "2026-05-03T06:00:00Z") // 2 more 6ams: May-2 and May-3
	if got := countMissed(sched, last, now); got != 2 {
		t.Errorf("countMissed = %d, want 2", got)
	}
}

func TestCountMissed_NoSlots(t *testing.T) {
	f := newEngineFixture(t)
	parser := f.engine.cronParser
	sched, _ := parser.Parse("0 6 * * *")
	last := mustParse(t, "2026-05-01T06:00:00Z")
	now := mustParse(t, "2026-05-01T07:00:00Z") // same day, only 1h passed
	if got := countMissed(sched, last, now); got != 0 {
		t.Errorf("countMissed = %d, want 0", got)
	}
}

// ── helpers ─────────────────────────────────────────────────────────

// waitForPending blocks until the FakeClock has at least `n` pending
// timers or the deadline elapses. Used to synchronise tests with the
// engine's runCron goroutines that register their first After()
// asynchronously after Start.
func waitForPending(t *testing.T, c *FakeClock, n int, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if c.Pending() >= n {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("expected %d pending timers within %s; got %d", n, within, c.Pending())
}

