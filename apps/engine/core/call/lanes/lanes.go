package lanes

import (
	"context"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

type Lane int

const (
	Commands Lane = iota
	Walks
	// invariant: an Inline job holds no slot: it only waits on another subject's
	// reading, so a lane full of waiters can never starve the jobs they wait on.
	Inline
)

const (
	commandBudget = 2 * time.Second
	walkBudget    = 10 * time.Second
	coalesce      = 50 * time.Millisecond
	closeGrace    = 2 * time.Second
)

type Job[R any] struct {
	Key    string
	Lane   Lane
	Budget time.Duration
	Do     func(ctx context.Context) func(*R)
	// value: a job whose part fills in steps (one walk, many rows) emits each
	// step; its own answer is the last.
	Partial func(ctx context.Context, emit func(func(*R))) func(*R)
	Late    func(*R)
}

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type Pool struct {
	ctx    context.Context
	cancel context.CancelFunc
	work   sync.WaitGroup
	clock  Clock
	mu     sync.Mutex
	lanes  [2]*lane
	prefer string
	seq    uint64
	shared map[string]any
}

type lane struct {
	free  int
	queue []*ticket
}

type ticket struct {
	key   string
	seq   uint64
	start chan struct{}
}

func NewPool(ctx context.Context, clock Clock, commands, walks int) *Pool {
	ctx, cancel := context.WithCancel(ctx)
	return &Pool{
		ctx:    ctx,
		cancel: cancel,
		clock:  clock,
		lanes:  [2]*lane{{free: commands}, {free: walks}},
		shared: map[string]any{},
	}
}

func (p *Pool) Close() {
	// invariant: Close ends every job and stream of the pool, shared or not, so
	// the commands they run are killed, then waits for them to be reaped (at
	// most closeGrace): station never exits leaving a probe alive behind it.
	p.mu.Lock()
	p.cancel()
	p.mu.Unlock()
	reaped := make(chan struct{})
	go func() {
		p.work.Wait()
		close(reaped)
	}()
	grace, stop := context.WithCancel(context.Background())
	defer stop()
	go func() {
		_ = p.clock.Sleep(grace, closeGrace)
		stop()
	}()
	select {
	case <-reaped:
	case <-grace.Done():
	}
}

// invariant: track holds the pool open for one goroutine of work; once the pool
// is closed it refuses, so Close never waits on work that starts after it.
func (p *Pool) track() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctx.Err() != nil {
		return false
	}
	p.work.Add(1)
	return true
}

func (p *Pool) Prefer(prefix string) {
	// invariant: the preferred prefix's queued jobs start before any other; then
	// every lane keeps the order the jobs were asked in (display order).
	p.mu.Lock()
	p.prefer = prefix
	for _, l := range p.lanes {
		p.order(l)
	}
	p.mu.Unlock()
}

func (p *Pool) Forget(prefix string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key := range p.shared {
		if strings.HasPrefix(key, prefix) {
			delete(p.shared, key)
		}
	}
}

func (p *Pool) order(l *lane) {
	sort.SliceStable(l.queue, func(i, j int) bool {
		a, b := p.preferred(l.queue[i].key), p.preferred(l.queue[j].key)
		if a != b {
			return a
		}
		return l.queue[i].seq < l.queue[j].seq
	})
}

func (p *Pool) preferred(key string) bool { return p.prefer != "" && strings.HasPrefix(key, p.prefer) }

// invariant: enqueue takes the job's place in its lane at once, in the order asked, so
// display order survives the goroutines that wait for the slot.
func (p *Pool) enqueue(which Lane, key string) (wait func(ctx context.Context) (release func(), ok bool)) {
	if which == Inline {
		return func(context.Context) (func(), bool) { return func() {}, true }
	}
	l := p.lanes[which]
	p.mu.Lock()
	p.seq++
	t := &ticket{key: key, seq: p.seq, start: make(chan struct{})}
	l.queue = append(l.queue, t)
	p.order(l)
	p.dispatch(l)
	p.mu.Unlock()
	release := func() {
		p.mu.Lock()
		l.free++
		p.dispatch(l)
		p.mu.Unlock()
	}
	return func(ctx context.Context) (func(), bool) {
		select {
		case <-t.start:
			return release, true
		case <-ctx.Done():
			p.mu.Lock()
			defer p.mu.Unlock()
			select {
			case <-t.start:
				l.free++
				p.dispatch(l)
			default:
				l.queue = slices.DeleteFunc(l.queue, func(q *ticket) bool { return q == t })
			}
			return nil, false
		}
	}
}

func (p *Pool) dispatch(l *lane) {
	for l.free > 0 && len(l.queue) > 0 {
		t := l.queue[0]
		l.queue = l.queue[1:]
		l.free--
		close(t.start)
	}
}

type flight struct {
	done   chan struct{}
	answer any
	late   bool
}

// invariant: a job's budget counts from the moment it holds its slot, never
// from its queueing; past it the job answers Late and its context ends, so the
// command it ran is killed and its slot comes back.
func (p *Pool) execute(ctx context.Context, which Lane, budget time.Duration, wait func(context.Context) (func(), bool), do func(context.Context) any) (answer any, late bool) {
	release, ok := wait(ctx)
	if !ok {
		return nil, true
	}
	if budget == 0 {
		switch which {
		case Commands:
			budget = commandBudget
		case Walks:
			budget = walkBudget
		case Inline:
		}
	}
	if !p.track() {
		release()
		return nil, true
	}
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stopOnClose := context.AfterFunc(p.ctx, cancel)
	answered := make(chan any, 1)
	go func() {
		defer p.work.Done()
		defer stopOnClose()
		defer release()
		answered <- do(jobCtx)
	}()
	expired := make(chan struct{})
	if budget > 0 {
		go func() {
			if p.clock.Sleep(jobCtx, budget) == nil {
				close(expired)
			}
		}()
	}
	select {
	case a := <-answered:
		return a, false
	case <-expired:
		return nil, true
	case <-jobCtx.Done():
		return nil, true
	}
}

// invariant: one key is one execution for the life of the pool (until Forget):
// a second reader of the same key waits for the first one's answer.
func (p *Pool) start(ctx context.Context, which Lane, key string, shared bool, budget time.Duration, do func(context.Context) any) *flight {
	shared = shared && key != ""
	p.mu.Lock()
	if f, ok := p.shared[key].(*flight); ok && shared {
		p.mu.Unlock()
		return f
	}
	f := &flight{done: make(chan struct{})}
	base := ctx
	if shared {
		p.shared[key] = f
		base = p.ctx
	}
	p.mu.Unlock()
	wait := p.enqueue(which, key)
	go func() {
		f.answer, f.late = p.execute(base, which, budget, wait, do)
		close(f.done)
	}()
	return f
}

func (f *flight) await(ctx context.Context) (answer any, late bool) {
	select {
	case <-f.done:
		return f.answer, f.late
	case <-ctx.Done():
		return nil, true
	}
}

func Share[T any](ctx context.Context, p *Pool, key string, fn func(context.Context) T) T {
	// invariant: fn runs once per key for the life of the pool and every caller
	// gets its answer; it holds no lane slot, so the work fn does is its own.
	answer, _ := p.start(ctx, Inline, key, true, 0, func(c context.Context) any { return fn(c) }).await(ctx)
	v, _ := answer.(T)
	return v
}

func Command(name string, args []string) string {
	// value: the key of one command line, so every reader that asks the same
	// words (gh auth status, read by status and by config) shares one run.
	return "cmd/" + name + " " + strings.Join(args, " ")
}

func Shared[O any](ctx context.Context, p *Pool, key string, run func(context.Context) (O, error)) (O, error) {
	type result struct {
		out O
		err error
	}
	r := Share(ctx, p, key, func(ctx context.Context) result {
		out, err := run(ctx)
		return result{out: out, err: err}
	})
	if err := ctx.Err(); err != nil {
		var zero O
		return zero, err
	}
	return r.out, r.err
}

func Run[R any](ctx context.Context, p *Pool, frame R, jobs []Job[R], clone func(R) R) <-chan R {
	out := make(chan R, 1)
	answers := make(chan func(*R), len(jobs))
	partials := make(chan func(*R))
	done := make(chan struct{})
	emit := func(set func(*R)) {
		select {
		case partials <- set:
		case <-done:
		}
	}
	for _, j := range jobs {
		do := func(c context.Context) any { return j.Do(c) }
		if j.Partial != nil {
			do = func(c context.Context) any { return j.Partial(c, emit) }
		}
		f := p.start(ctx, j.Lane, j.Key, j.Partial == nil, j.Budget, do)
		go func() {
			answer, late := f.await(ctx)
			set, ok := answer.(func(*R))
			switch {
			case late || !ok:
				answers <- j.Late
			default:
				answers <- set
			}
		}()
	}
	go func() {
		defer close(done)
		collect(ctx, p.clock, frame, len(jobs), answers, partials, clone, out)
	}()
	return out
}

func collect[R any](ctx context.Context, clock Clock, report R, left int, answers, partials <-chan func(*R), clone func(R) R, out chan<- R) {
	defer close(out)
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	var (
		ready  = clone(report)
		send   = out
		last   = clock.Now()
		wakeUp chan struct{}
		dirty  bool
	)
	for {
		select {
		case <-ctx.Done():
			return
		case send <- ready:
			send, dirty, last = nil, false, clock.Now()
			if left == 0 {
				return
			}
		case set := <-partials:
			set(&report)
			dirty = true
			if send != nil {
				ready = clone(report)
			}
			if send == nil && wakeUp == nil {
				wake := make(chan struct{})
				wakeUp = wake
				wait := max(coalesce-clock.Now().Sub(last), 0)
				go func() {
					if clock.Sleep(ctx, wait) == nil {
						close(wake)
					}
				}()
			}
			continue
		case set := <-answers:
			left--
			if set != nil {
				set(&report)
			}
			dirty = true
			if wait := coalesce - clock.Now().Sub(last); left > 0 && wait > 0 && send == nil {
				if wakeUp == nil {
					wake := make(chan struct{})
					wakeUp = wake
					go func() {
						if clock.Sleep(ctx, wait) == nil {
							close(wake)
						}
					}()
				}
				continue
			}
			ready, send = clone(report), out
		case <-wakeUp:
			wakeUp = nil
			if dirty {
				ready, send = clone(report), out
			}
		}
		if left == 0 && send == nil {
			if !dirty {
				return
			}
			ready, send = clone(report), out
		}
	}
}

func Stream[T any](ctx context.Context, p *Pool, key string, start func(context.Context) <-chan T) <-chan T {
	// invariant: one producer per key for the life of the pool: every reader
	// gets the latest value at once, then each newer one, and its channel
	// closes with the producer's.
	p.mu.Lock()
	h, ok := p.shared[key].(*hub[T])
	if !ok {
		h = &hub[T]{}
		h.cond = sync.NewCond(&h.mu)
		p.shared[key] = h
	}
	p.mu.Unlock()
	switch {
	case ok:
	case p.track():
		source := start(p.ctx)
		go func() {
			defer p.work.Done()
			h.pump(source)
		}()
	default:
		ended := make(chan T)
		close(ended)
		h.pump(ended)
	}
	return h.subscribe(ctx)
}

type hub[T any] struct {
	mu      sync.Mutex
	cond    *sync.Cond
	latest  T
	version int
	closed  bool
}

func (h *hub[T]) pump(source <-chan T) {
	for v := range source {
		h.mu.Lock()
		h.latest = v
		h.version++
		h.mu.Unlock()
		h.cond.Broadcast()
	}
	h.mu.Lock()
	h.closed = true
	h.mu.Unlock()
	h.cond.Broadcast()
}

func (h *hub[T]) subscribe(ctx context.Context) <-chan T {
	out := make(chan T)
	stop := context.AfterFunc(ctx, h.cond.Broadcast)
	go func() {
		defer close(out)
		defer stop()
		seen := 0
		for {
			h.mu.Lock()
			for h.version == seen && !h.closed && ctx.Err() == nil {
				h.cond.Wait()
			}
			if ctx.Err() != nil || (h.version == seen && h.closed) {
				h.mu.Unlock()
				return
			}
			v, version := h.latest, h.version
			h.mu.Unlock()
			select {
			case out <- v:
				seen = version
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func Last[T any](ch <-chan T) T {
	var v T
	for got := range ch {
		v = got
	}
	return v
}
