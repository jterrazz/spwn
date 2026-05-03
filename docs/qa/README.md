# Manual QA

Spwn's automated test suite is in `tests/` and `packages/*/_test.go`.
This directory holds **manual QA passes** — scenario catalogs that a
human (or human-driven Claude session) walks through end-to-end,
plus the run reports they produce.

These are not part of `make test`. They run against a real
Anthropic-authenticated runtime, real Docker, and real disk, and
exercise paths that automated tests skip on purpose (cost,
non-determinism, agent-perception bugs).

## Suites

| Suite | Scenarios | Driver | What it probes |
|---|---|---|---|
| [`cli-scenarios/`](cli-scenarios/scenarios.md) | 50 | `harness.sh` (bash) | Realistic command sequences a user runs in a day — init → build → up → talk → down. Sub-assertions on exit codes, file existence, output substrings. Mostly automatable; uses `mock-claude` for runtime calls. |
| [`agent-behavior/`](agent-behavior/scenarios.md) | 50 | Human + live Claude | Whether host-side setup (CLAUDE.md, playbooks, skills, hooks, tools, knowledge, roster, messaging) is actually injected into the agent's perception. Cannot use mock-claude — needs a real session. |
| [`edge-cases/`](edge-cases/scenarios.md) | 72 | Mixed (bash + manual) | Concurrency, partial failure, state-machine holes, FS edges (special chars, perms, symlinks, large/binary), tool-install failure modes. |

## Run reports

Each pass leaves a dated report in its suite's `runs/` folder. The
report records: scoreboard, bugs surfaced, commits that fixed them,
and any scenario that's still failing.

| Pass | Date | Bugs fixed | Report |
|---|---|---|---|
| Agent-behavior | 2026-04-19 | 5 critical | [`agent-behavior/runs/2026-04-19.md`](agent-behavior/runs/2026-04-19.md) |
| Edge-cases | 2026-04-19 | 4 (cumulative: 11) | [`edge-cases/runs/2026-04-19.md`](edge-cases/runs/2026-04-19.md) |
| CLI scenarios | rolling | 2 fixed on baseline pass; runs against `main` continuously | (last green at commit `da978c1c`; no standalone report) |

## Adding a new pass

1. Drop a new dated file under the suite's `runs/` folder
   (`YYYY-MM-DD.md`). One report per pass; never edit an old one
   — they're a history of what the system looked like that day.
2. List the scoreboard at the top, the bugs in the middle, and any
   commits that fixed them at the bottom.
3. If you discover a category the existing three suites don't
   cover, propose a new suite folder rather than overloading one
   of the existing scenario files.

## How this relates to the automated suite

Manual QA tests **whole-system coherence** — the kind of bug where
two correct subsystems produce wrong behavior together. The
automated suite tests subsystems in isolation. Both are necessary;
neither replaces the other. See `tests/ARCHITECTURE.md` for the
automated layer pyramid and `../notes/test-architecture-rationale.md`
for the original design rationale that produced both.
