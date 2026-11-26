package goes

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type fakeClock struct {
	mu    sync.Mutex
	now   time.Time
	slept []time.Duration
	// value: called after each sleep with the time slept so far, on the
	// sleeper's goroutine.
	after func(elapsed time.Duration)
	// value: a blocking clock's Sleep returns only when its context ends.
	block bool
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if c.block {
		<-ctx.Done()
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	c.slept = append(c.slept, d)
	c.now = c.now.Add(d)
	elapsed := c.now.Sub(time.Time{})
	after := c.after
	c.mu.Unlock()
	if after != nil {
		after(elapsed)
	}
	return nil
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func (c *fakeClock) sleeps() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.slept)
}

type seen struct {
	lines    []string
	hooked   []Break
	deaths   []any
	restarts []string
}

type record struct {
	mu sync.Mutex
	seen
}

func (r *record) log(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines = append(r.lines, line)
}

func (r *record) hook(b Break) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooked = append(r.hooked, b)
}

func (r *record) die(v any) {
	r.mu.Lock()
	r.deaths = append(r.deaths, v)
	r.mu.Unlock()
	runtime.Goexit()
}

func (r *record) restart(lock string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.restarts = append(r.restarts, lock)
}

func (r *record) snapshot() seen {
	r.mu.Lock()
	defer r.mu.Unlock()
	return seen{lines: slices.Clone(r.lines), hooked: slices.Clone(r.hooked), deaths: slices.Clone(r.deaths), restarts: slices.Clone(r.restarts)}
}

func redact(text string) string { return strings.ReplaceAll(text, "hunter2", "•••") }

func newSet(r *record, c *fakeClock) *set {
	s := &set{die: r.die}
	o := Options{Log: r.log, Redact: redact, Restart: r.restart}
	if c != nil {
		o.Clock = c
	}
	s.opts.Store(&o)
	hook := r.hook
	s.hook.Store(&hook)
	return s
}

type tripped struct{}

func (tripped) FatalBreak() {}

func (tripped) Error() string { return "tripwire" }

var errPlain = errors.New("plain")

func TestTryReturnsTheCallsOwnError(t *testing.T) {
	r := &record{}
	s := newSet(r, nil)
	if err := s.try("call.ok", func() error { return nil }); err != nil {
		t.Fatalf("Try of a call that answers = %v, want nil", err)
	}
	if err := s.try("call.refused", func() error { return errPlain }); !errors.Is(err, errPlain) {
		t.Fatalf("Try of a call that fails = %v, want its own error", err)
	}
	if got := r.snapshot(); len(got.lines) != 0 || s.breaks("call.refused") != 0 {
		t.Fatalf("a call that returned is no break: lines %q, count %d", got.lines, s.breaks("call.refused"))
	}
}

func TestTryTurnsAPanicIntoABreak(t *testing.T) {
	r := &record{}
	s := newSet(r, nil)
	err := s.try("machines.create", func() error { panic(Script("clone failed")) })
	var b *Break
	if !errors.As(err, &b) {
		t.Fatalf("Try of a call that panicked = %v, want a *Break", err)
	}
	if b.Name != "machines.create" || b.Type != "goes.Script" || b.Text != "clone failed" {
		t.Fatalf("break = %+v", *b)
	}
	if !strings.Contains(string(b.Stack), "goes_test.go") {
		t.Fatalf("the stack does not reach the panic's site:\n%s", b.Stack)
	}
	if b.Error() != "broke in machines.create: goes.Script: clone failed" {
		t.Fatalf("Error() = %q", b.Error())
	}
	got := r.snapshot()
	if len(got.lines) != 1 || !strings.HasPrefix(got.lines[0], b.Error()+"\n") || !strings.Contains(got.lines[0], "goes_test.go") {
		t.Fatalf("log = %q, want the break's line then its stack", got.lines)
	}
	if len(got.hooked) != 1 || got.hooked[0].Name != "machines.create" {
		t.Fatalf("hook got %+v", got.hooked)
	}
	_ = s.try("machines.create", func() error { panic(Script("again")) })
	if n := s.breaks("machines.create"); n != 2 {
		t.Fatalf("breaks = %d, want 2", n)
	}
	if n := s.breaks("machines.start"); n != 0 {
		t.Fatalf("another name's breaks = %d, want 0", n)
	}
}

func TestTheRedactorWritesTheText(t *testing.T) {
	r := &record{}
	s := newSet(r, nil)
	err := s.try("vault.read", func() error { panic(Script("token hunter2 leaked")) })
	var b *Break
	if !errors.As(err, &b) || b.Text != "token ••• leaked" {
		t.Fatalf("break = %v, want the redacted text", err)
	}
	if got := r.snapshot(); len(got.lines) != 1 || strings.Contains(got.lines[0], "hunter2") || strings.Contains(got.hooked[0].Text, "hunter2") {
		t.Fatalf("a secret reached the log or the hook: %q", got.lines)
	}
}

func TestWithoutARedactorTheTextIsWithheld(t *testing.T) {
	r := &record{}
	s := &set{}
	s.opts.Store(&Options{Log: r.log})
	err := s.try("vault.read", func() error { panic(Script("token hunter2")) })
	var b *Break
	if !errors.As(err, &b) || b.Text != "" || b.Error() != "broke in vault.read: goes.Script" {
		t.Fatalf("break = %v, want its type alone", err)
	}
	if got := r.snapshot(); len(got.lines) != 1 || strings.Contains(got.lines[0], "hunter2") {
		t.Fatalf("log = %q", got.lines)
	}
}

func TestUnconfiguredABreakIsStillCounted(t *testing.T) {
	s := &set{}
	if err := s.try("call.x", func() error { panic(Script("x")) }); err == nil || s.breaks("call.x") != 1 {
		t.Fatalf("Try = %v, breaks %d", err, s.breaks("call.x"))
	}
}

func TestAFatalPanicPassesEveryPolicy(t *testing.T) {
	for _, value := range []any{tripped{}, fmt.Errorf("seal: %w", tripped{})} {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := &record{}
				c := &fakeClock{}
				s := newSet(r, c)
				var mu sync.Mutex
				s.watch("engine.mu", &mu)
				func() {
					defer func() {
						if got := recover(); got != value {
							t.Errorf("Try let %v through, want %v", got, value)
						}
					}()
					_ = s.try("call.x", func() error { panic(value) })
					t.Error("Try returned on a Fatal panic")
				}()
				go s.once("job.x", func() { panic(value) }, false)
				var runs atomic.Int32
				s.keep(t.Context(), "loop.x", func(context.Context) { runs.Add(1); panic(value) })
				synctest.Wait()
				got := r.snapshot()
				if len(got.deaths) != 2 || got.deaths[0] != value || got.deaths[1] != value {
					t.Fatalf("deaths = %v, want the value twice (Go, Keep)", got.deaths)
				}
				if runs.Load() != 1 || len(c.sleeps()) != 0 {
					t.Fatalf("Keep ran %d times and slept %v after a Fatal panic", runs.Load(), c.sleeps())
				}
				if len(got.lines) != 0 || len(got.hooked) != 0 || s.breaks("call.x")+s.breaks("job.x")+s.breaks("loop.x") != 0 || len(got.restarts) != 0 {
					t.Fatalf("a Fatal panic was reported as a break: %q", got.lines)
				}
			})
		})
	}
}

func TestGoEndsItsGoroutineAfterItsDefers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		s := newSet(r, nil)
		closed := make(chan struct{})
		go s.once("engine.session", func() {
			defer close(closed)
			panic(Script("guest gone"))
		}, false)
		<-closed
		synctest.Wait()
		got := r.snapshot()
		if s.breaks("engine.session") != 1 || len(got.deaths) != 0 || len(got.lines) != 1 {
			t.Fatalf("breaks %d, deaths %v, lines %q", s.breaks("engine.session"), got.deaths, got.lines)
		}
	})
}

func TestVitalReportsThenDies(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		s := newSet(r, nil)
		go s.once("machines.run-watch", func() { panic(Script("watch lost")) }, true)
		go s.once("machines.run-watch", func() {}, true)
		synctest.Wait()
		got := r.snapshot()
		if len(got.lines) != 1 || !strings.HasPrefix(got.lines[0], "broke in machines.run-watch: goes.Script: watch lost") {
			t.Fatalf("lines = %q", got.lines)
		}
		if len(got.deaths) != 1 || got.deaths[0] != Script("watch lost") {
			t.Fatalf("deaths = %v, want the panic's own value once", got.deaths)
		}
	})
}

func TestKeepBacksOffToFiveMinutes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		c := &fakeClock{}
		s := newSet(r, c)
		var runs atomic.Int32
		s.keep(t.Context(), "engine.drift", func(context.Context) {
			if runs.Add(1) <= 12 {
				panic(Script("drift"))
			}
		})
		synctest.Wait()
		want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 32 * time.Second,
			64 * time.Second, 128 * time.Second, 256 * time.Second, 5 * time.Minute, 5 * time.Minute, 5 * time.Minute}
		if got := c.sleeps(); !slices.Equal(got, want) {
			t.Fatalf("waits = %v, want %v", got, want)
		}
		if runs.Load() != 13 || s.breaks("engine.drift") != 12 {
			t.Fatalf("runs %d, breaks %d", runs.Load(), s.breaks("engine.drift"))
		}
	})
}

func TestKeepStartsOverAfterAQuietRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := &fakeClock{}
		s := newSet(&record{}, c)
		var runs atomic.Int32
		s.keep(t.Context(), "engine.readers", func(context.Context) {
			switch runs.Add(1) {
			case 1, 2:
				panic(Script("early"))
			case 3:
				c.advance(keepQuiet)
				panic(Script("late"))
			case 4:
				c.advance(keepQuiet - time.Second)
				panic(Script("not quiet"))
			}
		})
		synctest.Wait()
		want := []time.Duration{time.Second, 2 * time.Second, time.Second, 2 * time.Second}
		if got := c.sleeps(); !slices.Equal(got, want) {
			t.Fatalf("waits = %v, want %v", got, want)
		}
	})
}

func TestKeepEndsWhenItsJobReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := &fakeClock{}
		s := newSet(&record{}, c)
		var runs atomic.Int32
		s.keep(t.Context(), "engine.awake", func(context.Context) { runs.Add(1) })
		synctest.Wait()
		if runs.Load() != 1 || len(c.sleeps()) != 0 {
			t.Fatalf("runs %d, waits %v: a job that returned is over", runs.Load(), c.sleeps())
		}
	})
}

func TestKeepLeavesNoGoroutineOnCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := &fakeClock{block: true}
		s := newSet(&record{}, c)
		ctx, cancel := context.WithCancel(t.Context())
		before := runtime.NumGoroutine()
		var runs atomic.Int32
		s.keep(ctx, "engine.exposure", func(context.Context) { runs.Add(1); panic(Script("x")) })
		synctest.Wait()
		if n := runtime.NumGoroutine(); n != before+1 {
			t.Fatalf("goroutines while waiting = %d, want %d", n, before+1)
		}
		cancel()
		synctest.Wait()
		if n := runtime.NumGoroutine(); n != before || runs.Load() != 1 {
			t.Fatalf("goroutines after cancel = %d, want %d; runs %d", n, before, runs.Load())
		}
	})
}

func TestKeepDoesNotRestartAJobCutByItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := &fakeClock{}
		s := newSet(&record{}, c)
		ctx, cancel := context.WithCancel(t.Context())
		var runs atomic.Int32
		s.keep(ctx, "engine.night", func(context.Context) { runs.Add(1); cancel(); panic(Script("x")) })
		synctest.Wait()
		if runs.Load() != 1 || len(c.sleeps()) != 0 {
			t.Fatalf("runs %d, waits %v", runs.Load(), c.sleeps())
		}
	})
}

func TestKeepRefusesWithoutAClock(t *testing.T) {
	s := newSet(&record{}, nil)
	defer func() {
		v := recover()
		if !marked[Fatal](v) || !strings.Contains(fmt.Sprint(v), "engine.drift") {
			t.Fatalf("Keep without a clock panicked %v, want a Fatal naming the job", v)
		}
	}()
	s.keep(t.Context(), "engine.drift", func(context.Context) {})
}

func TestGoexitIsNoBreak(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		c := &fakeClock{}
		s := newSet(r, c)
		var returned atomic.Bool
		go func() {
			_ = s.try("call.x", func() error { runtime.Goexit(); return nil })
			returned.Store(true)
		}()
		go s.once("job.x", runtime.Goexit, false)
		var runs atomic.Int32
		s.keep(t.Context(), "loop.x", func(context.Context) { runs.Add(1); runtime.Goexit() })
		synctest.Wait()
		got := r.snapshot()
		if returned.Load() || runs.Load() != 1 || len(got.lines) != 0 || len(got.deaths) != 0 || len(c.sleeps()) != 0 {
			t.Fatalf("returned %v, runs %d, lines %q, deaths %v", returned.Load(), runs.Load(), got.lines, got.deaths)
		}
	})
}

func TestAReportThatBreaksCrashesLoudly(t *testing.T) {
	cases := map[string]func(*set){
		"log":    func(s *set) { c := *s.opts.Load(); c.Log = func(string) { panic("disk full") }; s.opts.Store(&c) },
		"redact": func(s *set) { c := *s.opts.Load(); c.Redact = func(string) string { panic("bad") }; s.opts.Store(&c) },
		"hook":   func(s *set) { h := func(Break) { panic("bad") }; s.hook.Store(&h) },
	}
	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := &record{}
				s := newSet(r, nil)
				breakIt(s)
				func() {
					defer func() {
						v := recover()
						if !marked[Fatal](v) || !strings.Contains(fmt.Sprint(v), "a break in call.x broke on a string") {
							t.Errorf("Try let %v through, want a Fatal naming the report", v)
						}
					}()
					_ = s.try("call.x", func() error { panic(Script("x")) })
				}()
				go s.once("job.x", func() { panic(Script("x")) }, false)
				synctest.Wait()
				if got := r.snapshot(); len(got.deaths) != 1 || !marked[Fatal](got.deaths[0]) {
					t.Fatalf("Go deaths = %v, want one Fatal", got.deaths)
				}
			})
		})
	}
}

func TestALockLeftHeldRestartsTheEngineOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		s := newSet(r, &fakeClock{})
		var mu, other sync.Mutex
		s.watch("engine.acts", &other)
		s.watch("engine.mu", &mu)
		go s.once("engine.session", func() {
			mu.Lock()
			panic(Script("held"))
		}, false)
		synctest.Wait()
		_ = s.try("call.x", func() error { panic(Script("again")) })
		synctest.Wait()
		got := r.snapshot()
		if !slices.Equal(got.restarts, []string{"engine.mu"}) {
			t.Fatalf("restarts = %v, want engine.mu once", got.restarts)
		}
		if !slices.Contains(got.lines, "a lock stayed held after the break: engine.mu; the engine restarts") {
			t.Fatalf("lines = %q", got.lines)
		}
	})
}

func TestADeferredUnlockRestartsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		s := newSet(r, &fakeClock{})
		var mu sync.Mutex
		s.watch("engine.mu", &mu)
		go s.once("engine.session", func() {
			mu.Lock()
			defer mu.Unlock()
			panic(Script("held"))
		}, false)
		synctest.Wait()
		if got := r.snapshot(); len(got.restarts) != 0 {
			t.Fatalf("restarts = %v", got.restarts)
		}
	})
}

func TestALockFreedWithinTheWindowRestartsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		c := &fakeClock{}
		var mu sync.Mutex
		mu.Lock()
		c.after = func(elapsed time.Duration) {
			if elapsed == 2*time.Second-lockEvery {
				mu.Unlock()
			}
		}
		s := newSet(r, c)
		s.watch("engine.mu", &mu)
		_ = s.try("call.x", func() error { panic(Script("x")) })
		synctest.Wait()
		if got := r.snapshot(); len(got.restarts) != 0 {
			t.Fatalf("restarts = %v: a holder that let go within %v is no leak", got.restarts, lockFor)
		}
		if mu.TryLock() {
			mu.Unlock()
		} else {
			t.Fatal("the watch left the lock held")
		}
	})
}

func TestABreakDuringAWatchIsWatchedAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		c := &fakeClock{}
		var first, later sync.Mutex
		s := newSet(r, c)
		s.watch("engine.acts", &later)
		s.watch("engine.mu", &first)
		first.Lock()
		c.after = func(elapsed time.Duration) {
			if elapsed == time.Second {
				later.Lock()
				_ = s.try("call.y", func() error { panic(Script("y")) })
				first.Unlock()
			}
		}
		_ = s.try("call.x", func() error { panic(Script("x")) })
		synctest.Wait()
		if got := r.snapshot(); !slices.Equal(got.restarts, []string{"engine.acts"}) {
			t.Fatalf("restarts = %v, want the lock a break left held during the first watch", got.restarts)
		}
	})
}

func TestAThousandBrokenGoroutinesLeaveNone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		s := newSet(r, nil)
		before := runtime.NumGoroutine()
		for range 1000 {
			go s.once("lanes.job", func() { panic(Script("x")) }, false)
		}
		synctest.Wait()
		if n := runtime.NumGoroutine(); n != before {
			t.Fatalf("goroutines = %d, want %d", n, before)
		}
		if n := s.breaks("lanes.job"); n != 1000 {
			t.Fatalf("breaks = %d, want 1000", n)
		}
	})
}

func TestThePackageRunsThroughOneSet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &record{}
		old := std
		std = &set{die: r.die}
		defer func() { std = old }()
		c := &fakeClock{}
		Set(Options{Log: r.log, Redact: redact, Clock: c, Restart: r.restart})
		OnBreak(r.hook)
		var mu sync.Mutex
		WatchLock("engine.mu", &mu)
		if err := Try("call.x", func() error { panic(Script("x")) }); err == nil {
			t.Fatal("Try = nil")
		}
		Go("job.x", func() { panic(Script("x")) })
		Runner("job.run")(func() { panic(Script("x")) })
		Vital("job.vital", func() { panic(Script("x")) })
		var runs atomic.Int32
		Keep(t.Context(), "loop.x", func(context.Context) {
			if runs.Add(1) == 1 {
				panic(Script("x"))
			}
		})
		synctest.Wait()
		for _, name := range []string{"call.x", "job.x", "job.run", "job.vital", "loop.x"} {
			if Breaks(name) != 1 {
				t.Errorf("Breaks(%s) = %d, want 1", name, Breaks(name))
			}
		}
		got := r.snapshot()
		if len(got.hooked) != 5 || len(got.deaths) != 1 || len(got.restarts) != 0 || runs.Load() != 2 {
			t.Fatalf("hooked %d, deaths %v, restarts %v, runs %d", len(got.hooked), got.deaths, got.restarts, runs.Load())
		}
	})
}

func ok() error { return nil }

func BenchmarkTry(b *testing.B) {
	s := newSet(&record{}, nil)
	for b.Loop() {
		_ = s.try("call.x", ok)
	}
}

func BenchmarkTryBreak(b *testing.B) {
	s := &set{}
	broken := func() error { panic(Script("x")) }
	for b.Loop() {
		_ = s.try("call.x", broken)
	}
}
