# packages/transpile/worldbook

Spwn's opinionated world content — the runtime-neutral prose blocks
that every renderer inlines into the boot prompt it emits.

## Role

worldbook hands renderers three markdown strings: physics, faculties,
roster. Each runtime's renderer then decides where to place them
(the claude-code adapter inlines all three into every agent's
`CLAUDE.md`; a future codex adapter may do something different).

The split keeps content authored once and shared across every
runtime renderer.

## What lives here

| File | Content |
|---|---|
| `physics.go` | `GeneratePhysics`, `GenerateFaculties` — the world filesystem rules and the verified-tools briefing. |
| `context.go` | `GenerateRoster` (who else is in this world) plus its `ColonyAgentSpec` value type. |
| `architect.go` | `ArchitectIdentity`, `ArchitectSystemFiles`, architect skills (fleet ops, task planning, monitoring). |

## What does NOT live here

- **Runtime-specific layout** — which file path, what entrypoint filename, whether to use `@-imports`. Lives in `packages/runtimes/<name>/render.go`.
- **Per-agent CLAUDE.md assembly** — the self-contained system prompt is built by `packages/runtimes/claudecode/render_agent.go` (inlines physics + faculties + roster + playbook index + conventions).
- **User content** — the user's own `AGENTS.md`, `SOUL.md`, `playbooks/`, `skills/`. That flows through `packages/transpile/source/` unchanged.
- **Compile state** — tool lists, manifest, agent roster data. Lives in `transpile.Input`.

## Consumers

- `packages/runtimes/claudecode/render.go` — calls `GeneratePhysics`, `GenerateFaculties`, `GenerateRoster` for the inlined sections of each agent's CLAUDE.md.
- `packages/architect/build.go` — bakes `ArchitectSystemFiles()` into the architect image.

## Related

- **Imported by** — `packages/runtimes/*`, `packages/architect`
- **Imports** — stdlib only. Pure content package.
