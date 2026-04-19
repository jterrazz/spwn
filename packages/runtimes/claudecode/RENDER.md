# packages/runtimes/claudecode

The **Claude Code** runtime adapter.

Full three-facet adapter:

| Facet | File | What it does |
|---|---|---|
| Tool | `tool.go` | Image-build recipe — curl-based native installer for the `claude` binary. |
| Render | `render.go`, `render_agent.go` | Translates a provider-neutral `transpile.Input` into a Tree laid out the way Claude Code expects. |
| Spawn | `spawn.go` | Host-side spawn-time behavior — `BuildCommand`, credential sync, prelaunch shell, default config files. |

The `Adapter` value in `adapter.go` bundles all three and self-registers with `packages/runtimes` at init time.

## What Claude Code expects

[Claude Code](https://www.anthropic.com/claude-code) reads a `CLAUDE.md` from the working directory on startup, treats `@path/to/file.md` as `#include`-style imports, and uses the composed text as the session's system prompt.

This package is the only place in spwn that encodes those conventions. Everything user-authored under `spwn/` uses the neutral name `AGENTS.md`; the rename to `CLAUDE.md` happens here.

## Render = layout, not content

The rendered Tree contains two kinds of entries:

- **Spwn-opinionated content** — physics, faculties, roster, the agent operating manual, system skills, architect identity, role-aware agent context. These are runtime-neutral strings. The prose lives in **`packages/transpile/worldbook/`** — this renderer just decides where to place it on disk.
- **Runtime-specific layout** — `agents/<name>/CLAUDE.md` with Claude's `@-import` syntax. That's `render_agent.go` (`GenerateAgentCLAUDEMD`), the one truly Claude-specific piece.

Split cleanly: when a second runtime (codex, …) grows a renderer, it imports the same `worldbook` content and picks different paths/names. No prose is duplicated.

## Emitted paths

| Path | Source |
|---|---|
| `world/physics.md` | `worldbook.GeneratePhysics` |
| `world/faculties.md` | `worldbook.GenerateFaculties` |
| `world/AGENTS.md` | `worldbook.AgentsBook` (agent operating manual) |
| `world/roster.md` | `worldbook.GenerateRoster` |
| `world/skills/*.md` | `worldbook.SystemSkills` (mind-management, collaboration, world-awareness, self-evolution) |
| `agents/<name>/CLAUDE.md` | `claudecode.GenerateAgentCLAUDEMD` — Claude-specific entrypoint |
| `agents/<name>/worlds/<id>/role.md` | Per-deployment role description |

Anything else is a bug — `architect.Spawn`'s tree materialiser rejects unknown prefixes.

## What this adapter does NOT touch

- **Runtime state** — `inbox/`, `outbox/`, `notes/` are `mkdir`'d by `architect.Spawn`, not rendered. They're state, not content.
- **Docker config** — base image, binds, env vars, labels. `packages/compile` and `packages/architect`.

## Extending

- Edit the prose: `packages/transpile/worldbook/*.go`.
- Edit where prose lands in a Claude image: `render.go`.
- Edit the Claude entrypoint syntax: `render_agent.go`.
- Edit the install recipe: `tool.go`.
- Edit host/spawn-time behavior: `spawn.go`.

## Testing

- Unit tests next to each file (`tool_test.go`, `spawn_test.go`). Generator-level tests for content live in `worldbook/`.
- Golden-fixture tests at the render boundary: `packages/runtimes/golden_test.go` walks `testdata/<case>/` and diffs the rendered Tree against `output_claude_code/`. Regenerate with `UPDATE_GOLDEN=1 go test ./packages/runtimes/...`.
