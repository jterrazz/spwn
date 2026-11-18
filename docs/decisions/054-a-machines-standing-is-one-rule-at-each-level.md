# ADR-054: A machine's standing is one rule at each level, and every verdict reads it

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

A Mac's status had the same fault. `SentenceOf`, `UnseenVerdict` and the Mac branch of `Findings` each tested
reachability and the engine on their own: a Mac whose engine is not installed, on another Mac, was told it `has its
engine stopped`, and a Mac that does not answer was a broken finding while its `Unseen` verdict said nothing proves it
down. The findings of a VM disagreed with its head the same way: a workspace ended unexpectedly needed a look while its
head said down, and a running VM whose SSH fails was broken while its head said running. No answer said how all the
machines stand, so each client counted its own.

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

5. **A Mac has one standing, the first rule that matches:** `asleep` (it does not answer, and last answered on battery
   with its lid closed), `not-seen` (it does not answer otherwise, never answered included), `engine-stopped`,
   `engine-not-installed`, `online` (its engine runs, or this Mac cannot know it). `MacStandingOf` owns it, and
   `machines.list` carries it as `MacView.Standing` (`MachinesMacStanding`). `SentenceOf`, `UnseenVerdict` and the Mac
   branch of `Findings` switch on it with no default. Tones: `online` ok, `asleep` plain, `not-seen` warn (a Mac that is
   silent and not asleep is worth a look, never down), `engine-stopped` bad, `engine-not-installed` warn.
6. **A set of machines rolls up to one standing**, `RollUp{State, N}` (`MachinesRollUp`), read from the Mac and VM
   standings alone, the first rule that matches, bad before warn: `down` (a VM down), `engine-stopped` (a Mac whose
   engine is stopped), `not-seen` (a Mac not seen), `engine-not-installed` (a Mac whose engine is not installed),
   `empty` (no VM at all), `fine`. A Mac not seen comes before a Mac without its engine, since a silent Mac may hide a
   VM down and a missing engine is a known gap in its setup. `N` counts what the state names: VMs for `down`, Macs
   for the others. A VM of a Mac that does not answer counts through its Mac, and an asleep Mac is no fault.
   `machines.list` carries the roll-up of every Mac as `AllMachines.Standing`, and each Mac's own as
   `MacView.Machines`. The word `fleet` is retired, so the type is named for what it does.

    Its words live on the type (`Word()`, `Noun(n)`) and carry their noun through `One` and `Other`
    (`1 machine down`, `1 Mac with its engine stopped`, `2 Macs not seen`, `1 Mac without its engine`,
    `no machine yet`, `all running`), so a client draws `N` and the word and never makes its own plural. A state that
    counts takes the tone of the standing it counts: `down` and `engine-stopped` bad, `not-seen` and
    `engine-not-installed` warn; `empty` plain, `fine` ok. So a roll-up never says a Mac worse than its own standing
    does: a Mac whose engine is not installed is not a machine down.

7. **A finding's severity is its machine's standing's tone**: broken where it is bad, needs a look otherwise
   (`severityOf`). What a finding says is still read from the facts; that it is broken is not. So a workspace ended
   unexpectedly is broken (its standing is down), a running VM whose SSH fails needs a look (its standing is running;
   its head says running), a Mac that does not answer needs a look, and a server stopped while its record wants it
   running, down with no finding before, is `<name> is down`.

## Consequences

- The menu bar, Health and the window say down for a failing check and for a workspace in a loop or ended
  unexpectedly, where the menu bar said nothing before.
- A VM of a silent Mac whose last answer had a restart walking reads `Not seen.`, never `Restarting.`.
- A test over every combination of state, lifetime, check, loop, restart, want and detail holds Head, Health's
  `Down.` and `Down()` to the standing, so a later rule in one of them alone fails it.
- A client words the standing, never the raw state; `MachinesVMState` leaves `words.json` with the window's switch
  to the standing.
- A test over every combination of reach, sleep, engine and a hosted server holds a Mac's `Unseen` and its findings
  to its standing, and one over the VM cross product holds every VM finding broken exactly when its standing is down.
- A test over every pair of Macs, one of each standing, holds a roll-up's tone to the worst standing it counts.
- A Mac's sentences are byte-identical. What moves: a far Mac whose engine is not installed `has no engine installed`;
  `Unseen` of a Mac not seen and not asleep is warn; the findings' severities above, in the CLI's Needs you, the
  Overview's and the menu bar's needs.
- The CLI's own Mac and VM columns still judge their rows apart from the standing, until its own lot.
