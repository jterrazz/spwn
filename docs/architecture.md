# Architecture

Multi-module Go monorepo. Two apps, a handful of domain packages, one web UI.

## Module map

```
spwn/
├── packages/                       Domain libraries (Go modules)
│   ├── world/                        World lifecycle (the core)
│   ├── agent/                        Mind lifecycle, composition, evolution
│   ├── manifest/                     Project manifest: parse, validate, build
│   ├── imagebuilder/                 Composable Docker images, tool catalog
│   ├── messenger/                    Filesystem inbox / outbox
│   ├── migration/                    Schema migrations across releases
│   └── foundation/                   Primitives (paths, IDs, auth, activity)
├── apps/
│   ├── cli/                          The spwn binary
│   └── web/                          Next.js + Tauri web/desktop UI
├── examples/                       Bundled example gallery
└── fixtures/                       Test fixtures (mock-claude, testdata)
```

Each package exposes a small public API in its root `.go` file. Implementation details live under `internal/`. The CLI is deliberately thin: parse flags, call a domain API, format output.

## Core abstractions

| Abstraction | Where          | Purpose                                                  |
|-------------|----------------|----------------------------------------------------------|
| Runtime     | `packages/world/internal/runtime` | How an agent runs (builds the CLI command to exec inside a container) |
| Backend     | `packages/world/internal/backend` | Where worlds run (container lifecycle, image management) |
| Mind        | `packages/mind`                   | How an agent persists and evolves across runs            |
| Manifest    | `packages/manifest`               | Project definition on disk + artifact pipeline           |

Each one is a Go interface or type whose implementations can be swapped. Today there is one of each. Adding a new runtime adapter is ~50 lines of Go plus an install recipe in the imagebuilder catalog.

## Runtime adapters

| Runtime     | Status     |
|-------------|------------|
| Claude Code | 🟢 working |
| Codex       | 🔴 planned |
| Aider       | 🔴 planned |
| Gemini CLI  | 🔴 planned |
| OpenCode    | 🔴 planned |

Only Claude Code can actually be spawned today. The other names appear in the tool catalog (imagebuilder can install their binaries into a container) but no runtime adapter wires them into `spwn up` yet.

## State and data flow

**Source of truth for live worlds**: container labels. `sh.spwn.*` labels are set at container creation time and read back by `packages/world/internal/state`. There is no `state.json` that tracks worlds anymore - the labels are the state.

**Source of truth for project config**: `spwn.yaml` + `./spwn/`. `packages/project` parses these and validates them via a rule engine (15 rules, see `internal/validate/`). `packages/compile` renders the validated tree into a runtime-specific `Tree`, and `spwn build` bakes that `Tree` onto a base image to produce a project-specific Docker image.

**Source of truth for user identity**: `~/.spwn/`. Credentials, daemon state, activity log. Never project-scoped.

## Key invariants

- **Per-repository**. Agents and local blocks live in `./spwn/`, not `~/.spwn/`. `spwn init` enforces this on a fresh directory.
- **Declared deps only**. An agent can only reach for packs its `agent.yaml` declares in `deps:`. Anything not listed is physically absent from the world's image.
- **External deps, local blocks**. `deps:` is for external references (`@spwn/*`, `github.com/*`). Local authoring goes in typed directories: `spwn/skills/` (bare `.md`), `spwn/tools/` (install recipes), `spwn/hooks/` (lifecycle scripts). See [`docs/dependencies.md`](dependencies.md) for the full model.
- **Transitive resolution**. Dependencies declared in a pack's `pack.yaml` are resolved recursively and topologically sorted. Users only list direct deps.
- **Lock file is text**. `spwn.lock` is line-oriented (one dep per line), trivially diffable. Managed by `spwn install` / `spwn uninstall`.
- **Labels are truth**. Any world info the CLI displays comes from reading Docker labels, not an on-disk state file.
- **Compile is deterministic**. Running `spwn build --tree-only` on an unchanged project tree produces byte-identical output - covered by the renderer golden tests in `packages/compile/runtimes/`.

## Roadmap

- 🟢 Per-repository projects (`spwn init` / `check` / `build`)
- 🟢 World creation and isolation
- 🟢 Persistent agent identity and memory
- 🟢 Composable tool catalog (imagebuilder)
- 🟢 Reproducible build artifacts
- 🟡 Agent evolution (dream, sleep, fork)
- 🟡 Multi-agent coordination via filesystem inboxes
- 🟡 Snapshots and rollback
- 🟡 Desktop app (Tauri) + web UI
- 🔴 Registry for sharing agents, tool packs, skills, profiles
- 🔴 Additional runtime adapters (Codex, Aider, OpenCode, Gemini CLI)
- 🔴 Additional backends (Firecracker, gVisor, K3s, Fly.io)
- 🔴 Cloud-hosted worlds
