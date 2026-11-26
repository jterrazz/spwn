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

### A lock left held

A lock a break may leave held is registered with `goes.WatchLock`. After each recovered break each watched lock is tried for 2 s. One still held logs "a lock stayed held after the break: <name>; the engine restarts", and the engine exits with code 70 for launchd to start it afresh. _Owner's to overturn: a held lock restarts the engine, rather than the engine running on degraded and saying so._

### What is logged

One line in the engine log, `<RFC3339> engine: broke in <name> on <call>: <type of the value>: <its text through secret.Kept>`, then the stack. Never the value itself, nor the call's input or output. A break is counted per name and handed to the `OnBreak` hook. Each name is a string literal unique across the engine; the call a boundary served is data, the break's `Of`.

### What the user sees

- **A call the engine serves:** `Error: the engine broke on machines.create-workspace; it kept running` / `Hint: its trace is in ~/.local/state/station/logs/engine.log`, its type `failed`, so exit 1. A process that serves a stream and keeps no log of its own (`guest serve --stdio`, `engine attach`) words it `station broke on <call>; it kept running` / `run it again`.
- **The window and the menu bar:** the gesture fails with those words; a feed that broke ends its watch, and the client reconnects it.
- **A create cut by a panic** leaves its VM stopped and "was left half made", never created. _Owner's to overturn: a create never retries on its own; the owner deletes or starts it. The night's upgrade that broke is run again by the next night._
- **A break that repeats** (one signature 3 times in 10 min) raises a Needs you finding, "The engine keeps breaking on <name>", its hint the log. _Owner's to overturn: a finding, rather than a neutral line on the Engine page only._

### The rule

In the engine's non-test code, no goroutine starts outside `core/goes`: no `go` statement and no `.Go(` call. Each starts through `goes.Go`, `goes.Keep` or `goes.Vital`, named by a string literal unique across the engine. An architecture spec holds it.

### The lots that finish it

- **L1** `core/goes` itself: done.
- **L2** the call boundary, the `broke` code, `world panic` and the create's half-made fix: done.
- **L3a** the fan-out helpers: a job's panic is that job's `broke` error, the other jobs answer.
- **L4** the engine's own: `WatchLock` on the engine's and the act board's locks, `Vital` on the run watch and the accept loop, the night's jobs ending their state in a `defer`.
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
- `broke` is an additive member of the wire's error codes: an older peer shows its message and hint, which travel in the frame, so the state revision stays ([ADR-101](101-a-change-an-older-peer-cannot-ignore-raises-the-state-revision.md)).
- Every new goroutine of the engine starts through `core/goes`, and the architecture spec refuses one that does not, once L3b lands.
