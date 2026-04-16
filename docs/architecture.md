# Architecture

Multi-language monorepo (Go + TypeScript + Rust). Go packages organized in enforced layers; TS and Rust have their own boundaries.

## Layer architecture

Imports flow **downward only**. Enforced at three levels:

1. **depguard** (`.golangci.yml`) — lint-time deny rules per layer
2. **Go `internal/`** — compile-time privacy for implementation details
3. **This document** — review-time ground truth

```
L7 Surface   apps/cli/  apps/web/  apps/api/        (imports anything)
   ────────────────────────────────────────────────────────────────
L6 Runtime   packages/world/                        (orchestration hub)
   ────────────────────────────────────────────────────────────────
L5 Build     packages/compile/  packages/image/     (project → image)
   ────────────────────────────────────────────────────────────────
L4 Project   packages/project/                      (manifest + validation)
   ────────────────────────────────────────────────────────────────
L3 Domain    packages/deps/  packages/agent/        (dependencies + agents)
   ────────────────────────────────────────────────────────────────
L2 Platform  packages/activity/  packages/auth/
             packages/upgrade/   packages/mailbox/  (platform utilities)
   ────────────────────────────────────────────────────────────────
L1 Foundation packages/paths/                       (constants + IDs)
```

### What each package owns

| Pkg | Layer | Owns |
|-----|-------|------|
| `paths` | L1 | Directory constants, ID generation, PATH setup, runtime image constants |
| `activity` | L2 | Append-only event log (JSONL) |
| `auth` | L2 | Provider resolution, credential storage (keychain/env/file/OAuth) |
| `upgrade` | L2 | Version checking + schema migrations runner |
| `mailbox` | L2 | Agent-to-agent filesystem inbox |
| `deps` | L3 | `spwn.yaml` dependency schema, ref parsing, `spwn.lock` read/write, filesystem loaders |
| `agent` | L3 | Agent mind (identity/skills/knowledge/playbooks/journal), evolution, session |
| `project` | L4 | Project manifest, 15 validation rules, scaffolding |
| `image` | L5 | Docker image build, tool registry, transitive dep resolution, dependency→Tool adapter |
| `compile` | L5 | Pure render: `Input → Tree`, runtime backends (claudecode) |
| `world` | L6 | Container lifecycle (spawn, destroy, colony), docker-cp sync |
| `apps/cli` | L7 | `spwn` binary — commands, UI |
| `apps/web` | L7 | Next.js + Tauri desktop app |
| `packages/api` | L7 | HTTP server (backs web UI) |

### Module map

```
spwn/
├── apps/
│   ├── cli/                    Go — the spwn CLI binary
│   └── web/                    TS + Rust — Next.js + Tauri desktop
├── packages/
│   ├── paths/                  L1  dir constants, IDs, env setup
│   ├── activity/               L2  event log
│   ├── auth/                   L2  credentials
│   ├── upgrade/                L2  version + migrations
│   ├── mailbox/                L2  agent messaging
│   ├── deps/                   L3  dependency schema, refs, lockfile
│   ├── agent/                  L3  agent mind + evolution
│   ├── project/                L4  project manifest + validation
│   ├── compile/                L5  render Input → Tree
│   ├── image/                  L5  Docker image build
│   ├── world/                  L6  container orchestration
│   └── api/                    L7  HTTP server
├── catalog/
│   ├── dependencies/                  built-in dependencies (embedded)
│   ├── runtimes/               runtime dependency definitions
│   └── examples/               bundled example projects
└── tests/                      e2e suite (vitest + real Docker)
```

## Enforcement mechanisms

### 1. depguard (`.golangci.yml`)

Each layer has deny rules preventing upward imports:

- **Platform (L1-L2)**: may not import any domain or higher
- **Domain (L3)**: may not import project, compile, image, world
- **Project (L4)**: may not import compile, image, world
- **Build (L5)**: may not import world

Violation produces: `L3 (domain) must not import L5 (build)`. Failed lint blocks PRs.

### 2. Go `internal/`

Implementation details live under `internal/` — the Go compiler itself rejects wrong imports. Examples:
- `packages/project/internal/validate/` — rule engine, only reachable from `project`
- `packages/project/internal/manifest/` — parsing details
- `packages/project/internal/resolve/` — dep merging
- `packages/world/internal/runtime/` — runtime adapters
- `packages/world/internal/backend/` — Docker API wrapper

### 3. This document

The layer diagram above is the ground truth. When in doubt, check here.

## Core abstractions

| Abstraction | Where | Purpose |
|-------------|-------|---------|
| Dependency | `packages/deps` | Distribution unit (schema, refs, lockfile) |
| Tool | `packages/image` | Interface any installable capability implements |
| Runtime | `packages/compile/runtimes` | Translates agent composition → runtime files |
| Backend | `packages/image/backend` | Container runtime (Docker today) |
| Mind | `packages/agent` | How an agent persists and evolves |

## Data flow: `spwn up`

```
spwn.yaml       →  project.Load    →  project.Manifest
  + agents/**                         + []AgentRef

project.Manifest →  project/resolve  →  []string (merged deps)

deps             →  deps.Parse       →  *deps.Parsed (schema + files)

*deps.Parsed     →  image.ToolFromParsed → image.Tool

[]image.Tool     →  image.Build     →  Docker image

Docker image     →  compile.Render  →  compile.Tree (in-memory files)

compile.Tree     →  world.Spawn     →  running container + synced agents
                   (architect)
```

Every arrow crosses a layer boundary. Every layer has one job.

## Key invariants

- **Per-repository**. Agents and local blocks live in `./spwn/`, not `~/.spwn/`.
- **Declared deps only**. An agent can only use dependencies declared in `deps:`. Unlisted = physically absent.
- **External deps, local blocks**. `deps:` for `@spwn/*` and `github.com/*`. Local authoring in typed dirs: `spwn/skills/`, `spwn/tools/`, `spwn/hooks/`. See [`docs/dependencies.md`](dependencies.md).
- **Transitive resolution**. Dependency dependencies resolved recursively, topologically sorted.
- **Lock file is text**. `spwn.lock` — one dep per line, trivially diffable.
- **Labels are truth**. World info comes from Docker labels, not on-disk state.
- **Compile is deterministic**. Same input → same output, covered by golden tests.
- **Layers flow downward**. Enforced by depguard. No upward imports.

## Roadmap

- 🟢 Per-repository projects (`spwn init` / `check` / `build`)
- 🟢 World creation and isolation
- 🟢 Persistent agent identity and memory
- 🟢 Composable dependency catalog
- 🟢 Reproducible build artifacts
- 🟢 Layered package architecture with depguard enforcement
- 🟡 Agent evolution (dream, sleep, fork)
- 🟡 Multi-agent coordination via filesystem inboxes
- 🟡 Snapshots and rollback
- 🟡 Desktop app (Tauri) + web UI
- 🔴 GitHub-based dependency distribution (`github.com/owner/repo`)
- 🔴 Additional runtime adapters (Codex, Aider, OpenCode, Gemini CLI)
- 🔴 Additional backends (Firecracker, gVisor, K3s, Fly.io)
