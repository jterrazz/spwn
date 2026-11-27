# ADR-108: A panic breaks its call or its job, never the engine

**Status:** Proposed
**Date:** 2026-11-26

The engine is one process that serves every surface, runs every VM's watch and keeps the store the pages are drawn from. Before this record, a panic in any goroutine of it killed it whole. The choice below was taken by the owner's delegation on 2026-11-26, on the recommendations of the engine-panic plan. Three of its choices are the owner's to overturn, and each is marked where it is decided. The record waits for his acceptance.

## Context

A Go panic that no goroutine recovers ends the process. The engine starts some 127 goroutines (its tick, its accept loop, a session per VM, the readers of its store, the night's jobs, the fan-out of a call over every Mac), and it serves calls whose input a peer controls: a guest, another Mac, the window. Launchd starts a dead engine afresh, the VMs keep running, and adoption rebuilds what it knew. But every break then drops every watch, every create and build in flight, the activity history held in memory, and up to 10 s of supervision. A panic a peer can provoke becomes a crash loop.

## Decision

Mix D of the plan: every goroutine root and every call boundary recovers through one helper, `core/goes`, which names it, logs it and fails only that call or that job.

### The three policies

- **A call** the engine serves runs under `goes.Try` (`TryOf` when it names the call it ran). A panic there answers that call with the coded error `broke`, and the connection answers its next frame. A watch whose feed panics ends with the same error, and its client reconnects as it does after any ended watch.
- **A job** runs under `goes.Go`: a one-shot whose break is reported and ends it, its own defers run. A loop runs under `goes.Keep`: run again after 1 s, doubling to 5 min, and from 1 s again after a run of 10 min.
- **A vital goroutine** runs under `goes.Vital`: the run watch of a VM (a lost one never says the VM ended, so the engine would believe it runs forever) and the accept loop (an engine that accepts nothing is a zombie). Its break is reported, then panicked again, and launchd starts the engine afresh.

### What is never recovered

- A value that implements `goes.Fatal` is panicked again by every helper: the spec build's secret tripwire (`secret.Tripped`) and the strict page seal (`page.Unsealed`).
- In the spec build (`-tags world`), a value that is not `goes.Scripted` is panicked again, so a story still dies on a real bug. A test or a `world panic` fault breaks on purpose with a `goes.Script` value.
- A report that itself panics (the log, the redactor, the hook) ends the process with a `Fatal` naming the break it lost.

### What L2 covers, and what it does not yet

L2 covers the call boundary: every frame `remote/serve` serves runs its handler under `TryOf`, the setup and the send loop of a watch included. It does not cover what runs on a goroutine of its own, where a panic still ends the engine: the registry's watch producers (`core/call/call.go`), the socket's connection goroutine outside `TryOf` (its repair, the act board's end, the journal's keep, the far invoke), the fan-out helpers and the page feeds (L3a, L3b), and the background jobs and the per-VM supervision (L4, L3b).

### A lock left held

A recovered panic under a lock released without `defer` would leave that lock held, and the engine hung where it used to die and be started afresh. Lot L2b releases by `defer` every lock of the engine's non-test code that runs a call, an index or a channel operation under it, and every fake of the world that calls `Fails` under the world's lock; what stays undeferred reads or writes fields alone.

L4 is to watch a lock left held all the same, by this design, and `goes.WatchLock` is not wired to any lock until then. It watches only a lock never held across I/O: the engine's `d.mu` is held across tailscale calls of up to 5 s each today, and `sync.Mutex.TryLock` also fails while the mutex starves, so a 2 s try would restart a healthy engine on any unrelated break. A watched lock counts its acquisitions, and the engine restarts only when the SAME acquisition is still held after T ≥ 30 s; a test proves first that a healthy long hold beside an unrelated break restarts nothing. The restart is an exit with code 70, which launchd's `KeepAlive` starts afresh: after the throttle when the job ran under 10 s, otherwise at once, then the engine's own start. _Owner's to overturn: a held lock restarts the engine, rather than the engine running on degraded and saying so._

### What is logged

One line in the engine log, `<RFC3339> engine: broke in <name> on <call>: <type of the value>: <its text through secret.Kept>`, then the stack. The text is kept only when the value is an error (a `runtime.Error` included) or a string; any other value, a struct or a map that may hold a secret or a peer's input, is told by its type alone. Never the value itself, nor the call's input or output. The stack shows each frame's argument words in hex: a string or a slice as its pointer and its length, never its bytes; a small integer or an array passed by value as its value. A break is counted per name and handed to the `OnBreak` hook. Each name is a string literal unique across the engine; the call a boundary served is data, the break's `Of`. A process that serves a stream, `guest serve --stdio` or `engine attach`, appends the same line to its own machine's engine log, since the side that runs it keeps none of its stderr.

### What the user sees

- **A call the engine serves:** `Error: the engine broke on machines.create-workspace; it kept running` / `Hint: its trace is in ~/.local/state/station/logs/engine.log`, its type `failed`, so exit 1. A process that serves a stream (`guest serve --stdio`, `engine attach`) words it `station broke on <call>; it kept running` / `run it again`.
- **The window and the menu bar:** the gesture fails with those words; a feed that broke ends its watch, and the client reconnects it.
- **A create cut by a panic** leaves its VM stopped and "was left half made", never created. _Owner's to overturn: a create never retries on its own; the owner deletes or starts it. The night's upgrade that broke is run again by the next night._
- **A break that repeats** is to raise, with L5, a Needs you finding, "The engine keeps breaking on <name>", its hint the log, once one signature broke 3 times in 10 min, and to back the feed off. Until L5, a page projection that breaks on every store revision is reconnected about every second and writes a full stack each time, into a log no rotation bounds. _Owner's to overturn: a finding, rather than a neutral line on the Engine page only._

### The rule

In the engine's non-test code, no goroutine is to start outside `core/goes`: no `go` statement and no `.Go(` call. Each is to start through `goes.Go`, `goes.Keep` or `goes.Vital`, named by a string literal unique across the engine. An architecture spec is to hold it once L3b lands; none does yet.

### The lots that finish it

- **L1** `core/goes` itself: done.
- **L2** the call boundary, the `broke` code, `world panic` and the create's half-made fix: done.
- **L2b** every lock under which a recovered panic could be left held, released by `defer`; the break's text kept for words alone; the stream processes' breaks in their engine log: done.
- **L3a** the fan-out helpers: a job's panic is that job's `broke` error, the other jobs answer.
- **L4** the engine's own: the lock watch of "A lock left held", `Vital` on the run watch and the accept loop, the night's jobs ending their state in a `defer`.
- **L3b** every other site, and the architecture spec of the rule. It lands last.
- **L5** the repeat finding.
- **L7** the untrusted input made total, fuzzed where it is parsed.

## Alternatives

- **A, recover at the call boundary only.** It covers only the synchronous part of a handler, and it adds the held-lock hazard: a hung engine is worse than a dead one, since launchd restarts a dead one and nothing restarts a hung one.
- **B, crash-only, made cheap.** Half of it is built: VMs survive, adoption, half-made births, interrupted acts. But every break drops every watch and every act in flight, and a panic a peer provokes becomes a crash loop.
- **C, the untrusted entry points made total and fuzzed, crash-only elsewhere.** Necessary, not sufficient: it cannot be complete, and any miss is a crash loop. It is kept, as L7.

## Consequences

- A bug in one call or one job costs that call or that job, and a line in the log; the engine, its watches and its acts keep running.
- A call the CLI runs in its own process is outside this record: its panic ends that command, and the engine, a process apart, is untouched.
- `broke` is a new member of the wire's closed enum of error codes, which [the API reference](../reference/api.md) classes as breaking: a member added to an enum raises the revision. It enters revision 2 without raising it only because no release carries revision 2 yet, and a revision no release carries takes a breaking change in ([ADR-101](101-a-change-an-older-peer-cannot-ignore-raises-the-state-revision.md)). An older peer that met it would still show its message and hint, which travel in the frame.
- Every new goroutine of the engine starts through `core/goes`, and the architecture spec refuses one that does not, once L3b lands.
