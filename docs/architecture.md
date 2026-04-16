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

- **Per-repository**. Agents and plugins live in `./spwn/`, not `~/.spwn/`. `spwn init` enforces this on a fresh directory.
- **Declared plugins only**. An agent can only reach for plugins its `agent.yaml` declares under the unified `plugins:` list. Anything not listed is physically absent from the world's image.
- **Everything is a pack**. Tools, runtime-config injectors, and skills are one concept — the only thing that differs is which fields the manifest populates. A `pack.yaml` with an `install:` block is a tool; one with a `runtime-config:` block also injects runtime config (MCP servers, hooks, settings) into the target runtime's config file at spawn time; a bare `.md` file is a skill.
- **Dependencies resolve like npm**. `@spwn/<name>` is a catalog pack compiled into the binary; `<bare-name>` is a local pack under `spwn/packs/<name>/` (directory form) or `spwn/packs/<name>.md` (bare-markdown skill); `@<owner>/<name>` is reserved for a future community registry. Catalog pins live in `spwn.lock` at the project root, managed by `spwn install` / `spwn install`. `spwn check` flags drift between agent.yaml and the lockfile.
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
