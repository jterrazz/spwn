package goes

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

const (
	keepFirst = time.Second
	keepMost  = 5 * time.Minute
	// value: a run of a kept job that lasted this long before it broke was
	// quiet, so its next wait starts again at keepFirst.
	keepQuiet = 10 * time.Minute
	lockFor   = 2 * time.Second
	lockEvery = 10 * time.Millisecond
)

type Break struct {
	// invariant: a Break holds what a panic may say and never the panic's
	// value: the boundary's name, the value's type, its text through the
	// redactor (empty without one), and the stack of the goroutine that broke.
	// Of is what a boundary that serves many ran when it broke, a call's
	// name, data beside the literal Name; empty for any other boundary.
	Name  string
	Of    string
	Type  string
	Text  string
	Stack []byte
}

func (b *Break) Error() string {
	at := b.Name
	if b.Of != "" {
		at += " on " + b.Of
	}
	if b.Text == "" {
		return "broke in " + at + ": " + b.Type
	}
	return "broke in " + at + ": " + b.Type + ": " + b.Text
}

type Fatal interface {
	// invariant: a panic value that implements Fatal, or an error whose
	// chain holds one, is never recovered: Try, Go, Keep and Vital panic it
	// again, unreported. The spec build's secret tripwire and the strict page
	// seal panic with one.
	FatalBreak()
}

type Scripted interface {
	// invariant: in the spec build (the `world` tag) a panic value that is
	// not Scripted, nor an error whose chain holds one, is panicked again so
	// the run dies on it; a test or a world fault that breaks on purpose
	// panics with one, a Script.
	ScriptedBreak()
}

type Script string

func (Script) ScriptedBreak() {}

func (s Script) Error() string { return string(s) }

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type Lock interface {
	TryLock() bool
	Unlock()
}

type Options struct {
	// invariant: every field may stay nil. Without Log a break is counted
	// and hooked but written nowhere; without Redact its text is withheld;
	// Keep refuses to start without Clock; the lock watch runs only with
	// both Clock and Restart.
	Log     func(line string)
	Redact  func(text string) string
	Clock   Clock
	Restart func(lock string)
}

type watched struct {
	name string
	lock Lock
}

type set struct {
	opts      atomic.Pointer[Options]
	hook      atomic.Pointer[func(Break)]
	restarted atomic.Bool
	mu        sync.Mutex
	counts    map[string]int
	locks     []watched
	watching  bool
	again     bool
	// value: how a goroutine root dies on a value it may not recover: nil
	// is a panic, which ends the process; a test's records it and ends the
	// goroutine.
	die func(v any)
}

var std = &set{}

func Set(o Options) { std.opts.Store(&o) }

func OnBreak(fn func(Break)) { std.hook.Store(&fn) }

func WatchLock(name string, l Lock) {
	// invariant: a watched lock is one a break may leave held: after each
	// recovered break it is tried for lockFor, and one still held calls
	// Options.Restart with its name, at most once per process.
	std.watch(name, l)
}

func Breaks(name string) int { return std.breaks(name) }

func Try(name string, fn func() error) error {
	// invariant: every name given to goes is a string literal unique across
	// the engine, `<family>.<what>`; it names the boundary in the log and
	// keys its counter. Try runs fn on the caller's goroutine and returns
	// its error, or a *Break when fn panicked; a runtime.Goexit inside fn
	// ends the goroutine as Go defines it, with no break.
	return std.try(name, fn)
}

func TryOf(name, of string, fn func() error) error {
	// invariant: Try at a boundary that serves many, named by data: the
	// call it runs. name stays the literal that keys the counter; of rides
	// in the Break and its log line.
	return std.tryOf(name, of, fn)
}

func Go(name string, fn func()) {
	// invariant: fn runs on a goroutine of its own; a panic is reported and
	// ends that goroutine alone, after fn's own defers ran.
	go std.once(name, fn, false)
}

func Keep(ctx context.Context, name string, fn func(context.Context)) {
	// invariant: fn runs on a goroutine of its own and again after each
	// break, waiting keepFirst, doubling to keepMost, and keepFirst again
	// after a run that lasted keepQuiet; it never gives up. fn returning,
	// ctx ending or a runtime.Goexit ends the job.
	std.keep(ctx, name, fn)
}

func Vital(name string, fn func()) {
	// invariant: fn runs on a goroutine of its own; a panic is reported, then
	// panicked again so the process dies and its supervisor starts it afresh.
	go std.once(name, fn, true)
}

func Runner(name string) func(func()) {
	return func(fn func()) { Go(name, fn) }
}

func (s *set) watch(name string, l Lock) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locks = append(s.locks, watched{name: name, lock: l})
}

func (s *set) breaks(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[name]
}

func (s *set) try(name string, fn func() error) error { return s.tryOf(name, "", fn) }

func (s *set) tryOf(name, of string, fn func() error) (err error) {
	defer func() {
		v := recover()
		if v == nil {
			return
		}
		b, pass := s.caught(name, of, v)
		if pass != nil {
			panic(pass)
		}
		err = b
		s.afterBreak()
	}()
	return fn()
}

func (s *set) once(name string, fn func(), vital bool) (broke bool) {
	defer func() {
		v := recover()
		if v == nil {
			return
		}
		_, pass := s.caught(name, "", v)
		switch {
		case pass != nil:
			s.crash(pass)
		case vital:
			s.crash(v)
		default:
			broke = true
			s.afterBreak()
		}
	}()
	fn()
	return false
}

func (s *set) crash(v any) {
	if s.die == nil {
		panic(v)
	}
	s.die(v)
}

func (s *set) keep(ctx context.Context, name string, fn func(context.Context)) {
	o := s.opts.Load()
	if o == nil || o.Clock == nil {
		panic(misuse("goes: Keep " + name + " needs the Clock of goes.Set"))
	}
	go s.kept(ctx, o.Clock, name, fn)
}

func (s *set) kept(ctx context.Context, clock Clock, name string, fn func(context.Context)) {
	wait := keepFirst
	for ctx.Err() == nil {
		start := clock.Now()
		if !s.once(name, func() { fn(ctx) }, false) || ctx.Err() != nil {
			return
		}
		if clock.Now().Sub(start) >= keepQuiet {
			wait = keepFirst
		}
		if clock.Sleep(ctx, wait) != nil {
			return
		}
		wait = min(2*wait, keepMost)
	}
}

// invariant: caught judges a recovered value. A Fatal one, or in the spec
// build an unscripted one, comes back as pass, unreported. Any other is
// counted, logged and hooked as a Break; if that report panics, pass is a
// Fatal naming it, so the process dies loudly rather than lose the break.
func (s *set) caught(name, of string, v any) (b *Break, pass any) {
	if marked[Fatal](v) || (strict && !marked[Scripted](v)) {
		return nil, v
	}
	s.mu.Lock()
	if s.counts == nil {
		s.counts = map[string]int{}
	}
	s.counts[name]++
	s.mu.Unlock()
	b = &Break{Name: name, Of: of, Type: fmt.Sprintf("%T", v), Stack: stack()}
	if lost := s.report(b, v); lost != nil {
		return nil, lost
	}
	return b, nil
}

func (s *set) report(b *Break, v any) (lost any) {
	defer func() {
		if r := recover(); r != nil {
			lost = lostReport(fmt.Sprintf("goes: the report of a break in %s broke on a %T\n%s", b.Name, r, stack()))
		}
	}()
	o := s.opts.Load()
	if o != nil && o.Redact != nil {
		b.Text = o.Redact(fmt.Sprint(v))
	}
	if o != nil && o.Log != nil {
		o.Log(b.Error() + "\n" + string(b.Stack))
	}
	if h := s.hook.Load(); h != nil && *h != nil {
		(*h)(*b)
	}
	return nil
}

func (s *set) afterBreak() {
	o := s.opts.Load()
	if o == nil || o.Clock == nil || o.Restart == nil || s.restarted.Load() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.locks) == 0 {
		return
	}
	if s.watching {
		s.again = true
		return
	}
	s.watching = true
	go s.watchLocks(o)
}

func (s *set) watchLocks(o *Options) {
	for {
		s.mu.Lock()
		locks := slices.Clone(s.locks)
		s.mu.Unlock()
		for _, w := range locks {
			if !free(o.Clock, w.lock) && s.restarted.CompareAndSwap(false, true) {
				if o.Log != nil {
					o.Log("a lock stayed held after the break: " + w.name + "; the engine restarts")
				}
				o.Restart(w.name)
				break
			}
		}
		s.mu.Lock()
		if !s.again || s.restarted.Load() {
			s.watching, s.again = false, false
			s.mu.Unlock()
			return
		}
		s.again = false
		s.mu.Unlock()
	}
}

func free(clock Clock, l Lock) bool {
	deadline := clock.Now().Add(lockFor)
	for {
		if l.TryLock() {
			l.Unlock()
			return true
		}
		if !clock.Now().Before(deadline) {
			return false
		}
		_ = clock.Sleep(context.Background(), lockEvery)
	}
}

func marked[T any](v any) bool {
	if _, ok := v.(T); ok {
		return true
	}
	err, ok := v.(error)
	if !ok {
		return false
	}
	var t T
	return errors.As(err, &t)
}

func stack() []byte {
	buf := make([]byte, 4096)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}

type misuse string

func (misuse) FatalBreak() {}

func (m misuse) Error() string { return string(m) }

type lostReport string

func (lostReport) FatalBreak() {}

func (l lostReport) Error() string { return string(l) }
