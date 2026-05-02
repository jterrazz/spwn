// Package automation is the trigger-driven agent wakeup engine.
//
// The architect daemon owns the engine — automations only fire while
// architect is running. The engine reads a project's spwn.yaml,
// registers each declared automation, and dispatches a "wake the
// agent with this prompt" call when the trigger fires.
//
// Phase 2 (this package, in flight): cron triggers + receipts +
// catch-up math + serialise-per-agent concurrency.
//
// Phase 3 (next): fs watcher triggers via fsnotify.
//
// # Architecture
//
// The package is organised around four collaborators that the engine
// composes:
//
//	Clock       — wall time abstraction; production = RealClock,
//	              tests = FakeClock you advance manually.
//	Dispatcher  — the "send this prompt to this agent in this world"
//	              callback. Architect-side adapter handles world
//	              auto-spawn + the actual exec into the runtime.
//	Receipts    — append-only log at <project>/.spwn/runs.jsonl,
//	              one line per fire (matches jterrazz-os' prior art).
//	State       — last-fired timestamps, used at startup to compute
//	              missed slots for catch-up.
//
// All four are interfaces. The engine has no Docker, fsnotify, or
// architect imports — it is testable in isolation against fakes.
//
// # Catch-up semantics
//
// On Engine.Start, for each cron automation:
//
//   - Read State.LastFired(world, name).
//   - If zero → no catch-up; just register with the scheduler.
//   - If non-zero → count the cron occurrences strictly after the
//     last fired time and at-or-before clock.Now().
//   - In "collapse" mode (default), fire ONCE with reason="catchup",
//     missed=N, last_fired=prev. Mirrors Apple Reminders semantics:
//     overdue badges collapse, never stack into a notification flood.
//   - In "skip" mode, no fire. Schedule resumes at the next slot.
//
// # Concurrency model
//
//   - One goroutine per registered automation, each waiting on its
//     own clock.After(d) for the next scheduled tick.
//   - Per-agent serialisation: a sync.Mutex keyed on (world, agent)
//     gates Dispatch. Two cron expressions both targeting curie at
//     6am will fire one after the other, never in parallel — agents
//     can't sensibly talk to themselves twice at once.
//   - Cross-agent dispatches in the same tick run in parallel.
package automation

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"spwn.sh/packages/project"
)

// Engine is the in-process trigger scheduler. One Engine per
// architect daemon — Register tracked automations before Start, then
// Stop on shutdown.
type Engine struct {
	clock      Clock
	dispatcher Dispatcher
	receipts   ReceiptWriter
	state      StateStore
	fs         *FSWatcher       // nil if no fs triggers were registered
	commands   CommandResolver  // nil → only prompt: bodies allowed

	cronParser cron.Parser
	registered []*registered

	// agentLocks serialises Dispatch per (world, agent) pair. We use
	// sync.Map only as a lock-free key→mutex registry; the muxes
	// themselves do the actual locking.
	agentLocks sync.Map

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// registered is the engine's view of one automation: the parsed
// trigger, the body, and the cron schedule object built from the
// expression at Register time so we don't re-parse on every tick.
type registered struct {
	World    string
	Name     string
	Auto     project.Automation
	Schedule cron.Schedule // nil for fs (Phase 3)
}

// Config bundles the engine's collaborators. Clock/Dispatcher/
// Receipts/State are required; FS is optional — projects with only
// cron automations can leave it nil. New rejects fs-trigger
// registrations when FS is nil.
//
// Commands resolves `command:` refs to their markdown body. Pass
// nil for in-process tests with prompt-only automations; in
// production the architect supplies a loader that reads
// spwn/commands/<name>.md from the project root.
type Config struct {
	Clock      Clock
	Dispatcher Dispatcher
	Receipts   ReceiptWriter
	State      StateStore
	FS         *FSWatcher
	Commands   CommandResolver
}

// CommandResolver loads the body of a `command/<name>` ref. The
// engine calls Resolve once per fire (no caching) so authors editing
// command files mid-session see updates without restarting the
// architect.
type CommandResolver interface {
	Resolve(ref string) (string, error)
}

// CommandResolverFunc adapts a plain function to CommandResolver.
type CommandResolverFunc func(ref string) (string, error)

// Resolve calls the underlying function.
func (f CommandResolverFunc) Resolve(ref string) (string, error) { return f(ref) }

// New constructs an Engine. Returns an error if any collaborator is
// nil — callers can defer those decisions to a builder, but this
// package keeps the constructor strict so a missing dependency is a
// loud panic at boot rather than a silent no-op at trigger time.
func New(cfg Config) (*Engine, error) {
	if cfg.Clock == nil {
		return nil, fmt.Errorf("automation: Clock is required")
	}
	if cfg.Dispatcher == nil {
		return nil, fmt.Errorf("automation: Dispatcher is required")
	}
	if cfg.Receipts == nil {
		return nil, fmt.Errorf("automation: Receipts is required")
	}
	if cfg.State == nil {
		return nil, fmt.Errorf("automation: State is required")
	}
	return &Engine{
		clock:      cfg.Clock,
		dispatcher: cfg.Dispatcher,
		receipts:   cfg.Receipts,
		state:      cfg.State,
		fs:         cfg.FS,
		commands:   cfg.Commands,
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}, nil
}

// Register adds a world's automations to the engine. Must be called
// before Start. fs triggers are tolerated but ignored in Phase 2 —
// they'll be wired in Phase 3 without API changes here.
func (e *Engine) Register(world string, autos map[string]project.Automation) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return fmt.Errorf("automation: Register called after Start")
	}
	for name, a := range autos {
		r := &registered{World: world, Name: name, Auto: a}
		switch {
		case a.On.Cron != "":
			sched, err := e.cronParser.Parse(a.On.Cron)
			if err != nil {
				return fmt.Errorf("automation %s/%s: invalid cron %q: %w", world, name, a.On.Cron, err)
			}
			r.Schedule = sched
		case a.On.FS != nil:
			if e.fs == nil {
				return fmt.Errorf("automation %s/%s: fs trigger registered but engine Config.FS is nil", world, name)
			}
			// Capture r in the closure so the handler routes to the
			// right registered{} when its fired.
			rr := r
			spec := FSWatchSpec{
				ID:        stateKey(world, name),
				Path:      a.On.FS.Path,
				Events:    a.On.FS.Events,
				Recursive: a.On.FS.Recursive,
				Patterns:  a.On.FS.Patterns,
				Debounce:  a.On.FS.Debounce.AsDuration(),
			}
			handler := func(ev DebouncedFSEvent) {
				e.fire(context.Background(), rr, FireSource{
					Kind:       "fs",
					Reason:     ev.Kind + ":" + filepath.Base(ev.Paths[0]),
					Now:        e.clock.Now(),
					EventPaths: ev.Paths,
					EventKind:  ev.Kind,
				})
			}
			if err := e.fs.Watch(spec, handler); err != nil {
				return fmt.Errorf("automation %s/%s: fs watch: %w", world, name, err)
			}
		default:
			// Validation already rejects this; defence-in-depth.
			return fmt.Errorf("automation %s/%s: no trigger (validation should have caught this)", world, name)
		}
		e.registered = append(e.registered, r)
	}
	// Stable order for tests + receipt determinism.
	sort.Slice(e.registered, func(i, j int) bool {
		if e.registered[i].World != e.registered[j].World {
			return e.registered[i].World < e.registered[j].World
		}
		return e.registered[i].Name < e.registered[j].Name
	})
	return nil
}

// Start performs catch-up for every cron automation, then launches
// one scheduler goroutine per registered cron. Returns immediately —
// goroutines run in the background. Call Stop to drain them.
//
// Calling Start twice is a programming error and returns an error.
// Catch-up errors do NOT abort Start: a single automation with an
// unreachable Dispatcher shouldn't keep the rest of the project's
// automations from running.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return fmt.Errorf("automation: Start called twice")
	}
	e.started = true
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.mu.Unlock()

	for _, r := range e.registered {
		if r.Schedule == nil {
			continue
		}
		e.runCatchUp(ctx, r)
		e.wg.Add(1)
		go e.runCron(ctx, r)
	}
	// Fs watcher pump runs only if fs triggers were registered.
	// Idempotent: harmless to call when no specs exist.
	if e.fs != nil {
		if err := e.fs.Start(ctx); err != nil {
			return fmt.Errorf("automation: fs Start: %w", err)
		}
	}
	return nil
}

// Stop cancels the engine's context and waits for every scheduler
// goroutine to drain. Safe to call before Start (no-op).
//
// The fs watcher's Close() is called too — its pump goroutine and
// any in-flight debounce timers exit before Stop returns.
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return
	}
	cancel := e.cancel
	fs := e.fs
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if fs != nil {
		_ = fs.Close()
	}
	e.wg.Wait()
}

// runCron is the per-automation loop: compute the next scheduled
// time, sleep until then via the Clock (so fakes can advance), fire,
// repeat. Exits when the context is cancelled.
func (e *Engine) runCron(ctx context.Context, r *registered) {
	defer e.wg.Done()
	for {
		now := e.clock.Now()
		next := r.Schedule.Next(now)
		d := next.Sub(now)
		if d < 0 {
			// Schedule.Next is documented to never return a time
			// before its argument; guard anyway so a bug elsewhere
			// can't spin the goroutine.
			d = 0
		}
		select {
		case <-ctx.Done():
			return
		case <-e.clock.After(d):
			e.fire(ctx, r, FireSource{
				Kind:      "cron",
				Reason:    "on-time",
				Scheduled: next,
				Now:       e.clock.Now(),
			})
		}
	}
}

// runCatchUp inspects the state store and, if the automation has been
// fired before, counts the cron slots missed since. Emits exactly one
// "catchup" fire (collapse mode) or zero (skip mode). Schedules
// resume normally afterwards via runCron.
//
// Called synchronously from Start so a caller that immediately
// inspects receipts after Start sees the catch-up entry already
// committed. Tests rely on this ordering.
func (e *Engine) runCatchUp(ctx context.Context, r *registered) {
	if r.Auto.Catchup == "skip" {
		return
	}
	last, ok := e.state.LastFired(r.World, r.Name)
	if !ok {
		// First-ever boot — never fire catch-up. The engine has no
		// "what ought to have happened before I existed" view.
		return
	}
	now := e.clock.Now()
	missed := countMissed(r.Schedule, last, now)
	if missed == 0 {
		return
	}
	e.fire(ctx, r, FireSource{
		Kind:      "cron",
		Reason:    "catchup",
		Scheduled: lastScheduled(r.Schedule, last, now),
		Now:       now,
		Missed:    missed,
		LastFired: last,
	})
}

// fire is the dispatch path shared by on-time + catch-up cron fires
// and fs events. Renders the prompt, serialises on the per-agent
// lock, dispatches, writes a receipt.
func (e *Engine) fire(ctx context.Context, r *registered, src FireSource) {
	body := r.Auto.Prompt
	if body == "" && r.Auto.Command != "" {
		if e.commands == nil {
			e.failFire(r, src, fmt.Errorf("command ref %q used but no CommandResolver configured", r.Auto.Command))
			return
		}
		resolved, err := e.commands.Resolve(r.Auto.Command)
		if err != nil {
			e.failFire(r, src, fmt.Errorf("resolve %s: %w", r.Auto.Command, err))
			return
		}
		body = resolved
	}
	prompt, renderErr := renderPrompt(body, "", src)

	mu := e.lockForAgent(r.World, r.Auto.Agent)
	mu.Lock()
	defer mu.Unlock()

	rec := Receipt{
		World:      r.World,
		Automation: r.Name,
		Trigger:    src.Kind,
		Scheduled:  src.Scheduled,
		Fired:      e.clock.Now(),
		Reason:     src.Reason,
	}
	if src.Missed > 0 {
		rec.Missed = src.Missed
		rec.LastFired = src.LastFired
	}

	if renderErr != nil {
		rec.Finished = e.clock.Now()
		rec.OK = false
		rec.Error = "render: " + renderErr.Error()
		_ = e.receipts.Write(rec)
		return
	}

	dispatchErr := e.dispatcher.Dispatch(ctx, DispatchRequest{
		World:  r.World,
		Agent:  r.Auto.Agent,
		Prompt: prompt,
		Source: src,
	})
	rec.Finished = e.clock.Now()
	rec.DurationMS = rec.Finished.Sub(rec.Fired).Milliseconds()
	if dispatchErr != nil {
		rec.OK = false
		rec.Error = dispatchErr.Error()
	} else {
		rec.OK = true
		// Record success so the next catch-up math has a fresh
		// anchor. We record the SCHEDULED time, not Fired — a
		// catch-up that ran 2h late should advance the cursor to
		// the slot it covered, not to "now".
		_ = e.state.RecordFire(r.World, r.Name, src.Scheduled)
	}
	_ = e.receipts.Write(rec)
}

// failFire commits a not-OK receipt without ever calling Dispatch.
// Used when prompt rendering / command resolution fails — the
// dispatcher would have nothing to send anyway.
func (e *Engine) failFire(r *registered, src FireSource, cause error) {
	now := e.clock.Now()
	rec := Receipt{
		World:      r.World,
		Automation: r.Name,
		Trigger:    src.Kind,
		Scheduled:  src.Scheduled,
		Fired:      now,
		Finished:   now,
		Reason:     src.Reason,
		OK:         false,
		Error:      cause.Error(),
	}
	if src.Missed > 0 {
		rec.Missed = src.Missed
		rec.LastFired = src.LastFired
	}
	_ = e.receipts.Write(rec)
}

// lockForAgent returns the mutex serialising Dispatch for a given
// (world, agent) pair. Lazy-allocated.
func (e *Engine) lockForAgent(world, agent string) *sync.Mutex {
	key := world + "/" + agent
	if m, ok := e.agentLocks.Load(key); ok {
		return m.(*sync.Mutex)
	}
	m, _ := e.agentLocks.LoadOrStore(key, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// countMissed returns the number of cron occurrences strictly after
// `last` and at-or-before `now`. The exclusive-inclusive boundary is
// intentional: `last` is the time of the previous successful fire,
// not a missed slot itself.
func countMissed(schedule cron.Schedule, last, now time.Time) int {
	count := 0
	t := schedule.Next(last)
	for !t.After(now) {
		count++
		t = schedule.Next(t)
		// Defensive: prevent runaway if Next ever returns a non-
		// monotonic time. In practice robfig/cron is well-behaved.
		if count > 100_000 {
			return count
		}
	}
	return count
}

// lastScheduled returns the most recent scheduled occurrence in
// (last, now]. Used as the receipt's "scheduled" field for catch-up
// fires so the dashboard renders the slot the fire covered, not a
// time that doesn't match the cron grid.
func lastScheduled(schedule cron.Schedule, last, now time.Time) time.Time {
	prev := time.Time{}
	t := schedule.Next(last)
	for !t.After(now) {
		prev = t
		t = schedule.Next(t)
		if prev.Year() > 9000 {
			break
		}
	}
	return prev
}
