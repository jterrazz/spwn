# Spwn - Project Conventions

## Core Principle: The Building Blocks of Agent Intelligence

Spwn is the **operating system for autonomous agent worlds**. Compose tools, skills, and identity into agents, then spawn them into isolated worlds where they wake up, find their tools, and get to work.

The domain has three main abstractions, each owning one concern:

| Abstraction | What it owns | Implementation |
|---|---|---|
| **Runtime** | How an agent actually runs (CLI invocation, session capture, credential plumbing) | `packages/runtimes` - Claude Code today, codex next, others plug in as a ~50 LOC Adapter |
| **Backend** | Where worlds run | `packages/container/backend` - Docker; container labels are the source of truth for world state |
| **Mind** | How an agent persists across worlds | `packages/agent` - flat markdown layers (skills/playbooks/journal) on the host filesystem plus a single `SOUL.md` at the agent root. Knowledge is world-scoped, not in the Mind — declare a host path via `worlds.<name>.knowledge` in `spwn.yaml` (e.g. `./knowledge`) and it gets bind-mounted into `/world/knowledge/`. Omit the key to spawn a world whose agents are never told a knowledge base exists. |

## Vocabulary

### Entities
- **Agent**: A persistent mind. Composed from tools, skills, and an identity. Has memory and evolution history. The main thing you create and ship.
- **World**: A runtime instance. Ephemeral. Where an agent actually runs - Docker container with filesystem, tools, and lifecycle. Dies when stopped.
- **Architect**: The always-on orchestration daemon. Connected to all channels. Creates/destroys worlds. Self-manages via spwn.

### Building blocks (composable, reusable)
- **Dependency**: The distribution unit. A `spwn.yaml` manifest (catalog or GitHub repo) that ships any combination of tools, skills, hooks, and agents. Installed via `spwn install`, pinned in `spwn.lock`. Agents reference them as external deps.
- **Skill (bare form)**: A `spwn/skills/<name>.md` file. Simplest authoring path for "write a paragraph of instructions."

### Agent internals
- **Soul**: Who the agent is - purpose, voice, values, in a single file at `spwn/agents/<name>/SOUL.md`. Persists across world restarts. (Formerly split across `identity/profile.md`, `purpose.md`, `traits.md`; collapsed in 2026-04.)
- **Memory**: Journal and sessions. Persists across worlds, grows with experience. (Knowledge is world-scoped, not agent-scoped.)
- **Composition**: An agent's active dependencies (tools, skills, hooks), declared as a unified `dependencies:` list in `agent.yaml` using the `spwn:`/`skill:`/`tool:`/`hook:` schemes.

### Hierarchy (inside a world - "coming soon" on landing page)
- **Chief**: Lead agent inside a world. Decomposes tasks, delegates to workers, aggregates results.
- **Worker**: Persistent worker agent. Has its own identity and memory.
- **NPC**: Ephemeral agent. No persistent memory. Single task, fire & forget.

### Evolution
- **Dream**: Analyze experience → discover patterns → promote successes to playbooks. `spwn agent dream <name>`
- **Sleep**: Graceful shutdown - save state, consolidate, prune. `spwn agent sleep <name>`
- **Fork**: Clone an agent with everything it knows. `spwn agent fork <src> <dst>`

## CLI Commands

**Grammar: `spwn <noun> <verb>`** plus compose-style shortcuts
(`spwn up`, `spwn ls`, `spwn down`) and name-only shortcuts
(`spwn agent neo`, `spwn world default`). With no args, the
shortcuts act on every world declared in `spwn.yaml`.

```bash
# ── Project workflow ─────────────────────────────────────────────
spwn init                                      # Scaffold spwn.yaml + ./spwn/ + .spwn/
spwn check                                     # Validate the tree
spwn build --tree-only                         # Render the project tree to ./dist (preview/debug)
spwn build                                     # Transpile + compile into a project-specific Docker image
spwn up                                        # Spawn a world from the current project

# ── Compose-style shortcuts ──────────────────────────────────────
spwn up                                        # Bring up every world in spwn.yaml
spwn up default                                # Bring up one world by name
spwn agent neo                                 # Start the world that contains neo
spwn ls                                        # Agent-centric status (running/stopped/orphan)
spwn down                                      # Stop every world

# ── Agents ───────────────────────────────────────────────────────
spwn agent new neo                             # Create a blank agent in ./spwn/agents/
spwn agent ls                                  # List project agents
spwn agent inspect neo                         # Inspect composition, memory, history
spwn agent fork neo neo-v2                     # Clone memory + composition
spwn agent rm neo                              # Delete an agent

# Compose (via the project-level install / uninstall verbs)
spwn install python                            # Catalog dep, every agent
spwn install python --agent neo                # Catalog dep, only neo
spwn install skill:paper-reading --agent neo   # Local skill, only neo
spwn install tool:ffmpeg --agent neo           # Local tool, only neo
spwn install hook:pre-spawn --agent neo        # Local hook, only neo
spwn uninstall python --agent neo              # Detach from one agent

# Talk + messaging
spwn agent talk  neo "refactor auth"           # Full form of `spwn talk`
spwn agent send  neo "do this" --from morpheus # Async message to an agent's inbox
spwn agent inbox neo                           # Show neo's inbox
spwn agent watch neo                           # Tail neo's inbox live

# Evolution
spwn agent dream neo                           # Analyze experience, promote playbooks
spwn agent sleep neo                           # Consolidate memory, prune stale patterns

# ── Worlds ───────────────────────────────────────────────────────
# Worlds are inline map entries in spwn.yaml#worlds; there is no
# spwn/worlds/ directory any more.
spwn world start [name]                        # Start a world (no arg: every world in spwn.yaml)
spwn world stop  [name]                        # Stop a world
spwn world ls                                  # List running worlds
spwn world inspect <id>                        # Inspect a running world
spwn world enter   <id>                        # Interactive shell inside the world
spwn world snap save|ls|restore|rm             # World snapshots

# ── Dependencies ────────────────────────────────────────────
spwn install spwn:python              # Install a dep (adds to every agent + lockfile)
spwn uninstall spwn:python          # Remove a dep

spwn skill   new|edit|show|rm <name>           # Bare-markdown skill authoring (./spwn/skills/<name>.md)

# ── Registry (planned) ───────────────────────────────────────────
spwn agent   get github:community/sci          # Install a shared agent     [planned]
spwn install github:acme/fuzzer                # Install from GitHub [planned]
spwn *       publish <name>                    # Push to registry           [planned]

# ── System ───────────────────────────────────────────────────────
spwn architect start|stop|status|talk|logs     # Always-on orchestration daemon
spwn web                                       # Open the local web UI
spwn auth login|logout|token                   # Provider credentials
```

**Design rules:**
- Strict noun-first grammar: `spwn <noun> <verb>`. Three shortcuts exist: `up`, `ls`, `talk`. No other top-level verbs.
- `rm` is contextual: `spwn agent rm neo` deletes the agent; `spwn agent rm neo --dependency X` removes a dep from it.
- Inside a project, commands resolve against `./spwn/` first. Outside a project, they operate on user-level paths (legacy).

## IDs

- World: `world-{planet}-{5digits}` (e.g. `world-rhea-84721`)
- Agent: `agent-{name}-{5digits}` (e.g. `agent-neo-52103`)
- Generated with `crypto/rand`.

## Config layout (per-repo)

A spwn project is **in the repo**, not in your home directory. `~/.spwn/` holds user-level credentials and daemon state only.

```
my-project/
├── spwn.yaml                    # manifest - version, name, inline worlds map
├── spwn/                        # committed project assets
│   ├── agents/
│   │   └── neo/
│   │       ├── agent.yaml       # composition: dependencies + runtime.backend
│   │       ├── AGENTS.md         # entry point (provider-neutral, compiled per runtime)
│   │       ├── SOUL.md          # who the agent is (one file: purpose, voice, values)
│   │       ├── skills/          # Mind memory layer (runtime-written, opaque to spwn - no discovery or auto-injection)
│   │       ├── playbooks/       # promoted patterns (via dream)
│   │       └── journal/         # per-run history
│   ├── worlds/
│   │   └── neo/
│   │       └── knowledge/       # world-scoped facts, bind-mounted to /world/knowledge/
│   ├── skills/                  # project-scoped skill files
│   └── tools/                   # project-scoped tool dependencies (optional)
└── .spwn/                       # gitignored local state
    ├── state.json               # live world IDs bound to this project
    └── cache/
```

Worlds are declared **inline** under `spwn.yaml#worlds` - the
world record (agents, workspaces, tool overrides) lives in yaml,
not in separate yaml files. A world optionally owns one filesystem
artifact: the directory referenced by its `knowledge:` key (e.g.
`./knowledge`), which gets bind-mounted at `/world/knowledge/` inside
the running container. Omit the key and no mount happens — the agent
is never told a knowledge base exists. Each world entry names the
agents it deploys, the workspace mounts, and optional tool
overrides.

```
~/.spwn/                         # USER-LEVEL only, not per-project
├── credentials/                 # auth material surfaced to containers at /credentials
├── activity.jsonl               # global activity log
└── state/                       # architect daemon state
```

**Config hierarchy:** `agent.yaml` declares composition via a unified `dependencies:` list (`spwn:<name>` for catalog deps; `skill:<name>` / `tool:<name>` / `hook:<name>` for local blocks authored under `spwn/skills/`, `spwn/tools/`, `spwn/hooks/`) plus `runtime.backend`. `spwn.yaml#worlds[<name>]` declares the runtime environment (agents + workspaces). The union of project-wide and agent-specific dependencies is what actually materializes inside the container.

## Repository Structure

Polyglot monorepo: Go modules + Next.js/Tauri web UI, wired together
with Go workspaces, pnpm, and a top-level Makefile.

```
spwn/
├── go.work                          # Go workspace
├── pnpm-workspace.yaml              # JS workspace (apps/*, tests)
├── Makefile                         # Single entry point for Go + JS tasks
│
├── apps/                            # End-user binaries
│   ├── cli/                         #   go.mod - the `spwn` binary
│   │   ├── cmd/spwn/main.go         #     Entry point
│   │   ├── root.go                  #     Root cobra command
│   │   ├── world/                   #     spwn world (up, down, ls, inspect, logs, enter)
│   │   ├── agent/                   #     spwn agent (new, ls, rm, talk, fork, export…)
│   │   ├── snap/                    #     spwn world snap (save, ls, restore, rm)
│   │   ├── architect/               #     spwn architect (start, stop, status)
│   │   ├── web/                     #     spwn web (launches the web UI)
│   │   ├── auth/                    #     spwn auth (login, logout, token)
│   │   ├── skill/                   #     spwn skill (bare-markdown authoring)
│   │   ├── team/                    #     spwn team
│   │   ├── organization/            #     spwn organization
│   │   ├── logs/                    #     spwn logs
│   │   └── ui/                      #     Stepper, table, style
│   │
│   └── web/                         #   Next.js + Tauri desktop/web app
│       ├── src/                     #     Next.js app (React)
│       └── src-tauri/               #     Tauri shell (Rust)
│
├── packages/                        # Go domain modules (shared libraries)
│   ├── world/                       #   go.mod - world lifecycle (the core)
│   │   ├── world.go                 #   Public API (World, Manifest, Architect…)
│   │   └── internal/
│   │       ├── architect/           #     Orchestration (spawn, destroy, deploy)
│   │       ├── backend/             #     Docker adapter
│   │       ├── runtime/             #     Claude Code runtime
│   │       ├── api/                 #     HTTP API server (consumed by apps/web)
│   │       ├── physics/             #     physics.md / faculties.md generation
│   │       ├── manifest/            #     Config parsing (world.yaml, agent.yaml)
│   │       ├── labels/              #     Container labels as source of truth
│   │       ├── state/               #     State hydrated from labels
│   │       ├── runtimestate/        #     Mutable runtime state (sessions, agents)
│   │       └── models/              #     Domain types
│   │
│   ├── agent/                       #   go.mod - agent/mind management
│   │   ├── agent.go                 #   Public API (InitMind, Validate, Export, Fork…)
│   │   └── internal/{mind,journal,session,evolution,memory}/
│   │
│   ├── messenger/                   #   go.mod - agent-to-agent messaging
│   │
│   ├── imagebuilder/                #   go.mod - composable tool-based image builder
│   │
│   ├── migration/                   #   go.mod - ~/.spwn schema migrations
│   │
│   └── foundation/                  #   go.mod - cross-cutting primitives
│       ├── constants.go             #     Defaults, directory layout, mind layers
│       ├── paths.go                 #     BaseDir(), WorldsDir(), AgentsDir()
│       ├── identity.go              #     GenerateWorldID(), GenerateAgentID()
│       ├── auth/                    #     Credential resolution
│       ├── activity/                #     Activity log
│       └── update/                  #     Self-update logic
│
├── examples/                        # Shipped example worlds
├── fixtures/                        # Test fixtures
│   ├── Dockerfile.test              #   Mock-claude test image
│   ├── mock-claude/                 #   Bash script standing in for claude CLI
│   └── testdata/                    #   Shared fixtures
├── tests/                           # TypeScript vitest E2E suite
│   ├── e2e/                         #   Behavioral specs against the compiled binary
│   ├── setup/                       #   Test harness (world-assertion, state-assertion…)
│   └── ui/                          #   Playwright specs for the web UI
├── docs/                            # Prose docs (architecture, releasing, CLI man pages)
│
├── Makefile
├── README.md
└── CLAUDE.md                        # (this file)
```

## Container Architecture: Docker-outside-of-Docker (DooD)

spwn uses **DooD (Docker-outside-of-Docker)**, not DinD (Docker-in-Docker). The host's Docker daemon is shared via socket mount (`/var/run/docker.sock`). All containers are **siblings** on the same daemon - no nesting, no privilege escalation, no performance overhead.

```
Host machine
└── Docker daemon (/var/run/docker.sock)
    ├── Architect container (always-on, socket-mounted)
    ├── World containers (siblings, created by Architect)
    └── Desktop App container (sibling)
```

**Two modes:**
- **Local CLI (direct)** - `spwn up` calls Docker directly from the host. No Architect container needed.
- **Hosted Architect (containerized)** - `spwn architect start` launches the Architect in a long-lived container with the Docker socket mounted. It creates/manages world containers as siblings. Channels connect here.

## Dependency Graph

```
apps/cli  ──→ packages/{world, agent, mailbox, image, migration, base, project}
packages/world ──→ packages/{agent, image, base}
packages/agent ──→ packages/base
packages/mailbox ──→ packages/base
packages/compile ──→ (no spwn deps)
packages/migration ──→ packages/base
packages/project ──→ (no spwn deps)
```

Each `packages/` module exposes a public API in its root `.go` file.
Implementation details live under `internal/`.

## Code Style

- No cgo
- Errors: `error: lowercase message.\nActionable hint.`
- Domain modules own all business logic - CLI is a thin wrapper
  (parse flags → call domain API → format output)
- Types avoid stutter: `world.World` not `world.WorldInstance`,
  `agent.Info` not `agent.AgentInfo`. Package name provides context.

## Build

```bash
make build               # cd apps/cli && go build -o ../../bin/spwn ./cmd/spwn
make build-test-image    # docker build spwn-test:latest for E2E

make test                # Unit tests across all modules
make test-foundation     # cd packages/base && go test -v ./...
make test-world          # cd packages/world && go test -v ./...
make test-agent          # cd packages/agent && go test -v ./...
make test-cli            # cd apps/cli && go test -v ./...

make test-e2e            # Go E2E against Docker
make test-e2e-world      # Same, explicit alias
make test-web            # Playwright web E2E (Docker + browser)

make lint                # go vet across all modules
make clean               # rm -rf bin/
```

## Testing Strategy

Three-layer pyramid:

| Layer | Location | Speed | Infra |
|-------|----------|-------|-------|
| **Unit** | `*_test.go` next to source files | ~1s | None |
| **E2E (Go)** | `packages/world/tests/e2e/` | ~30s | Docker |
| **E2E (TS)** | `tests/e2e/` | ~2min | Built binary |

Each domain tests only its own contract. Cross-domain flows (spawn universe + agent → verify journal) are the CLI's responsibility.

## Development Methodology: Spec-First

Spwn follows a **spec-first** development process:

1. **Specify** - Define behavior in the knowledge (what the system SHOULD do)
2. **Encode** - Write tests that encode those specs (they fail initially)
3. **Implement** - Write code that makes the tests pass
4. **Verify** - The test suite IS the living specification

The E2E test suite is the behavioral specification of spwn. Each test describes a user-visible behavior:

```go
// GIVEN a world with a chief and two workers
// WHEN the chief delegates a task
// THEN both workers receive work
// AND the chief aggregates results
```

### Test layers:
- **Behavioral specs** (`packages/world/tests/e2e/`, `tests/e2e/`) - what the system does (the specification)
- **CLI specs** (`apps/cli/cli_test.go`) - what the user sees (flag parsing, help, output)
- **Unit tests** (`*_test.go` next to source) - how the code works (implementation details)

The behavioral specs are the source of truth. If a spec fails, the implementation is wrong - not the spec.
