# ADR-054: A VM's standing is one rule, and every verdict reads it

**Status:** Proposed
**Date:** 2026-11-18

## Context

A VM's status was said in four places by three predicates. `HeadOf` judged down with a private `down()`: the public
`Down()` (a server in a loop, powered off from inside, or stopped while wanted running), any VM in a loop or ended
unexpectedly, and a server running whose check fails. `Down()`, which the menu bar's rows, badge, needs and finder
read, held no failing check. `HealthVerdict` said a failing check in a branch of its own, `Its check fails.`. A client
worded the raw `State` (`MachinesVMState`, `Running`, tone ok).

So the engine disagreed with itself. In the `stories` recording, infra is `State: running` with its check failing, and
its `Head` says `infra is down.` (`Its check fails since 10:18.`); the menu bar drew its row green, the window wrote
`Running`, and the CLI printed `◐ infra ... Running · always on ... check failing since 10:18`.

## Decision

1. **A VM has one standing, a closed list, the first rule that matches:** `not-seen` (its Mac does not answer),
   `restarting` (a restart walks), `creating`, `down`, `stopped`, `suspended`, `healthy` (a server running that passes
   every step), `running`. `StandingOf` in `apps/engine/machines/standing.go` owns it, and `machines.list` carries it
   as `VM.Standing` (`MachinesVMStanding`).
2. **Down is one rule.** `down` is a VM in a loop or ended unexpectedly; a server in a loop, powered off from inside,
   or stopped while its record wants it running; a server running whose check fails. `Down()` is
   `StandingOf(vm) == down` and owns no clause: the private `down()` and Health's check-failing branch are deleted, so
   a failing check reads `Down.` with the line its head draws.
3. **Head is derived from the standing.** `HeadOf` opens on a switch over it, with one case per value and no default,
   so the `exhaustive` lint fails a value left out. Each value's tone is its word's tone; only `running` and `healthy`
   go on to say how the VM stands with its file, and may turn `warn` there.
4. **Words and tones live once**, on the type (`Word()`, `Tone()`), and `make generate` writes them to
   `packages/api/words.json` lowercase: `not seen`, `restarting`, `creating`, `down`, `stopped`, `suspended`,
   `healthy`, `running`. Tones: `healthy` and `running` ok, `restarting` and `creating` new, `down` bad, the rest
   plain.

## Consequences

- The menu bar, Health and the window say down for a failing check and for a workspace in a loop or ended
  unexpectedly, where the menu bar said nothing before.
- A VM of a silent Mac whose last answer had a restart walking reads `Not seen.`, never `Restarting.`.
- A test over every combination of state, lifetime, check, loop, restart, want and detail holds Head, Health's
  `Down.` and `Down()` to the standing, so a later rule in one of them alone fails it.
- A client words the standing, never the raw state; `MachinesVMState` leaves `words.json` with the window's switch
  to the standing.
- A Mac's standing and the fleet's are later lots of the same shape.
