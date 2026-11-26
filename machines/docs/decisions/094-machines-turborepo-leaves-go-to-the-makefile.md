# ADR-094: Machines' Turborepo leaves Go to the Makefile

**Status:** Proposed
**Date:** 2026-11-25

**Amends:** [ADR-046](046-the-repository-is-a-monorepo-of-members.md): its line "Turborepo carries what is per-module and cacheable: Go `lint` and `test` per module".

## Context

[ADR-088](088-agents-live-in-the-one-binary-as-families-reached-by-calls.md) has the engine require Agents' `spwn.sh/packages/*` modules, each by a `replace` line to its folder at the repository root, outside `machines/`. Its context foresaw the cost of putting Go packages out of Machines' task graph; this record is where that cost is paid.

Turborepo 2.11.7's `experimentalGoWorkspaces` reads the whole build list of `machines/go.work` (`go list -m -json all`) and refuses every module replaced by a folder outside its repository root, `machines/`, with no setting to admit one: a symlink does not pass, a second `go.work` and vendoring are refused too (`crates/turborepo-repository/src/go.rs`, `local_replacement_target`). With the engine's `replace` lines in, `make lint` stops at Turborepo with "is outside the repository and cannot be cached, watched, or pruned safely". Turborepo is right to refuse: it would hash the engine's sources and not the Agents packages it compiles, and replay a stale test result.

What the flag bought was small: `make lint` already runs `golangci-lint` (with `govet` enabled) over the three Go modules, so Turborepo's `go vet` repeated it; the `specs` module was uncached; `catalog` has no test. Its one gain was the replay of the engine's `go test ./...` on a push that left the engine unchanged. The gates never used Turborepo for Go.

The alternatives lost: one Turborepo graph at the repository root over both worlds (it loses this replay anyway, merges the two Go graphs and duplicates every run), Agents' modules moved under `machines/` (eleven root modules still require them until the old binary is gone), and tagged module versions (every change to a package would become a release).

## Decision

- **Machines' Turborepo runs the npm members alone.** `turbo.json` sets no `experimentalGoWorkspaces`; Go's lint and tests are direct Makefile lines.
- **`make test` runs one `go test ./apps/engine/... ./catalog/... ./specs/...`** from `machines/`, in workspace mode, the same three modules `make lint` judges; Go's own test cache replays a warm local run.
- **The engine keeps its `replace` lines**, as ADR-088 decides. The root's Turborepo and `go.work` are untouched.
- **The engine's `turbo.json` and the spec that guarded its test inputs go** (`TestTurboReplaysNoStalePass`): no Go test result is replayed, so no input list can be stale.

## Consequences

- A Machines CI run no longer replays the engine's tests from Turborepo's cache on a push that leaves the engine unchanged; Go's test cache covers only the tests that read no file. During the lots that move Agents into the engine nearly every push touches it, so the loss is close to none, and two CI runs after the merge measure it.
- Turning the flag back on reddens `make lint` at once with Turborepo's own error, so no spec restates this configuration.
- The way back: once the old binary is gone, the root's Go is Agents' file modules alone and the engine their only consumer. They then move under `machines/` (or fold into the families), join `machines/go.work`, the flag comes back on, and the record that does it supersedes this one.
