# Spwn — Project Conventions

## Core Principle: The Building Blocks of Agent Intelligence

Spwn is the **operating system for autonomous agent worlds**. Compose tools, skills, and profiles into agents, then spawn them into isolated worlds where they wake up, find their tools, and get to work. Every layer is an interface (port) with swappable adapters. If a tool dies tomorrow, swap one adapter. Core logic never changes.

### The 8 Ports

| Port | What it abstracts | Default adapter |
|------|-------------------|-----------------|
| **Runtime** | How agents think | Claude Code (ACP) |
| **Provider** | Which LLM | Anthropic |
| **Channel** | How Architect talks to outside | CLI |
| **Backend** | Where worlds run | Docker |
| **Memory** | How agents persist | Filesystem (markdown) |
| **Store** | How state is tracked | JSON file |
| **Tool** | What agents can do | Built-in + MCP |
| **Skill** | Reusable capabilities | Local files |

## Vocabulary

### Entities
- **Agent**: A persistent mind. Composed from tools, skills, and a profile. Has identity, memory, and evolution history. The main thing you create and ship.
- **World**: A runtime instance. Ephemeral. Where an agent actually runs — Docker container with filesystem, tools, and lifecycle. Dies when stopped.
- **Architect**: The always-on orchestration daemon. Connected to all channels. Creates/destroys worlds. Self-manages via spwn.

### Building blocks (composable, reusable)
- **Tool**: A reusable tool pack (`@spwn/unix`, `@spwn/python`). Plugs into an agent as a capability. If not listed in an agent, it doesn't exist in its world.
- **Skill**: A reusable procedure, playbook, or piece of knowledge. Authored in markdown, shared across agents.
- **Profile**: A reusable personality template. Role, tone, purpose, behavior. Agents inherit a profile for their baseline personality.

### Agent internals
- **Identity**: Immutable core — purpose, traits, bonds. Never changes, even across forks.
- **Memory**: Journal, sessions, and knowledge. Persists across worlds, grows with experience.
- **Composition**: An agent's active tools + skills + profile declared in `agent.yaml`.

### Hierarchy (inside a world — "coming soon" on landing page)
- **Chief**: Lead agent inside a world. Decomposes tasks, delegates to workers, aggregates results.
- **Worker**: Persistent worker agent. Has its own identity and memory.
- **NPC**: Ephemeral agent. No persistent memory. Single task, fire & forget.

### Bridge
- **Gate**: Bridge between world and host. Host-side (Go) manages tool bridging. Container-side (Rivet) normalizes runtimes.
- **Rivet**: Runtime normalization layer. One API across all agent runtimes. Event streaming, session persistence.

### Evolution
- **Dream**: Analyze experience → discover patterns → promote successes to playbooks. `spwn agent dream <name>`
- **Sleep**: Graceful shutdown — save state, consolidate, prune. `spwn agent sleep <name>`
- **Fork**: Clone an agent with everything it knows. `spwn agent fork <src> <dst>`

## CLI Commands

**Grammar: `spwn <noun> <verb>`.** Three shortcuts for the 80% cases: `spwn up`, `spwn ls`, `spwn talk`.

```bash
# ── Shortcuts ─────────────────────────────────────────────────────
spwn up --agent neo -w ./my-project            # Spawn a world (alias for `spwn world up`)
spwn ls                                        # List active worlds (alias for `spwn world ls`)
spwn talk neo "what is this project?"          # Talk to an agent (alias for `spwn agent talk`)

# ── Agents (the composed mind) ────────────────────────────────────
spwn agent new neo                             # Create a blank agent
spwn agent new neo --from @community/sci       # Fork from a shared agent
spwn agent ls                                  # List agents
spwn agent show neo                            # Inspect composition
spwn agent rm neo                              # Delete agent (also: `rm neo --tool X` removes a block)
spwn agent fork neo neo-v2                     # Clone + evolve independently
spwn agent publish neo                         # Ship to registry (memory stripped)
spwn agent pull @community/curie               # Install a shared agent

# Compose capabilities onto an agent
spwn agent add neo --tool @spwn/python         # Add a tool block
spwn agent add neo --skill paper-reading       # Add a skill block
spwn agent add neo --profile researcher        # Apply a profile
spwn agent rm  neo --tool @spwn/python         # Remove a block

# Evolution
spwn agent dream neo                           # Analyze experience, promote playbooks
spwn agent sleep neo                           # Consolidate memory, prune stale strategies
spwn agent talk  neo "refactor auth"           # Full form of `spwn talk`

# ── Worlds (runtime instances) ────────────────────────────────────
spwn world up --agent neo -w ./project         # Full form of `spwn up`
spwn world ls                                  # Full form of `spwn ls`
spwn world inspect <id>                        # Inspect a running world
spwn world down <id>                           # Destroy world (agent survives)
spwn world enter <id>                          # Interactive shell inside the world

# ── Snapshots ─────────────────────────────────────────────────────
spwn snap save <id>                            # Save world state
spwn snap ls                                   # List snapshots
spwn snap restore <snap-id>                    # Rollback
spwn snap rm <snap-id>                         # Remove a snapshot

# ── Tools (composable blocks) ─────────────────────────────────────
spwn tool ls                                   # Installed tool packs
spwn tool show @spwn/python                    # Inspect a tool
spwn tool search python                        # Search the registry
spwn tool install @spwn/python                 # Install a tool pack
spwn tool rm @spwn/python                      # Uninstall
spwn tool publish ./my-tool                    # Ship to registry

# ── Skills (composable blocks) ────────────────────────────────────
spwn skill ls
spwn skill new paper-reading                   # Author a new skill
spwn skill edit paper-reading                  # Open in $EDITOR
spwn skill show paper-reading
spwn skill publish paper-reading
spwn skill install @community/rust-review
spwn skill rm paper-reading

# ── Profiles (composable blocks — personality templates) ──────────
spwn profile ls
spwn profile new researcher
spwn profile edit researcher
spwn profile show researcher
spwn profile publish researcher
spwn profile install @community/pragmatic-dev
spwn profile rm researcher

# ── Messaging ─────────────────────────────────────────────────────
spwn msg send neo --from morpheus "task"       # Inter-agent messaging
spwn msg ls neo                                # Neo's inbox
spwn msg show <msg-id>

# ── Architect (always-on orchestration daemon) ────────────────────
spwn architect start
spwn architect stop
spwn architect status
spwn architect connect <channel>

# ── Web UI ────────────────────────────────────────────────────────
spwn web                                       # Start + open in browser
spwn web --no-open --port 3002                 # Headless / custom port

# ── System ────────────────────────────────────────────────────────
spwn auth login / logout / token
```

**Design rules:**
- Strict noun-first grammar: `spwn <noun> <verb>`. Three shortcuts exist: `up`, `ls`, `talk`. No other top-level verbs.
- `rm` is contextual: `spwn agent rm neo` deletes the agent; `spwn agent rm neo --tool X` removes a block from it.
- Agent/tool/skill/profile names via positional args; flags for composition (`--tool`, `--skill`, `--profile`).
- Global flags: `-w` (workspace path for world spawning).

## IDs

- World: `w-{config-name}-{5digits}` (e.g. `w-default-84721`)
- Agent: `a-{agent-name}-{5digits}` (e.g. `a-leonardo-52103`)
- Generated with `crypto/rand`

## Config Paths

```
~/.spwn/
├── claw/
│   ├── state.json           # Active worlds, channels
│   └── claw.yaml            # Claw runtime config
├── worlds/
│   └── default.yaml         # World configs (tools available at runtime)
├── agents/
│   └── neo/
│       ├── agent.yaml       # Composition — tools, skills, profile, runtime
│       ├── profile.md       # Personality — role, style, purpose, behavior
│       ├── skills/          # Procedures, checklists
│       ├── knowledge/       # Facts, codebase info
│       ├── playbooks/       # Step-by-step workflows promoted from experience
│       └── journal/         # Session logs per world
├── tools/                   # Installed tool packs
├── skills/                  # Authored and installed skill files
└── profiles/                # Authored and installed profile templates
```

**Config hierarchy:** `agent.yaml` declares composition (tools + skills + profile + runtime). `world.yaml` declares the runtime environment. An agent runs in a world.

## Repository Structure

Multi-module Go monorepo + Turborepo-ready JS workspace:

```
spwn/
├── go.work                          # Go workspace
├── pnpm-workspace.yaml              # JS workspace (apps/, platform/)
├── turbo.json                       # Turborepo task orchestration
│
├── core/                            # Domain libraries (the product)
│   ├── universe/                    #   go.mod — world management
│   │   ├── universe.go              #   Public API (World, Manifest, Architect, Desktop App)
│   │   └── internal/
│   │       ├── architect/           #     Orchestration (spawn, destroy, list)
│   │       │   ├── colony.go        #       Multi-agent: SpawnAgents, Chief/Manager/Worker
│   │       │   └── npc.go           #       Ephemeral: SpawnNPC
│   │       ├── backend/             #     Docker adapter (Backend port)
│   │       ├── runtime/             #     Claude Code adapter (Runtime port)
│   │       ├── provider/            #     Anthropic + OpenAI adapters (Provider port)
│   │       ├── channel/             #     CLI adapter (Channel port)
│   │       ├── get/                  #     LocalRegistry adapter (Get/Marketplace port)
│   │       ├── api/                 #     HTTP API server (internal) (/api/worlds, /api/agents)
│   │       ├── sync/                #     Git config sync (SyncToGit, PullFromGit)
│   │       ├── physics/             #     Physics/faculties generation
│   │       ├── manifest/            #     Config parsing (world.yaml, agent.yaml)
│   │       ├── state/               #     Universe + Claw state (JSON)
│   │       ├── models/              #     Domain types (World, Manifest, Status, AgentRecord)
│   │       └── ports/               #     8 port interfaces (Runtime, Backend, Provider, etc.)
│   │
│   ├── agent/                       #   go.mod — agent management
│   │   ├── agent.go                 #   Public API (Info, InitProfile, Reflect, Sleep, Fork)
│   │   └── internal/
│   │       ├── profile/             #     Profile CRUD (init, validate, list, inspect, export)
│   │       ├── journal/             #     Episodic memory (append, list)
│   │       ├── session/             #     Session persistence (load, save)
│   │       ├── evolution/           #     Reflexion, Sleep, Forking
│   │       └── memory/              #     Filesystem adapter (Memory port)
│   │
│   ├── gate/                        #   go.mod — bridge protocol
│   │   ├── gate.go                  #   Public API (Server, Bridge, ExecHandler)
│   │   └── internal/
│   │       ├── bridge/              #     Wrapper scripts + capability enforcement
│   │       └── server/              #     HTTP-over-TCP gate server
│   │
│   ├── messenger/                   #   go.mod — agent-to-agent communication
│   │   ├── messenger.go             #   Public API (Send, Check, CheckUnread, MarkRead, ListAll)
│   │   └── internal/
│   │       ├── inbox/               #     Filesystem-based inbox read/write
│   │       └── models/              #     Message type
│   │
│   └── foundation/                  #   go.mod — cross-cutting primitives
│       ├── constants.go             #     Defaults, ProfileLayers, BaseImage
│       ├── paths.go                 #     BaseDir(), WorldsDir(), AgentsDir(), OrgPath()
│       ├── identity.go              #     GenerateWorldID(), GenerateAgentID()
│       └── names.go                 #     RandomCosmosWord(), RandomAgentName()
│
├── apps/                            # Deployable consumers
│   ├── cli/                         #   go.mod — the spwn binary
│   │   ├── cmd/spwn/main.go         #     Entry point
│   │   ├── root.go                  #     Root cobra command
│   │   ├── defaults.go              #     Auto-create defaults on first run
│   │   ├── world/                   #     World subcommands (up, down, ls, logs, enter, inspect)
│   │   ├── agent/                   #     Agent subcommands (new, ls, rm, talk, fork, export, import)
│   │   ├── profile/                 #     Profile subcommands (full character sheet)
│   │   ├── msg/                     #     Messaging subcommands (send, inbox, watch)
│   │   ├── snap/                    #     Snapshot subcommands (save, ls, restore, rm)
│   │   ├── architect/                #     Architect subcommands (start, stop, status, connect)
│   │   ├── get/                     #     Marketplace subcommands (install, ls, search, rm)
│   │   ├── auth/                    #     Auth subcommands (login, logout, token)
│   │   ├── ui/                      #     Stepper, table, style, format
│   │   └── tests/
│   │       └── integration/         #   Cross-domain flows (world + agent)
│   │
│   └── dash/                        #   Visual dashboard (CLI placeholder, Next.js planned)
│       └── package.json
│
├── platform/                        # Build infrastructure
│   ├── images/                      #   Docker images
│   │   ├── Dockerfile               #     spwn/world production image
│   │   ├── Dockerfile.test          #     Test image with mock Claude
│   │   └── embed.go                 #     go:embed for runtime auto-build
│   ├── gate-runtime/                #   Container-side gate (Rust)
│   │   ├── Cargo.toml
│   │   └── src/main.rs
│   └── fixtures/                    #   Test fixtures
│       ├── mock-claude/             #     Mock Claude binary for E2E
│       └── testdata/                #     Shared test data
│
├── Makefile
├── README.md
└── CLAUDE.md                        # (this file)
```

## Container Architecture: Docker-outside-of-Docker (DooD)

spwn uses **DooD (Docker-outside-of-Docker)**, not DinD (Docker-in-Docker). The host's Docker daemon is shared via socket mount (`/var/run/docker.sock`). All containers are **siblings** on the same daemon — no nesting, no privilege escalation, no performance overhead.

```
Host machine
└── Docker daemon (/var/run/docker.sock)
    ├── Architect container (always-on, socket-mounted)
    ├── World containers (siblings, created by Architect)
    └── Desktop App container (sibling)
```

**Two modes:**
- **Local CLI (direct)** — `spwn up` calls Docker directly from the host. No Architect container needed.
- **Hosted Architect (containerized)** — `spwn architect start` launches the Architect in a long-lived container with the Docker socket mounted. It creates/manages world containers as siblings. Channels connect here.

## Dependency Graph

```
apps/cli ──→ core/universe, core/agent, core/gate, core/messenger, core/foundation
core/universe ──→ core/agent, core/gate, core/foundation
core/agent ──→ core/foundation
core/gate ──→ core/foundation
core/messenger ──→ core/foundation
```

5 Go modules + `platform/images`. Each `core/` module exposes a public API in its root `.go` file. Adapters (runtime, provider, channel, skill, etc.) live inside `core/universe/internal/` — private per module.

## Code Style

- No cgo
- Errors: `error: lowercase message.\nActionable hint.`
- **Ports & Adapters everywhere** — every external dependency goes through an interface (port). Adapters are swappable.
- Domain modules own all business logic — CLI is a thin wrapper (parse flags → call domain API → format output)
- Backend is a port — Docker is just one adapter. No direct Docker calls outside the backend adapter.
- Types avoid stutter: `universe.World` not `universe.Universe`, `agent.Info` not `agent.AgentInfo`, `gate.Bridge` not `gate.GateBridge`
- Package name provides context — don't repeat it in type names

## Build

```bash
make build               # cd apps/cli && go build -o ../../bin/spwn ./cmd/spwn
make build-image         # docker build spwn/world:latest from platform/images/
make build-test-image    # docker build spwn-test:latest for E2E
make build-gate          # cd platform/gate-runtime && cargo build --release

make test                # Unit tests across all modules
make test-foundation     # cd core/foundation && go test -v ./...
make test-universe       # cd core/universe && go test -v ./...
make test-agent          # cd core/agent && go test -v ./...
make test-gate           # cd core/gate && go test -v ./...
make test-cli            # cd apps/cli && go test -v ./...

make test-e2e            # All integration/E2E tests
make test-e2e-universe   # Universe integration (Docker required)
make test-e2e-agent      # Agent integration (filesystem only)

make lint                # go vet across all modules
make clean               # rm -rf bin/
```

## Testing Strategy

Three-layer pyramid:

| Layer | Location | Speed | Infra |
|-------|----------|-------|-------|
| **Unit** | `*_test.go` next to source files | ~1s | None |
| **E2E (Go)** | `core/universe/tests/e2e/` | ~30s | Docker |
| **E2E (TS)** | `tests/e2e/` | ~2min | Built binary |

Each domain tests only its own contract. Cross-domain flows (spawn universe + agent → verify journal) are the CLI's responsibility.

## Development Methodology: Spec-First

Spwn follows a **spec-first** development process:

1. **Specify** — Define behavior in the knowledge (what the system SHOULD do)
2. **Encode** — Write tests that encode those specs (they fail initially)
3. **Implement** — Write code that makes the tests pass
4. **Verify** — The test suite IS the living specification

The E2E test suite is the behavioral specification of spwn. Each test describes a user-visible behavior:

```go
// GIVEN a world with a chief and two workers
// WHEN the chief delegates a task
// THEN both workers receive work
// AND the chief aggregates results
```

### Test layers:
- **Behavioral specs** (`core/universe/tests/e2e/`, `tests/e2e/`) — what the system does (the specification)
- **CLI specs** (`apps/cli/cli_test.go`) — what the user sees (flag parsing, help, output)
- **Unit tests** (`*_test.go` next to source) — how the code works (implementation details)

The behavioral specs are the source of truth. If a spec fails, the implementation is wrong — not the spec.
