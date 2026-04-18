# packages/transpile/worldbook

Spwn's opinionated world content — the default prose, skills, and identity material that every compiled world ships with regardless of which runtime renders it.

## Role

Runtime renderers (`packages/runtimes/<name>/render.go`) do two things:

1. Emit spwn's opinionated world content (physics, agent operating manual, system skills, architect identity, roster, role-aware agent context).
2. Place each file at a runtime-specific path (`agents/<n>/CLAUDE.md` for Claude, whatever codex wants, …).

worldbook owns (1). It's runtime-neutral — every renderer imports from here for the prose and layers only its layout choices on top. This keeps the content authored once and shared across every runtime that ships a renderer.

## What lives here

| File | Content |
|---|---|
| `physics.go` | `GeneratePhysics`, `GenerateFaculties` — the world filesystem rules and the verified-tools briefing. |
| `manual.go` | `AgentsBook`, `SystemSkills` + every system-skill const (mind-management, collaboration, world-awareness, self-evolution). Two variants gated by `WorldKnowledgeMounted`. |
| `context.go` | `GenerateAgentContext` (role-aware prompt: chief / manager / worker / npc / architect), `GenerateRoster`, shared value types (`Workspace`, `AgentContextOpts`, `AgentInfo`, `ColonyAgentSpec`). |
| `architect.go` | `ArchitectIdentity`, `ArchitectSystemFiles`, architect skills (fleet ops, task planning, monitoring). |

## What does NOT live here

- **Runtime-specific layout** — which file path, what entrypoint filename, whether to use `@-imports`. Lives in `packages/runtimes/<name>/render.go`.
- **User content** — the user's own `AGENTS.md`, `SOUL.md`, `mind/`, `skills/`. That flows through `packages/transpile/source/` unchanged.
- **Compile state** — tool lists, manifest, agent roster data. Lives in `transpile.Input`.

## Consumers

- `packages/runtimes/claudecode/render.go` — emits `worldbook.GeneratePhysics(...)`, `worldbook.AgentsBook(...)`, etc. at Claude-specific paths.
- `packages/architect/build.go` — bakes `worldbook.ArchitectSystemFiles()` into the architect image.
- `packages/architect/npc.go` — uses `worldbook.GenerateAgentContext` for NPC prompt composition.
- `packages/architect/colony.go` — uses `worldbook.GenerateRoster` to regenerate `roster.md` on hot-deploy.
- `packages/architect/spawn_workspaces.go`, `spawn_agents.go` — use the shared `Workspace` / `ColonyAgentSpec` value types.

## Future: `spwn build --bare`

The content/layout split makes a "bare" compile mode a one-flag change: skip `worldbook` injection and emit only user content + runtime layout. Useful for users who want spwn to compile refs and produce a tree without injecting spwn's opinions about how agents should think about their world.

## Related

- **Imported by** — `packages/runtimes/*`, `packages/architect`
- **Imports** — stdlib only. Pure content package.
