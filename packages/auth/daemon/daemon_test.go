package daemon

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeLog captures Printf calls so tests can assert on log output.
type fakeLog struct {
	mu    sync.Mutex
	lines []string
}

func (f *fakeLog) Printf(format string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lines = append(f.lines, fmt.Sprintf(format, args...))
}

func (f *fakeLog) Lines() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.lines))
	copy(out, f.lines)
	return out
}

func TestProgram_StartTickRunsImmediately(t *testing.T) {
	var calls int32
	p := &Program{
		Interval:  time.Hour, // long enough that only StartTick fires
		StartTick: true,
		Refresh: func(ctx context.Context) (int, []error) {
			atomic.AddInt32(&calls, 1)
			return 0, nil
		},
		Log: &fakeLog{},
	}

	if err := p.Start(nil); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop(nil)

	// StartTick should run before Start returns to its caller (or
	// immediately after). Wait briefly for the goroutine to fire.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&calls) >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected exactly 1 refresh call from StartTick, got %d", got)
	}
}

func TestProgram_TicksAtInterval(t *testing.T) {
	var calls int32
	p := &Program{
		Interval:  60 * time.Millisecond,
		StartTick: false,
		Refresh: func(ctx context.Context) (int, []error) {
			atomic.AddInt32(&calls, 1)
			return 1, nil
		},
		Log: &fakeLog{},
	}

	// Bypass MinInterval clamp by setting after Start would normalize.
	// Start clamps anything < MinInterval to DefaultInterval, so we
	// stub Refresh and call run directly with a tiny interval.
	ctx, cancel := context.WithCancel(context.Background())
	p.done = make(chan struct{})
	go p.run(ctx)

	time.Sleep(220 * time.Millisecond) // ~3 ticks
	cancel()
	<-p.done

	if got := atomic.LoadInt32(&calls); got < 2 {
		t.Errorf("expected ≥2 ticks in 220ms with 60ms interval, got %d", got)
	}
}

func TestProgram_StopCancelsTicker(t *testing.T) {
	var calls int32
	p := &Program{
		Interval:  10 * time.Millisecond,
		StartTick: false,
		Refresh: func(ctx context.Context) (int, []error) {
			atomic.AddInt32(&calls, 1)
			return 0, nil
		},
		Log: &fakeLog{},
	}

	// Same trick as above — bypass clamping by driving run directly.
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.done = make(chan struct{})
	go p.run(ctx)
	time.Sleep(50 * time.Millisecond)

	if err := p.Stop(nil); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	got := atomic.LoadInt32(&calls)
	time.Sleep(80 * time.Millisecond)
	if after := atomic.LoadInt32(&calls); after != got {
		t.Errorf("ticker kept firing after Stop: %d → %d", got, after)
	}
}

func TestProgram_ZeroIntervalGetsDefault(t *testing.T) {
	p := &Program{Interval: 0, Refresh: func(_ context.Context) (int, []error) { return 0, nil }}
	if err := p.Start(nil); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop(nil)
	if p.Interval != DefaultInterval {
		t.Errorf("expected zero interval to be raised to DefaultInterval (%v), got %v", DefaultInterval, p.Interval)
	}
}

func TestProgram_LogsRefreshCount(t *testing.T) {
	flog := &fakeLog{}
	p := &Program{
		Interval:  time.Hour,
		StartTick: true,
		Refresh: func(_ context.Context) (int, []error) {
			return 2, nil
		},
		Log: flog,
	}
	if err := p.Start(nil); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop(nil)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(flog.Lines()) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	lines := flog.Lines()
	if len(lines) == 0 {
		t.Fatal("expected at least one log line for a successful refresh")
	}
	if lines[0] != "refreshed 2 MCP token(s)" {
		t.Errorf("unexpected log line: %q", lines[0])
	}
}

func TestProgram_LogsErrorsAsWarnings(t *testing.T) {
	flog := &fakeLog{}
	wantErr := fmt.Errorf("boom")
	p := &Program{
		Interval:  time.Hour,
		StartTick: true,
		Refresh: func(_ context.Context) (int, []error) {
			return 0, []error{wantErr}
		},
		Log: flog,
	}
	if err := p.Start(nil); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop(nil)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(flog.Lines()) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	lines := flog.Lines()
	if len(lines) == 0 {
		t.Fatal("expected a warning log line")
	}
	if lines[0] != "warning: boom" {
		t.Errorf("unexpected log line: %q", lines[0])
	}
}
