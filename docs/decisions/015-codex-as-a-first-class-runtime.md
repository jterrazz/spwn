# ADR-015: codex as a first-class runtime

**Status:** Proposed
**Date:** 2026-09-09

Written 2026-09-09 from the change audit `docs/notes/codex-as-first-class-runtime.md`
(committed 2026-05-03) and from the code that now implements it.

## Context

codex was registered as an adapter, but only as an *installable binary*: it
carried a `Tool` facet and a partial `Spawn` facet, so `spwn install codex`
worked and an image could be built with it. Nothing past that did.

Three things stood in the way of a codex-backed agent actually holding a
session, and each of them was the same mistake wearing a different hat —
claude-code's shape had leaked out of `packages/runtimes/claudecode/` and
hardened into layers that are supposed to be runtime-neutral.

- **No renderer.** `spwn build --runtime codex` resolved and invoked
  `transpile.Compile("codex", …)`, then failed: no codex renderer was
  registered. Claude Code composes its prompt from `@path` imports; codex
  reads `AGENTS.md` at the cwd and imports nothing, so the same tree cannot
  serve both.
- **One spawner for the whole Architect.** `Architect` captured a single
  `runtimes.GetSpawner("claude-code")` at construction and three call
  sites — `agent.go` twice, `npc.go` once — built their command from it. A
  correctly-built codex world would boot, then be asked to run
  `claude --dangerously-skip-permissions -p "…"`.
- **The CLI synthesised claude's flags and parsed claude's envelope.**
  `spwn agent talk` appended `--print --output-format json` after
  `BuildCommand`, and unmarshalled `{result, session_id}` from the output.
  Both are Claude Code's dialect; codex speaks `exec` and `thread_id`.

The audit costed the work at one to two days, most of it the renderer and the
test infrastructure a second runtime needs to be provable at all.

## Decision

**codex is a first-class runtime, and every runtime-specific behaviour lives
behind the `runtimes` port — never in the Architect, never in the CLI.**

Concretely: codex ships a `Render` facet beside its `Tool` and `Spawn`; the
Architect holds no spawner and resolves one per world from the world record;
and the `Spawner` interface grows the two pieces of dialect the CLI had
inlined, `OneShotFlags` (the flags that tell a CLI to print and exit) and
`ParseOneShotResult` (the text and session id read back out).

This is [ADR-010](010-ports-and-adapters.md) applied to a port that had one
implementation and had therefore stopped being one. A second runtime is the
only thing that can prove a runtime port is a port, which is why the promotion
is a decision and not a feature: it fixes a boundary, and adding a third
runtime is now an adapter rather than a sweep.

## Consequences

- The renderer inlines what Claude Code imports. codex's `AGENTS.md` carries
  SOUL, physics, faculties, roster, conventions, playbooks and role in one
  file, and the runtime-neutral prose stays in `packages/transpile/worldbook`
  so the two renderers cannot drift into two different worlds.
- The source file and the rendered file share the name `AGENTS.md` and are
  never the same file: the source stays at `spwn/agents/<name>/AGENTS.md` on
  the host, the projection lands at `/agents/<name>/AGENTS.md` in the
  container.
- A second runtime doubles the test surface. `tests/_simulators/codex`
  stands in for the real CLI so no E2E hits OpenAI, and every golden case
  under `packages/runtimes/testdata/` carries a codex output tree beside its
  claude-code one — 31 of them today, regenerated with `UPDATE_GOLDEN=1`.
- Two items of the audit stayed unbuilt on purpose. Neither runtime has a
  `catalog/<runtime>/spwn.yaml` stub, so neither is browsable from
  `spwn install`'s gallery — a UX gap that is the same for both, and so not
  codex's to close. And the scaffold template takes its backend from the
  caller rather than naming the alternatives in a comment.
- The registry is what a production binary reads. A new runtime is inert
  until it is blank-imported in `packages/runtimes/defaults/defaults.go`,
  which is the one place the shipped set is declared.
