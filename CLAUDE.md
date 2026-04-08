# Spwn — Project Conventions

## Core Principle: The Control Plane for AI Agents

Spwn is the **control plane for AI agents** — bringing order to agent chaos. Isolated worlds, persistent identity, physics-based security, multi-agent collaboration. Every layer is an interface (port) with swappable adapters. If a tool dies tomorrow, swap one adapter. Core logic never changes.

### The 8 Ports

| Port | What it abstracts | Default adapter |
|------|-------------------|-----------------|
| **Runtime** | How agents think | Claude Code (ACP) |
| **Provider** | Which LLM | Anthropic |
| **Channel** | How Architect talks to outside | CLI |
| **Backend** | Where universes run | Docker |
| **Memory** | How profiles persist | Filesystem (markdown) |
| **Store** | How state is tracked | JSON file |
| **Tool** | What agents can do | Built-in + MCP |
| **Skill** | Reusable capabilities | Local files |

## Vocabulary

### The Hierarchy
- **Organization**: Top-level manifest (org.yaml). Org-wide defaults, shared skills, governance, config sync.
- **Architect**: The always-on orchestration daemon (ZeroClaw implementation). Connected to all channels. Creates/destroys worlds. Self-manages via spwn.
- **Universe**: The reality — physics, constants, resource limits. One per org. Configured in `universe.yaml`. Defines what is physically possible.
- **World**: A living workspace inside the universe. Has agents, elements, and a project. Many per universe. Configured in `~/.spwn/worlds/`.
- **Leader**: Lead agent inside a world. Decomposes tasks, delegates to workers, aggregates results.
- **Worker**: Persistent worker agent. Has a Profile — remembers, learns, evolves.
- **Ephemeral**: Ephemeral agent. No Profile, no memory. Single task, fire & forget.
- **Observatory**: Visual dashboard. Real-time view of everything.

### The Physics
- **Physics**: Constants (CPU, memory, timeout), laws (network, max-processes), elements (@spwn/unix, @spwn/git, jq).
- **Elements**: Building blocks. @packs expand to collections. If not listed, doesn't exist.
- **Faculties**: Verified elements + gate bridges, auto-generated as `/world/faculties.md`.

### The Profile
- **Profile**: Persistent identity — persona, traits, purpose, bonds, skills, memory (knowledge, playbooks, journal), sessions.
- **Identity**: Core character — persona, purpose, traits. Lives in `identity/` directory.
- **Profile Manifest**: `profile.yaml` — declares role, engine, identity, requires, delegation.

### The Bridge
- **Gate**: Bridge between universe and host. Host-side (Go) manages element bridging. Container-side (Rivet) normalizes runtimes.
- **Rivet**: Runtime normalization layer. One API across all agent runtimes. Event streaming, session persistence.

### Evolution
- **Dream**: Analyze experience → discover patterns → promote successes to playbooks (auto-reflexion.md). `spwn agent dream <name>`
- **Sleep**: Graceful shutdown — save state, consolidate, prune. `spwn agent sleep <name>`
- **Forking**: Clone an agent from source to target. `spwn agent fork`

## CLI Commands

```bash
# World operations (top-level)
spwn up --agent neo -w ./my-project --detach   # Spawn a world
spwn up -c acme-org                            # Named config
spwn up --leader morpheus -w ~/acme-org
spwn ls                                        # List active worlds
spwn inspect <id>                              # Show world details
spwn down <id>                                 # Destroy a world
spwn logs <id>                                 # Stream agent output
spwn attach <id>                               # Interactive shell

# Agent management — "Profile is the passport. Agent is the person."
spwn agent new <name>                          # Create a new agent
spwn agent ls                                  # List all agents
spwn agent rm <name>                           # Remove an agent
spwn agent talk <name> [message]               # Talk to a running agent
spwn agent dream <name>                        # Analyze experience, discover patterns, promote playbooks
spwn agent sleep <name>                        # Shutdown — save state, consolidate, archive
spwn agent fork <src> <dst>                    # Clone an agent
spwn agent export <name>                       # Export agent as tar.gz
spwn agent import <file>                       # Import agent from tar.gz

# Profile (character sheet — the passport, not the person)
spwn profile <name>                            # Show full character sheet
spwn profile <name> purpose                    # Show/edit purpose
spwn profile <name> traits                     # Show/edit traits
spwn profile <name> persona                    # Show/edit persona
spwn profile <name> bonds                      # Show/edit bonds
spwn profile <name> skills                     # List skills
spwn profile <name> playbooks                  # List playbooks
spwn profile <name> knowledge                  # List knowledge
spwn profile <name> journal                    # Session history
spwn profile <name> sessions                   # Active sessions
spwn profile <name> edit                       # Edit profile.yaml
spwn profile <name> role                       # Show/set agent role
spwn profile <name> engine                     # Show/set runtime engine

# Messaging
spwn msg send <agent> --from <sender> "msg"    # Send message to agent
spwn msg inbox <agent>                         # Show inbox messages
spwn msg watch <agent>                         # Watch for new messages

# Snapshots
spwn snap save <id>                            # Save world state
spwn snap ls                                   # List snapshots
spwn snap restore <snap>                       # Restore from snapshot
spwn snap rm <snap>                            # Remove a snapshot

# Architect (your always-on world builder)
spwn architect start                           # Start the Architect daemon
spwn architect stop                            # Stop the Architect daemon
spwn architect status                          # Show status, channels, active worlds
spwn architect connect <channel>               # Connect to a messaging channel

# Dashboard
spwn dash start                                # Start the dashboard server
spwn dash open                                 # Open in browser

# Marketplace
spwn get install <name>                        # Install a package
spwn get ls                                    # List installed packages
spwn get search <query>                        # Search the marketplace
spwn get rm <name>                             # Remove a package

# Authentication
spwn auth login                                # Login
spwn auth logout                               # Logout
spwn auth token                                # Show/manage tokens

# Ephemeral agent
spwn agent --ephemeral "task" --world w-acme-84721
```

**Design rules:**
- `spwn` IS the verb — no "create" or "spawn" subcommand
- Top-level commands for world operations: up, down, ls, logs, attach, inspect
- Config name via `-c` flag, agent name via `-n` flag (not positional — avoids conflict with subcommands)
- Global flags: `--json`, `--quiet`/`-q`, `--verbose`/`-v`

## IDs

- World: `w-{config-name}-{5digits}` (e.g. `w-default-84721`)
- Agent: `a-{agent-name}-{5digits}` (e.g. `a-leonardo-52103`)
- Generated with `crypto/rand`

## Config Paths

```
~/.spwn/
├── org.yaml                 # Organization manifest (source of truth)
├── claw/
│   ├── state.json           # Active worlds, channels
│   └── claw.yaml            # Claw runtime config
├── worlds/
│   ├── default.yaml
│   └── acme-org.yaml
├── agents/
│   └── neo/
│       ├── profile.yaml     # Role, engine, identity, requires, delegation
│       ├── identity/        # persona.md, purpose.md, traits.md
│       ├── skills/          # Agent skills
│       ├── memory/
│       │   ├── knowledge/   # Facts, codebase info
│       │   ├── playbooks/   # Step-by-step workflows
│       │   └── journal/     # Session logs
│       ├── sessions/        # Active session state
│       └── bonds.md         # Relationships with other agents
└── skills/
    ├── local/               # Custom skills
    └── marketplace/         # Downloaded from marketplace
```

**Manifest hierarchy (cascading overrides):** `org.yaml` → `universe.yaml` → `profile.yaml`. Each level inherits from parent and can override.

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
│   │   ├── universe.go              #   Public API (World, Manifest, Architect, Observatory)
│   │   └── internal/
│   │       ├── architect/           #     Orchestration (spawn, destroy, list)
│   │       │   ├── colony.go        #       Multi-agent: SpawnAgents, Chief/Manager/Worker
│   │       │   └── npc.go           #       Ephemeral: SpawnNPC
│   │       ├── backend/             #     Docker adapter (Backend port)
│   │       ├── runtime/             #     Claude Code adapter (Runtime port)
│   │       ├── provider/            #     Anthropic + OpenAI adapters (Provider port)
│   │       ├── channel/             #     CLI adapter (Channel port)
│   │       ├── get/                  #     LocalRegistry adapter (Get/Marketplace port)
│   │       ├── observatory/         #     HTTP API server (/api/worlds, /api/agents)
│   │       ├── sync/                #     Git config sync (SyncToGit, PullFromGit)
│   │       ├── physics/             #     Physics/faculties generation
│   │       ├── manifest/            #     Config parsing (universe.yaml, profile.yaml, org.yaml)
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
│   │   ├── world/                   #     World subcommands (up, down, ls, logs, attach, inspect)
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
    └── Observatory container (sibling)
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
// GIVEN a universe with a chief and two workers
// WHEN the chief delegates a task
// THEN both workers receive work
// AND the chief aggregates results
```

### Test layers:
- **Behavioral specs** (`core/universe/tests/e2e/`, `tests/e2e/`) — what the system does (the specification)
- **CLI specs** (`apps/cli/cli_test.go`) — what the user sees (flag parsing, help, output)
- **Unit tests** (`*_test.go` next to source) — how the code works (implementation details)

The behavioral specs are the source of truth. If a spec fails, the implementation is wrong — not the spec.
