# Architecture

Multi-module Go monorepo. Two apps, domain packages organized in five layers, one web UI.

## Layer architecture

Imports flow **downward only**. A package in layer N may import from layers 1..N-1 but never from N+1 or above. This is enforced by [depguard](https://github.com/OpenPeeDeeP/depguard2) in `.golangci.yml` — a PR that adds an upward import fails lint with the violated rule name.

```
L6  CLI         apps/cli/                        (imports anything)
    ─────────────────────────────────────────────────────────────
L5  Runtime     packages/world/                  (container orchestration)
    ─────────────────────────────────────────────────────────────
L4  Build       packages/compile/                (pure render: Input → Tree)
                packages/image/                  (Docker image builder)
    ─────────────────────────────────────────────────────────────
L3  Project     packages/project/                (manifest + validation)
    ─────────────────────────────────────────────────────────────
L2  Domain      packages/pack/                   (pack schema, refs, lockfile)
                packages/agent/                  (agent mind + composition)
    ─────────────────────────────────────────────────────────────
L1  Foundation  packages/paths/                  (directory constants)
                packages/base/                   (shared types)
                packages/ids/                    (ID generation)
```

### What each layer owns

| Layer | Package | Owns | Does NOT own |
|-------|---------|------|-------------|
| L1 | `paths`, `base`, `ids` | Constants, shared types, ID generation | Nothing else — zero spwn imports |
| L2 | `pack` | `spwn.yaml` schema parsing, ref resolution (`@spwn/`, `github.com/`, bare-name), `spwn.lock` read/write, filesystem loaders | Validation rules, build pipeline |
| L2 | `agent` | `agent.yaml` parsing, mind layers (identity, skills, knowledge, playbooks, journal), evolution | Project manifest, Docker |
| L3 | `project` | Project manifest (`spwn.yaml` with `worlds:`), validation rule engine (15 rules), dep merging | Pack schema, image building |
| L4 | `compile` | Pure render: `Input → Tree`, runtime backends (claude-code, codex) | Source loading, container lifecycle |
| L4 | `image` | Docker image builder, tool registry, Dockerfile generation, tool probing | Pack schema, container orchestration |
| L5 | `world` | Container lifecycle (spawn, destroy, colony), agent sync (docker-cp in/out), runtime state | Pack parsing, compile rendering |
| L6 | `apps/cli` | Thin CLI layer: parse flags, call domain APIs, format output | Domain logic |

### Enforcement mechanisms

1. **depguard** (lint-time) — `.golangci.yml` contains deny rules per layer. Run `golangci-lint run -E depguard` to verify. A package in L3 importing from L5 produces:
   ```
   L3 (project) must not import L5 (runtime)
   ```

2. **Go `internal/`** (compile-time) — implementation details live under `internal/` and the Go compiler itself rejects wrong imports. Examples: `project/internal/validate/`, `project/internal/manifest/`, `world/internal/runtime/`.

3. **This document** (review-time) — the layer diagram is the ground truth. When in doubt, check here.

## Module map

```
spwn/
├── apps/
│   ├── cli/                          The spwn binary (L6)
│   │   ├── pack/                       install/uninstall logic
│   │   ├── skill/                      skill authoring (new/edit/show/rm)
│   │   ├── agent/                      agent commands
│   │   ├── world/                      world commands
│   │   └── install.go                  root-level spwn install/uninstall
│   └── web/                          Next.js + Tauri web/desktop UI
├── packages/
│   ├── pack/                         L2 — pack domain
│   │   ├── schema.go                   spwn.yaml pack format parser
│   │   ├── refs.go                     ref parsing + resolution
│   │   ├── lockfile.go                 spwn.lock text format
│   │   ├── adapter.go                  image.Tool adapter
│   │   └── loader.go                   filesystem + embed resolvers
│   ├── agent/                        L2 — agent mind
│   │   ├── manifest.go                 agent.yaml parsing
│   │   └── internal/                   mind layers, evolution, journal
│   ├── project/                      L3 — project manifest
│   │   ├── manifest.go                 spwn.yaml project parsing
│   │   └── internal/
│   │       ├── validate/               rule engine (15 rules)
│   │       ├── manifest/               manifest loader
│   │       └── resolve/                dep merging (project + agent → flat list)
│   ├── compile/                      L4 — pure render
│   │   ├── runtime.go                  Input → Tree interface
│   │   ├── tree.go                     in-memory file tree
│   │   ├── source/                     source loader (project → Input)
│   │   └── runtimes/
│   │       └── claudecode/             claude-code renderer
│   ├── image/                        L4 — Docker image builder
│   │   ├── imagebuilder.go             Build() + BuildFromBase()
│   │   ├── registry.go                 tool registry + transitive resolution
│   │   └── backend/                    Docker API abstraction
│   ├── world/                        L5 — container orchestration
│   │   ├── architect/                  spawn, destroy, colony, sync
│   │   └── internal/
│   │       ├── runtime/                runtime adapters (claude-code)
│   │       ├── backend/                backend adapter
│   │       └── state/                  container label state
│   ├── paths/                        L1 — directory constants
│   ├── base/                         L1 — shared types
│   └── ids/                          L1 — ID generation
├── catalog/
│   ├── packs/                        built-in packs (@spwn/unix, etc.)
│   ├── runtimes/                     runtime tool definitions
│   └── examples/                     bundled example projects
└── tests/                            e2e test suite
```

## Core abstractions

| Abstraction | Where | Purpose |
|-------------|-------|---------|
| Runtime | `packages/world/internal/runtime` | How an agent runs (builds the CLI command to exec inside a container) |
| Backend | `packages/image/backend` | Where worlds run (container lifecycle, image management) |
| Mind | `packages/agent` | How an agent persists and evolves across runs |
| Pack | `packages/pack` | The distribution unit: schema, refs, lockfile |

## State and data flow

**Source of truth for live worlds**: container labels. `sh.spwn.*` labels are set at container creation time and read back by `packages/world/internal/state`. No `state.json`.

**Source of truth for project config**: `spwn.yaml` + `./spwn/`. `packages/project` parses and validates. `packages/compile` renders to a runtime-specific `Tree`. `packages/image` bakes into Docker images.

**Source of truth for user identity**: `~/.spwn/`. Credentials, daemon state, activity log. Never project-scoped.

## Key invariants

- **Per-repository**. Agents and local blocks live in `./spwn/`, not `~/.spwn/`.
- **Declared deps only**. An agent can only use packs declared in `deps:`. Unlisted = physically absent.
- **External deps, local blocks**. `deps:` for `@spwn/*` and `github.com/*`. Local authoring in typed dirs: `spwn/skills/`, `spwn/tools/`, `spwn/hooks/`. See [`docs/dependencies.md`](dependencies.md).
- **Transitive resolution**. Pack dependencies resolved recursively, topologically sorted.
- **Lock file is text**. `spwn.lock` — one dep per line, trivially diffable.
- **Labels are truth**. World info comes from Docker labels, not on-disk state.
- **Compile is deterministic**. Same input → same output, covered by golden tests.
- **Layers flow downward**. Enforced by depguard. No upward imports.

## Roadmap

- 🟢 Per-repository projects (`spwn init` / `check` / `build`)
- 🟢 World creation and isolation
- 🟢 Persistent agent identity and memory
- 🟢 Composable pack catalog
- 🟢 Reproducible build artifacts
- 🟢 5-layer architecture with lint enforcement
- 🟡 Agent evolution (dream, sleep, fork)
- 🟡 Multi-agent coordination via filesystem inboxes
- 🟡 Snapshots and rollback
- 🟡 Desktop app (Tauri) + web UI
- 🔴 GitHub-based pack distribution (`github.com/owner/repo`)
- 🔴 Additional runtime adapters (Codex, Aider, OpenCode, Gemini CLI)
- 🔴 Additional backends (Firecracker, gVisor, K3s, Fly.io)
- 🔴 Cloud-hosted worlds
