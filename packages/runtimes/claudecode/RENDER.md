# packages/runtimes/claudecode

The **Claude Code** runtime adapter.

Full three-facet adapter:

| Facet | File | What it does |
|---|---|---|
| Tool | `tool.go` | Image-build recipe — curl-based native installer for the `claude` binary. |
| Render | `render.go`, `render_agent.go` | Turns a provider-neutral `transpile.Input` into a tree of files Claude Code can boot from. |
| Spawn | `spawn.go` | Host-side spawn-time behavior — `BuildCommand`, credential sync, prelaunch shell (including the `.claude/skills` symlink), default config files. |

The `Adapter` value in `adapter.go` bundles all three and self-registers with `packages/runtimes` at init time.

## What Claude Code expects

[Claude Code](https://www.anthropic.com/claude-code) reads a `CLAUDE.md` from the working directory on startup, treats `@path/to/file.md` as `#include`-style imports, and uses the composed text as the session's system prompt. Native skill discovery walks `$HOME/.claude/skills/<name>/SKILL.md`.

This package is the only place in spwn that encodes those conventions. Everything user-authored under `spwn/` uses the neutral name `AGENTS.md`; the rename to `CLAUDE.md` happens here.

## One self-contained prompt per agent

`render_agent.go`'s `GenerateAgentCLAUDEMD` builds a single CLAUDE.md that inlines **everything the agent needs at boot**:

- `@SOUL.md` — identity import
- Physics — world rules (inlined from `worldbook.GeneratePhysics`)
- Faculties — installed tools (inlined from `worldbook.GenerateFaculties`)
- Roster — who else is in this world (inlined from `worldbook.GenerateRoster`)
- `@worlds/<id>/role.md` — per-deployment role import
- Your playbooks — auto-index of frontmatter-promoted playbooks (omitted when empty)
- Conventions — memory, messaging, evolution rules folded in as bullets

No separate `world/physics.md`, `world/faculties.md`, `world/AGENTS.md`, `world/roster.md`, or `world/skills/*.md` files are emitted. Any file outside the paths below is a renderer bug.

## Emitted paths

| Path | Source |
|---|---|
| `agents/<name>/CLAUDE.md` | `claudecode.GenerateAgentCLAUDEMD` — self-contained system prompt |
| `agents/<name>/worlds/<id>/role.md` | Per-deployment role description |

Tool-shipped skills (`SKILL.md` files from each resolved `tool.Tool`) live at `/world/skills/<tool>/SKILL.md` inside the image (baked in by `packages/dependency/resolver.CollectSkills`). `PrelaunchShell` symlinks `$HOME/.claude/skills` → `/world/skills` at spawn time so Claude Code discovers them through its native mechanism — no manual index needed.

## What this adapter does NOT touch

- **Runtime state** — `inbox/`, `outbox/`, `notes/` are `mkdir`'d by `architect.Spawn`, not rendered. They're state, not content.
- **Docker config** — base image, binds, env vars, labels. `packages/compile` and `packages/architect`.

## Extending

- Edit the prose (physics/faculties/roster): `packages/transpile/worldbook/*.go`.
- Edit the CLAUDE.md shape, Conventions bullets, or playbook index format: `render_agent.go`.
- Edit the render-level layout (what gets emitted where): `render.go`.
- Edit the install recipe: `tool.go`.
- Edit host/spawn-time behavior: `spawn.go`.

## Testing

- Unit tests next to each file (`tool_test.go`, `spawn_test.go`).
- Golden-fixture tests at the render boundary: `packages/runtimes/golden_test.go` walks `testdata/<case>/` and diffs the rendered tree against `output_claude_code/`. Regenerate with `UPDATE_GOLDEN=1 go test ./packages/runtimes/...`.
