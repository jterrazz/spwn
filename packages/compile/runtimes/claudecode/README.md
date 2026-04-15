# packages/compile/runtimes/claudecode

The **Claude Code runtime** for the spwn compiler. Translates a
provider-neutral spwn project into the file layout Anthropic's
Claude Code agent runtime expects to find on boot.

## What is Claude Code

[Claude Code](https://www.anthropic.com/claude-code) is Anthropic's
terminal-based agent runtime. On startup it reads a `CLAUDE.md`
file from the working directory, treats `@path/to/file.md`
references inside markdown as `#include`-style imports, and uses
that as the system prompt for the session.

This package is the only place in spwn that knows about those
conventions. Everything under `packages/compile/runtimes/` is
runtime-private; nothing in `spwn/`-committed source should ever
mention `CLAUDE.md` directly.

## What it translates

| Source (provider-neutral)                | Target (Claude Code)                        |
| ---------------------------------------- | ------------------------------------------- |
| `spwn/agents/<name>/AGENT.md`            | `/agents/<name>/CLAUDE.md`                  |
| `spwn/agents/<name>/identity/profile.md` | `/agents/<name>/identity/profile.md` (as-is)|
| `spwn/agents/<name>/skills/*`            | `/agents/<name>/skills/*` (as-is)           |
| `compile.Input.VerifiedTools`            | `/world/faculties.md` (generated)           |
| `compile.Input.Manifest`                 | `/world/physics.md` (generated)             |
| `compile.Input.Agents`                   | `/world/roster.md` (generated)              |
| *(static content)*                       | `/world/AGENTS.md` (operating manual)       |
| *(static content)*                       | `/world/skills/*.md` (system skills)        |
| *(per-agent, per-world)*                 | `/agents/<name>/worlds/<id>/role.md`        |

The key rename is **AGENT.md → CLAUDE.md**: the committed source
uses the neutral name, the emitted tree uses Claude's convention.

## What this runtime emits

Paths in the `Tree` are grouped under two top-level namespaces:

- **`world/…`** - shared per-world files. `architect.Spawn` binds
  these into the container at `/world/` through a host directory
  under `~/.spwn/world-states/<id>/`.
- **`agents/<name>/…`** - per-agent home content. Bound into the
  container at `/agents/` through `~/.spwn/agents/`.

Anything else is a bug - the `materialiseWorldTree` helper in
`architect.Spawn` will return an error if the tree contains an
unexpected prefix.

## What this runtime does NOT touch

- **Runtime state** - inbox/outbox/notes directories are empty
  mkdir'd by `architect.Spawn`, not the renderer. They're state,
  not generated content.
- **Plugin settings** - `/home/spwn/.claude/settings.json` is merged
  at spawn time by `injectPluginRuntimeConfig`, because it reads
  the baseline from inside a running container and can't be done
  at compile time.
- **Docker config** - base image, binds, env vars, labels. Those
  live in `packages/image` and `packages/world/architect`.

## Where to hook

Want to change what Claude Code sees on boot? Edit the generator
you care about:

- `physics.go` - `GeneratePhysics(manifest)` - `/world/physics.md`
- `physics.go` - `GenerateFaculties(tools)` - `/world/faculties.md`
- `system_files.go` - `AgentsBook`, `SkillMindManagement`, etc. -
  static operating manual and system skills.
- `agent_context.go` - per-agent context generators, used by
  colony renders and NPC flows.
- `agent_entrypoint.go` - `GenerateAgentCLAUDEMD(name, role)` -
  the per-agent `CLAUDE.md` boot file.
- `runtime.go` - `Render(input)` - the top-level glue that stitches
  all of the above into one `Tree`.

## Testing

Unit tests live next to each generator
(`physics_test.go`, `system_test.go`, `architect_files_test.go`,
`agent_context_test.go`). They compare the rendered string against
expected content.

Phase 4 will add a golden-fixture test at the runtime boundary:
feed a fixed `compile.Input`, diff the produced `Tree` against a
checked-in `testdata/golden/` map. That's the true lockdown
against regressions.
