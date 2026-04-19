# packages/runtimes

Built-in agent-runtime adapters.

## Role

A "runtime" is the thinking engine that actually executes an agent's prompts — Claude Code, Codex, etc. Unlike tools (which live as YAML in the catalog), runtimes ship as Go code because some of what they do is inherently host-side: credential sync, default config materialisation, prelaunch shell wrapping, session-id parsing.

A runtime has up to three orthogonal facets, bundled into a single `Adapter`:

| Facet | What it owns | Where it runs |
|---|---|---|
| **Tool** | Install recipe (apt, curl, npm install, user-side config) | Image build time |
| **Render** | Source → Tree (`transpile.Runtime`) — where each piece of content lands on disk | Compile time |
| **Spawn** | `BuildCommand`, `SyncHostCredentials`, `PrelaunchShell`, `DefaultConfigFiles` | Host at spawn time + container prelaunch |

Each facet is optional. `claudecode` ships all three (full-featured runtime). `codex` ships Tool + Spawn (install + auth plumbing, no renderer yet). A future YAML-first runtime could ship Tool only.

## Key types

- `Adapter` — the umbrella struct bundling Tool + Render + Spawn with identity fields (`Name`, `DefaultProvider`).
- `Spawner` interface — the spawn-time port. Lives here because its only implementers are runtime adapters.
- `SpawnConfig` — parameters for `Spawner.BuildCommand`.
- `Register(Adapter)` / `All() []Adapter` / `Get(name)` / `Names()` — the Adapter registry.
- `RegisterSpawner(Spawner)` / `GetSpawner(name)` / `AllSpawners()` — the Spawner registry (populated automatically when an Adapter with a non-nil Spawn is registered).
- `RegisterDefaults(tool.Registry)` — sugar: iterates `All()` and registers every non-nil `Adapter.Tool` into the dependency resolver.

## Registration pattern

The top-level package does NOT import subpackages (breaks the cycle). Each runtime subpackage exports its `Adapter` value and self-registers via `init()`:

```go
// packages/runtimes/<name>/adapter.go
package <name>

import "spwn.sh/packages/runtimes"

var Adapter = runtimes.Adapter{ Name: …, Tool: Tool, Spawn: Spawner, … }

func init() { runtimes.Register(Adapter) }
```

Callers that want the built-in set blank-import `runtimes/defaults` once:

```go
import _ "spwn.sh/packages/runtimes/defaults"
```

This mirrors the `database/sql` driver pattern. Binaries can pick individual runtimes via direct blank-import if they don't want the full set.

## Subpackages

- `claudecode/` — Claude Code (Tool + Render + Spawn). See `claudecode/RENDER.md` for the layout contract.
- `codex/` — OpenAI Codex (Tool + Spawn). No renderer; codex sessions are launched ad-hoc.
- `defaults/` — convenience blank-import aggregator for every built-in runtime.

## Content vs layout

The spwn-opinionated world content (physics, faculties, roster, architect identity, role-aware NPC prompts) lives in **`packages/transpile/worldbook/`**, not here. Runtime renderers import worldbook's strings and decide how to surface them — the claude-code adapter inlines them into a single self-contained `CLAUDE.md` per agent; a future codex adapter may choose differently. This keeps the prose runtime-neutral and authored once.

## Related

- **Imported by** — `apps/cli`, `apps/api`, `packages/architect`, `tests/catalog`
- **Imports** — `packages/dependency` (tool interface), `packages/transpile` (render + Runtime interface)
