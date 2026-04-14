# Spwn - Project Conventions

## Core Principle: The Building Blocks of Agent Intelligence

Spwn is the **operating system for autonomous agent worlds**. Compose tools, skills, and profiles into agents, then spawn them into isolated worlds where they wake up, find their tools, and get to work.

The domain has three main abstractions, each owning one concern:

| Abstraction | What it owns | Implementation |
|---|---|---|
| **Runtime** | How an agent actually runs (CLI invocation, session capture) | `packages/world/internal/runtime` - Claude Code today, others swap in as a ~50 LOC Go file |
| **Backend** | Where worlds run | `packages/world/internal/backend` - Docker; container labels are the source of truth for world state |
| **Mind** | How an agent persists across worlds | `packages/agent` - flat markdown layers (core/skills/knowledge/playbooks/journal) on the host filesystem |

## Vocabulary

### Entities
- **Agent**: A persistent mind. Composed from tools, skills, and a profile. Has identity, memory, and evolution history. The main thing you create and ship.
- **World**: A runtime instance. Ephemeral. Where an agent actually runs - Docker container with filesystem, tools, and lifecycle. Dies when stopped.
- **Architect**: The always-on orchestration daemon. Connected to all channels. Creates/destroys worlds. Self-manages via spwn.

### Building blocks (composable, reusable)
- **Tool**: A reusable tool pack (`@spwn/unix`, `@spwn/python`). Plugs into an agent as a capability. If not listed in an agent, it doesn't exist in its world.
- **Skill**: A reusable procedure, playbook, or piece of knowledge. Authored in markdown, shared across agents.
- **Profile**: A reusable personality template. Role, tone, purpose, behavior. Agents inherit a profile for their baseline personality.

### Agent internals
- **Identity**: Immutable core - purpose, traits, bonds. Never changes, even across forks.
- **Memory**: Journal, sessions, and knowledge. Persists across worlds, grows with experience.
- **Composition**: An agent's active tools + skills + profile declared in `agent.yaml`.

### Hierarchy (inside a world - "coming soon" on landing page)
- **Chief**: Lead agent inside a world. Decomposes tasks, delegates to workers, aggregates results.
- **Worker**: Persistent worker agent. Has its own identity and memory.
- **NPC**: Ephemeral agent. No persistent memory. Single task, fire & forget.

### Evolution
- **Dream**: Analyze experience → discover patterns → promote successes to playbooks. `spwn agent dream <name>`
- **Sleep**: Graceful shutdown - save state, consolidate, prune. `spwn agent sleep <name>`
- **Fork**: Clone an agent with everything it knows. `spwn agent fork <src> <dst>`

## CLI Commands

**Grammar: `spwn <noun> <verb>`.** Three shortcuts for the 80% cases: `spwn up`, `spwn ls`, `spwn talk`. Full status matrix lives in the README; this is the reference map.

```bash
# ── Project workflow ─────────────────────────────────────────────
spwn init                                      # Scaffold spwn.yaml + ./spwn/ + .spwn/
spwn check                                     # Validate the tree (15 rules)
spwn build                                     # Flatten to .spwn/build/ (pinned artifact)
spwn up --build                                # Build then spawn from the artifact

# ── Shortcuts ────────────────────────────────────────────────────
spwn up                                        # Spawn the world declared in spwn.yaml
spwn ls                                        # List active worlds
spwn talk neo "what is this project?"          # Talk to an agent

# ── Agents ───────────────────────────────────────────────────────
spwn agent new neo                             # Create a blank agent in ./spwn/agents/
spwn agent ls                                  # List project agents
spwn agent show neo                            # Inspect composition
spwn agent fork neo neo-v2                     # Clone memory + composition
spwn agent rm neo                              # Delete an agent
spwn agent rm neo --tool @spwn/python          # Remove a block from an agent

# Compose blocks
spwn agent add neo --tool @spwn/python         # Add a tool block
spwn agent add neo --skill paper-reading       # Add a skill block
spwn agent add neo --profile researcher        # Apply a profile

# Talk + messaging
spwn agent talk  neo "refactor auth"           # Full form of `spwn talk`
spwn agent send  neo "do this" --from morpheus # Async message to an agent's inbox
spwn agent inbox neo                           # Show neo's inbox
spwn agent watch neo                           # Tail neo's inbox live

# Evolution
spwn agent dream neo                           # Analyze experience, promote playbooks
spwn agent sleep neo                           # Consolidate memory, prune stale patterns

# ── Worlds ───────────────────────────────────────────────────────
spwn world up                                  # Full form of `spwn up`
spwn world ls                                  # Full form of `spwn ls`
spwn world inspect <id>                        # Inspect a running world
spwn world down <id>                           # Destroy (agent survives)
spwn world enter <id>                          # Interactive shell inside the world
spwn world snap save|ls|restore|rm             # World snapshots

# ── Tools / skills / profiles ────────────────────────────────────
spwn tool    ls                                # Installed built-in packs
spwn skill   ls|new|edit                       # Authored skills in ./spwn/skills/
spwn profile ls|new|edit                       # Authored profiles in ./spwn/profiles/

# ── Registry (planned) ───────────────────────────────────────────
spwn agent   get @community/sci                # Install a shared agent     [planned]
spwn tool    get @community/rust-fuzzer        # Install a shared tool pack [planned]
spwn skill   get @community/rust-review        # Install a shared skill     [planned]
spwn *       publish <name>                    # Push to registry           [planned]

# ── System ───────────────────────────────────────────────────────
spwn architect start|stop|status|talk|logs     # Always-on orchestration daemon
spwn web                                       # Open the local web UI
spwn auth login|logout|token                   # Provider credentials
```

**Design rules:**
- Strict noun-first grammar: `spwn <noun> <verb>`. Three shortcuts exist: `up`, `ls`, `talk`. No other top-level verbs.
- `rm` is contextual: `spwn agent rm neo` deletes the agent; `spwn agent rm neo --tool X` removes a block from it.
- Inside a project, commands resolve against `./spwn/` first. Outside a project, they operate on user-level paths (legacy).

## IDs

- World: `spwn-world-{planet}-{5digits}` (e.g. `spwn-world-rhea-84721`)
- Agent: `a-{name}-{5digits}` (e.g. `a-neo-52103`)
- Generated with `crypto/rand`.

## Config layout (per-repo)

A spwn project is **in the repo**, not in your home directory. `~/.spwn/` holds user-level credentials and daemon state only.

```
my-project/
├── spwn.yaml                    # manifest - version, name, world, agents
├── spwn/                        # committed project assets
│   ├── agents/
│   │   └── neo/
│   │       ├── agent.yaml       # composition: tools + skills + profile + runtime
│   │       ├── CLAUDE.md        # entry point the runtime reads on startup
│   │       ├── core/profile.md  # identity
│   │       ├── skills/          # authored procedures
│   │       ├── knowledge/       # facts, codebase notes
│   │       ├── playbooks/       # promoted patterns (via dream)
│   │       └── journal/         # per-run history
│   ├── worlds/
│   │   └── default.yaml         # physics (cpu/memory/disk/timeout) + tools
│   ├── skills/                  # project-scoped skill files
│   ├── profiles/                # project-scoped personality templates
│   └── tools/                   # project-scoped tool packs (optional)
└── .spwn/                       # gitignored local state
    ├── state.json               # live world IDs bound to this project
    ├── build/                   # `spwn build` output (see packages/manifest)
    └── cache/
```

```
~/.spwn/                         # USER-LEVEL only, not per-project
├── credentials/                 # auth material surfaced to containers at /credentials
├── activity.jsonl               # global activity log
└── state/                       # architect daemon state
```

**Config hierarchy:** `agent.yaml` declares composition (tools + skills + profile + runtime). `worlds/<name>.yaml` declares the runtime environment (physics + tools). `spwn.yaml` at project root wires which world runs which agents.

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
│   │   ├── profile/                 #     spwn profile
│   │   ├── skill/                   #     spwn skill
│   │   ├── tool/                    #     spwn tool
│   │   ├── team/                    #     spwn team
│   │   ├── example/                 #     spwn example
│   │   ├── organization/            #     spwn organization
│   │   ├── get/                     #     spwn get (install from marketplace)
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
apps/cli  ──→ packages/{world, agent, messenger, imagebuilder, migration, foundation}
packages/world ──→ packages/{agent, imagebuilder, foundation}
packages/agent ──→ packages/foundation
packages/messenger ──→ packages/foundation
packages/imagebuilder ──→ (no spwn deps)
packages/migration ──→ packages/foundation
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
make test-foundation     # cd packages/foundation && go test -v ./...
make test-world          # cd packages/world && go test -v ./...
make test-agent          # cd packages/agent && go test -v ./...
make test-cli            # cd apps/cli && go test -v ./...

make test-e2e            # Go E2E against Docker
make test-e2e-world      # Same, explicit alias
make test-ui             # Playwright UI E2E (Docker + browser)

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
