package readers

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
)

type Pace int

const (
	Idle Pace = iota
	Attached
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type Reader struct {
	Name string
	// value: the period while no client is attached, and while one is; a
	// zero period reads at every tick, and its slices never turn stale.
	Idle, Attached time.Duration
	Budget         time.Duration
	// value: one reading: its puts, the keys it could not read with why, and
	// every key it owns now; a key it owned before and no longer does is
	// dropped.
	Read func(ctx context.Context, now *store.Snapshot) Reading
}

type Reading struct {
	Puts   []store.Put
	Failed map[store.Key]string
	Owns   []store.Key
}

func (r *Reader) every(p Pace) time.Duration {
	if p == Attached {
		return r.Attached
	}
	return r.Idle
}

type Set struct {
	store *store.Store
	clock Clock
	// value: how a reading runs beside the tick: a goroutine in the
	// release build, inline in a world.
	run     func(func())
	readers []*Reader
	far     *Far

	mu      sync.Mutex
	asleep  bool
	pace    Pace
	clients int
	started map[string]time.Time
	running map[string]bool
	due     map[string]bool
	owner   map[store.Key]*Reader
}

type Options struct {
	Store *store.Store
	Clock Clock
	// value: nil runs each reading in a goroutine of its own.
	Run     func(func())
	Readers []*Reader
	// value: the readers wait for the store's first ask before they read,
	// as a stepped engine's do.
	Lazy bool
	// value: the other Macs' slices, held from their engines; nil holds none.
	Far *Far
}

func New(o Options) *Set {
	run := o.Run
	if run == nil {
		run = func(f func()) { go f() }
	}
	s := &Set{store: o.Store, clock: o.Clock, run: run, readers: o.Readers, asleep: o.Lazy, far: o.Far,
		started: map[string]time.Time{}, running: map[string]bool{}, due: map[string]bool{}, owner: map[store.Key]*Reader{}}
	if s.far != nil {
		s.far.Store, s.far.Clock, s.far.Pace = o.Store, o.Clock, s.Pace
	}
	return s
}

func (s *Set) Remember(held engine.Held) {
	if s.far != nil {
		s.far.Remember(held)
	}
}

func (s *Set) Held() engine.Held {
	if s.far == nil {
		return nil
	}
	return s.far.Held()
}

func (s *Set) Ready(ctx context.Context) {
	// invariant: a stepped engine stands for one that has run all along: its
	// readers read when its store is first asked, at once, so the answer holds
	// what such an engine read before, and a command that never asks it runs
	// no reading it did not ask for. It waits for every other Mac's first
	// snapshot too.
	s.mu.Lock()
	wake := s.asleep
	s.asleep = false
	s.mu.Unlock()
	if !wake {
		return
	}
	s.Tick(ctx, false)
	if s.far != nil {
		s.far.Warm(ctx)
	}
}

func (s *Set) Store() *store.Store { return s.store }

func (s *Set) Now() time.Time { return s.clock.Now() }

func (s *Set) Sleep(ctx context.Context, d time.Duration) error { return s.clock.Sleep(ctx, d) }

func (s *Set) Attach() (release func()) {
	// invariant: a client of engine.state holds the pace attached for as
	// long as it streams, as a watch of pages does.
	s.mu.Lock()
	s.clients++
	s.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			s.clients--
			s.mu.Unlock()
		})
	}
}

func (s *Set) Pace() Pace {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pace
}

func (s *Set) Due(names ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range names {
		s.due[n] = true
	}
}

func (s *Set) Moved() { s.Due(NameMac, NamePorts) }

func (s *Set) DueAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.readers {
		s.due[r.Name] = true
	}
}

func (s *Set) Tick(ctx context.Context, watched bool) {
	// invariant: a tick starts every reader that is due and not running, and
	// never waits on one; the store is swept and its clock moved first, so a
	// slice past its period plus its budget is stale before anything reads it.
	now := s.clock.Now()
	s.mu.Lock()
	if s.asleep {
		s.mu.Unlock()
		return
	}
	s.pace = Idle
	if watched || s.clients > 0 {
		s.pace = Attached
	}
	pace := s.pace
	var start []*Reader
	for _, r := range s.readers {
		if s.running[r.Name] {
			continue
		}
		last, ever := s.started[r.Name]
		if !ever || s.due[r.Name] || now.Sub(last) >= r.every(pace) {
			s.running[r.Name], s.started[r.Name] = true, now
			delete(s.due, r.Name)
			start = append(start, r)
		}
	}
	s.mu.Unlock()
	s.store.Sweep(now, s.cadence)
	s.store.Clock(now)
	for _, r := range start {
		s.run(func() { s.read(ctx, r) })
	}
	if s.far != nil {
		s.far.Tick(ctx)
	}
}

func (s *Set) cadence(k store.Key) (store.Cadence, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.owner[k]
	if !ok || r.every(s.pace) == 0 {
		return store.Cadence{}, false
	}
	return store.Cadence{Every: r.every(s.pace), Budget: r.Budget}, true
}

func (s *Set) read(ctx context.Context, r *Reader) {
	defer func() {
		s.mu.Lock()
		s.running[r.Name] = false
		s.mu.Unlock()
	}()
	in := ctx
	if r.Budget > 0 {
		var cancel context.CancelFunc
		in, cancel = context.WithTimeout(ctx, r.Budget)
		defer cancel()
	}
	got := r.Read(in, s.store.Now())
	if ctx.Err() != nil {
		return
	}
	owns := map[store.Key]bool{}
	for _, k := range got.Owns {
		owns[k] = true
	}
	s.mu.Lock()
	var gone []store.Key
	for k, o := range s.owner {
		if o == r && !owns[k] {
			gone = append(gone, k)
			delete(s.owner, k)
		}
	}
	for k := range owns {
		s.owner[k] = r
	}
	s.mu.Unlock()
	at := s.clock.Now()
	for i := range got.Puts {
		if got.Puts[i].ReadAt.IsZero() {
			got.Puts[i].ReadAt = at
		}
	}
	_ = s.store.Put(got.Puts...)
	for _, k := range slices.Sorted(maps.Keys(got.Failed)) {
		_ = s.store.Fail(k, got.Failed[k])
	}
	for _, k := range gone {
		s.store.Drop(k)
	}
}

func (s *Set) Owns(k store.Key) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.owner[k]
	return ok
}

func Why(err error) string {
	// value: the word a failed reading leaves on its slice: late past its
	// budget, unreachable for a machine that does not answer, else the
	// refusal's own code.
	var refused *fail.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return store.WhyLate
	case errors.As(err, &refused) && refused.Type == fail.Unreachable:
		return store.WhyUnreachable
	case errors.As(err, &refused) && refused.Type == fail.FarTooOld:
		return store.WhyFarTooOld
	case errors.As(err, &refused) && refused.Code != "":
		return string(refused.Code)
	}
	return "failed"
}
