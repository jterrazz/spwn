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
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"spwn.sh/packages/project"
)

// promptSHA returns the first 12 hex chars of sha256(prompt). 48
// bits of identifier is plenty for "did the prompt change between
// fires?" debugging without storing the full body in every receipt.
func promptSHA(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])[:12]
}

// newRunID returns a 16-hex-char unique identifier for one fire.
// Used as the receipt's run_id so dashboards can trace a fire across
// the receipt log + structured-logger output. crypto/rand for
// uniqueness; 64 bits of entropy is overkill for per-day fire
// counts but cheap.
//
// On the off chance crypto/rand fails (kernel entropy starvation
// on a freshly-booted VM), fall back to a clock-derived id so the
// fire path never fails — receipts are observability, not
// correctness.
func newRunID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback: nanoseconds since epoch in hex. Not unique
		// across processes but never repeats within one process.
		now := time.Now().UnixNano()
		for i := 7; i >= 0; i-- {
			b[i] = byte(now)
			now >>= 8
		}
	}
	return hex.EncodeToString(b[:])
}

// Logger is the engine's minimal sink for non-fatal errors that
// would otherwise be swallowed (receipt write fails, state cursor
// fails, fsnotify watcher errors). The default writes a "spwn:
// automation:" prefixed line to stderr; production callers can
// inject a structured logger that routes to journald / etc.
type Logger interface {
	Warnf(format string, args ...any)
}

// stderrLogger is the default Logger. Wraps the standard log package
// with a fixed prefix so a single grep ("spwn: automation:") finds
// every dropped-error line in journalctl / `tail -f /var/log`.
type stderrLogger struct{ l *log.Logger }

func (s stderrLogger) Warnf(format string, args ...any) {
	s.l.Printf("spwn: automation: "+format, args...)
}

// defaultLogger writes to os.Stderr and is used when Config.Logger
// is nil. Constructed lazily in New so tests that inject a Logger
// don't see stray output.
func defaultLogger() Logger {
	return stderrLogger{l: log.New(os.Stderr, "", log.LstdFlags|log.Lmsgprefix)}
}

// Engine is the in-process trigger scheduler. One Engine per
// architect daemon — Register tracked automations before Start, then
// Stop on shutdown.
type Engine struct {
	clock      Clock
	dispatcher Dispatcher
	receipts   ReceiptWriter
	state      StateStore
	fs         *FSWatcher      // nil if no fs triggers were registered
	commands   CommandResolver // nil → only prompt: bodies allowed
	logger     Logger          // never nil; defaults to stderr

	cronParser cron.Parser
	registered []*registered

	// agentLocks serialises Dispatch per (world, agent) pair. We use
	// sync.Map only as a lock-free key→mutex registry; the muxes
	// themselves do the actual locking.
	agentLocks sync.Map

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	// runCtx is the cancellable context handed out to per-trigger
	// goroutines (cron loops and fs handlers). Set in Start, cancelled
	// in Stop. Nil before Start; readers should check.
	runCtx context.Context
	wg     sync.WaitGroup
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
	// Logger receives non-fatal errors that would otherwise be
	// swallowed (receipt write failure, state cursor write failure).
	// Defaults to a stderr logger when nil.
	Logger Logger
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
	logger := cfg.Logger
	if logger == nil {
		logger = defaultLogger()
	}
	return &Engine{
		clock:      cfg.Clock,
		dispatcher: cfg.Dispatcher,
		receipts:   cfg.Receipts,
		state:      cfg.State,
		fs:         cfg.FS,
		commands:   cfg.Commands,
		logger:     logger,
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
				ID:            stateKey(world, name),
				Path:          a.On.FS.Path,
				Events:        a.On.FS.Events,
				Recursive:     a.On.FS.Recursive,
				Patterns:      a.On.FS.Patterns,
				Debounce:      a.On.FS.Debounce.AsDuration(),
				IncludeHidden: a.On.FS.IncludeHidden,
			}
			handler := func(ev DebouncedFSEvent) {
				// Capture the engine's runCtx at fire time so an
				// in-flight Dispatch is interruptible by Stop. Until
				// Start runs, runCtx is nil — and the watcher's pump
				// also doesn't dispatch until Start, so the nil
				// branch is unreachable in normal flow. Defensive
				// fallback to Background keeps a misuse safe.
				ctx := e.runCtx
				if ctx == nil {
					ctx = context.Background()
				}
				now := e.clock.Now()
				e.fire(ctx, rr, FireSource{
					Kind: "fs",
					Reason: ev.Kind + ":" + filepath.Base(ev.Paths[0]),
					Now:  now,
					// Scheduled = Now for fs fires so the state
					// cursor advances. The replay-on-startup logic
					// uses this cursor to decide which files are
					// "new" since the last fire; a zero cursor would
					// match every file forever.
					Scheduled:  now,
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
	e.runCtx = ctx
	e.mu.Unlock()

	for _, r := range e.registered {
		if r.Schedule == nil {
			continue
		}
		e.runCatchUp(ctx, r)
		e.wg.Add(1)
		go e.runCron(ctx, r)
	}
	// Fs replay-on-startup: synthesise create events for files
	// whose mtime is newer than the last successful fire. Mirrors
	// cron's catch-up so a daemon-down window doesn't lose files
	// dropped into the inbox during the gap.
	for _, r := range e.registered {
		if r.Auto.On.FS == nil {
			continue
		}
		if r.Auto.Catchup == "skip" {
			continue
		}
		e.runFSReplay(ctx, r)
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

// catchUpStackCap caps how many missed slots a `catchup: stack`
// automation will fire on resume. Without it, a 7-day downtime of a
// minutely cron would dispatch 10080 receipts in a row, hammering
// the agent and the receipt log. 100 is enough to express
// "yesterday's queued work is processed individually" without
// blast-radius pathologies.
const catchUpStackCap = 100

// runCatchUp inspects the state store and, if the automation has been
// fired before, fires the catch-up dispatch(es). Three modes:
//
//   - collapse (default): one fire on resume regardless of slot count
//   - skip:               no fires on resume
//   - stack:              one fire per missed slot, capped at
//                         catchUpStackCap
//
// Called synchronously from Start so a caller that immediately
// inspects receipts after Start sees the catch-up entries already
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

	if r.Auto.Catchup == "stack" {
		// One fire per missed slot. Walk the schedule from `last`
		// forward, firing for each occurrence ≤ now, up to the cap.
		// Each fire records the slot it covered as Scheduled so a
		// catch-up that ran 1h late for the 06:00 slot still
		// receipts as "scheduled: 06:00".
		count := 0
		t := r.Schedule.Next(last)
		for !t.After(now) {
			if count >= catchUpStackCap {
				e.logger.Warnf("catchup stack cap reached for %s/%s after %d fires; remaining missed slots dropped", r.World, r.Name, count)
				break
			}
			e.fire(ctx, r, FireSource{
				Kind:      "cron",
				Reason:    "catchup",
				Scheduled: t,
				Now:       e.clock.Now(),
				LastFired: last,
				// Missed=1 marks each row in stack mode as a single
				// missed-slot replay, distinct from collapse mode's
				// rolled-up Missed=N.
				Missed: 1,
			})
			count++
			t = r.Schedule.Next(t)
		}
		return
	}

	// collapse mode (default).
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

// runFSReplay synthesises create events for files newer than the
// last successful fire. Bridges the gap between architect-down
// windows (when fsnotify isn't running) and "files dropped into
// inbox during downtime should be processed".
//
// Walks the watched directory, filters by the spec's pattern +
// events allow-list, and if any matches exist with mtime > last,
// fires once with the full path list as the burst payload. Same
// debounce-coalesces semantics as a real fsnotify burst — the
// agent sees one "burst of N files" prompt, not N separate fires.
//
// Skipped on first boot (no last-fired cursor → no comparison
// baseline). Skipped when catchup == "skip" — matches cron's
// catchup-skip semantics so users have one knob for both.
//
// Errors during the walk are logged but don't abort: a permission-
// denied subdirectory shouldn't block replay of the rest. The
// engine receipts the synthesised fire normally.
func (e *Engine) runFSReplay(ctx context.Context, r *registered) {
	last, ok := e.state.LastFired(r.World, r.Name)
	if !ok {
		return // first boot — nothing to compare against
	}

	spec := r.Auto.On.FS
	if spec == nil {
		return // defensive — caller should have filtered
	}

	// The replay synthesises create events. If the spec doesn't
	// allow Create, skip — there's no other event we can sensibly
	// reconstruct from a static directory snapshot.
	allowsCreate := false
	events := spec.Events
	if len(events) == 0 {
		events = []string{"create"} // mirror ApplyDefaults
	}
	for _, ev := range events {
		if ev == "create" {
			allowsCreate = true
			break
		}
	}
	if !allowsCreate {
		return
	}

	var paths []string
	collect := func(path string, info os.FileInfo) {
		if info.IsDir() {
			return
		}
		if !info.ModTime().After(last) {
			return
		}
		if !matchPatterns(spec.Patterns, filepath.Base(path)) {
			return
		}
		paths = append(paths, path)
	}

	if spec.Recursive {
		err := filepath.Walk(spec.Path, func(p string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				// Permission denied / vanished directory. Log and
				// skip — don't abort the whole replay.
				e.logger.Warnf("fs replay walk %s: %v", p, walkErr)
				return nil
			}
			collect(p, info)
			return nil
		})
		if err != nil {
			e.logger.Warnf("fs replay %s/%s walk failed: %v", r.World, r.Name, err)
			return
		}
	} else {
		entries, err := os.ReadDir(spec.Path)
		if err != nil {
			e.logger.Warnf("fs replay %s/%s readdir %s: %v", r.World, r.Name, spec.Path, err)
			return
		}
		for _, entry := range entries {
			info, ierr := entry.Info()
			if ierr != nil {
				continue
			}
			collect(filepath.Join(spec.Path, entry.Name()), info)
		}
	}

	if len(paths) == 0 {
		return
	}
	sort.Strings(paths)

	now := e.clock.Now()
	e.fire(ctx, r, FireSource{
		Kind:       "fs",
		Reason:     "replay:" + filepath.Base(paths[0]),
		Now:        now,
		Scheduled:  now,
		EventPaths: paths,
		EventKind:  "create",
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

	// EnqueuedAt: stamped BEFORE the per-agent lock so dashboards
	// can render lock-wait time = Fired - EnqueuedAt. Otherwise a
	// Fired stamp set inside the critical section conflates queue
	// time with runtime time.
	enqueuedAt := e.clock.Now()
	mu := e.lockForAgent(r.World, r.Auto.Agent)
	mu.Lock()
	defer mu.Unlock()

	rec := Receipt{
		World:         r.World,
		Automation:    r.Name,
		Agent:         r.Auto.Agent,
		Trigger:       src.Kind,
		RunID:         newRunID(),
		EngineVersion: EngineVersion,
		Scheduled:     src.Scheduled,
		EnqueuedAt:    enqueuedAt,
		Fired:         e.clock.Now(),
		Reason:        src.Reason,
	}
	if src.Missed > 0 {
		rec.Missed = src.Missed
		rec.LastFired = src.LastFired
	}
	if src.Kind == "fs" {
		rec.EventPaths = append([]string(nil), src.EventPaths...)
		rec.EventKind = src.EventKind
	}

	if renderErr != nil {
		rec.Finished = e.clock.Now()
		rec.OK = false
		rec.Error = "render: " + renderErr.Error()
		if werr := e.receipts.Write(rec); werr != nil {
			e.logger.Warnf("receipt write failed for %s/%s (render error): %v", r.World, r.Name, werr)
		}
		return
	}
	rec.PromptSHA = promptSHA(prompt)

	result := e.dispatcher.Dispatch(ctx, DispatchRequest{
		World:  r.World,
		Agent:  r.Auto.Agent,
		Prompt: prompt,
		Source: src,
	})
	rec.Finished = e.clock.Now()
	rec.DurationMS = rec.Finished.Sub(rec.Fired).Milliseconds()
	rec.Output = truncateOutput(result.Output)
	if result.Err != nil {
		rec.OK = false
		rec.Error = result.Err.Error()
	} else {
		rec.OK = true
		// Record success so the next catch-up math has a fresh
		// anchor. We record the SCHEDULED time, not Fired — a
		// catch-up that ran 2h late should advance the cursor to
		// the slot it covered, not to "now".
		if serr := e.state.RecordFire(r.World, r.Name, src.Scheduled); serr != nil {
			// State write failure is non-fatal: the next catch-up
			// will re-detect the missed slot and re-fire (collapse
			// mode) or skip (skip mode). Surface the error so an
			// operator with a tainted state file or a disk-full
			// condition has a trail.
			e.logger.Warnf("state cursor write failed for %s/%s: %v", r.World, r.Name, serr)
		}
	}
	if werr := e.receipts.Write(rec); werr != nil {
		e.logger.Warnf("receipt write failed for %s/%s: %v", r.World, r.Name, werr)
	}
}

// failFire commits a not-OK receipt without ever calling Dispatch.
// Used when prompt rendering / command resolution fails — the
// dispatcher would have nothing to send anyway.
func (e *Engine) failFire(r *registered, src FireSource, cause error) {
	now := e.clock.Now()
	rec := Receipt{
		World:         r.World,
		Automation:    r.Name,
		Agent:         r.Auto.Agent,
		Trigger:       src.Kind,
		RunID:         newRunID(),
		EngineVersion: EngineVersion,
		Scheduled:     src.Scheduled,
		Fired:         now,
		Finished:      now,
		Reason:        src.Reason,
		OK:            false,
		Error:         cause.Error(),
	}
	if src.Missed > 0 {
		rec.Missed = src.Missed
		rec.LastFired = src.LastFired
	}
	if src.Kind == "fs" {
		rec.EventPaths = append([]string(nil), src.EventPaths...)
		rec.EventKind = src.EventKind
	}
	if werr := e.receipts.Write(rec); werr != nil {
		e.logger.Warnf("receipt write failed for %s/%s (failFire %s): %v", r.World, r.Name, cause, werr)
	}
}

// agentLockKey is the registry key for a (world, agent) pair. We use
// a struct rather than `world+"/"+agent` because the slug regex
// allows `-` but not `/` per validation, however a programmatic
// caller (or a future scheme that does allow `/`) could collide
// `("foo/bar","baz")` with `("foo","bar/baz")` and silently
// serialise unrelated agents on a single mutex.
type agentLockKey struct {
	world string
	agent string
}

// lockForAgent returns the mutex serialising Dispatch for a given
// (world, agent) pair. Lazy-allocated.
func (e *Engine) lockForAgent(world, agent string) *sync.Mutex {
	key := agentLockKey{world: world, agent: agent}
	if m, ok := e.agentLocks.Load(key); ok {
		return m.(*sync.Mutex)
	}
	m, _ := e.agentLocks.LoadOrStore(key, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// catchUpIterCap bounds catch-up math iterations across both
// countMissed and lastScheduled. 10k slots covers seven days of
// minutely cron (most-frequent realistic cadence × longest realistic
// daemon-down window without the user noticing). Hand-edited or
// corrupt state cursors that produce millions of slots would
// otherwise block Engine.Start; the cap keeps boot bounded at ~6.5ms
// per registered cron in pathological cases.
const catchUpIterCap = 10_000

// countMissed returns the number of cron occurrences strictly after
// `last` and at-or-before `now`. The exclusive-inclusive boundary is
// intentional: `last` is the time of the previous successful fire,
// not a missed slot itself.
//
// Capped at catchUpIterCap iterations. Hitting the cap means catch-up
// "saw" 10000 missed slots — the user should treat this as "many"
// rather than a precise count.
func countMissed(schedule cron.Schedule, last, now time.Time) int {
	count := 0
	t := schedule.Next(last)
	for !t.After(now) {
		count++
		t = schedule.Next(t)
		if count >= catchUpIterCap {
			return count
		}
	}
	return count
}

// lastScheduled returns the most recent scheduled occurrence in
// (last, now]. Used as the receipt's "scheduled" field for catch-up
// fires so the dashboard renders the slot the fire covered, not a
// time that doesn't match the cron grid.
//
// Capped at catchUpIterCap iterations. Mirrors countMissed; in the
// cap-hit case the returned time is the catchUpIterCap'th slot after
// `last`, not the truly-most-recent slot. The user trades precision
// for bounded boot time.
func lastScheduled(schedule cron.Schedule, last, now time.Time) time.Time {
	prev := time.Time{}
	t := schedule.Next(last)
	count := 0
	for !t.After(now) {
		prev = t
		t = schedule.Next(t)
		count++
		if count >= catchUpIterCap {
			break
		}
	}
	return prev
}
