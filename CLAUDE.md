# Spwn — Project Conventions

## Core Principle: The Control Plane for AI Agents

Spwn is the **control plane for AI agents** — bringing order to agent chaos. Isolated worlds, persistent identity, physics-based security, multi-agent collaboration. Every layer is an interface (port) with swappable adapters. If a tool dies tomorrow, swap one adapter. Core logic never changes.

### The 8 Ports

| Port | What it abstracts | Default adapter |
|------|-------------------|-----------------|
| **Runtime** | How agents think | Claude Code (ACP) |
| **Provider** | Which LLM | Anthropic |
| **Channel** | How Claw talks to outside | CLI |
| **Backend** | Where universes run | Docker |
| **Memory** | How Minds persist | Filesystem (markdown) |
| **Store** | How state is tracked | JSON file |
| **Tool** | What agents can do | Built-in + MCP |
| **Skill** | Reusable capabilities | Local files |

## Vocabulary

### The Hierarchy
- **Organization**: Top-level manifest (org.yaml). Org-wide defaults, shared skills, governance, config sync.
- **Claw**: The God. Always-on ZeroClaw daemon in Docker. Connected to all channels. Creates/destroys worlds. Self-manages via spwn.
- **Universe**: The reality — physics, constants, resource limits. One per org. Configured in `universe.yaml`. Defines what is physically possible.
- **World**: A living workspace inside the universe. Has agents, elements, and a project. Many per universe. Configured in `~/.spwn/worlds/`.
- **Governor**: Leader agent inside a world. Decomposes tasks, delegates to citizens, aggregates results.
- **Citizen**: Persistent worker agent. Has a Mind — remembers, learns, evolves.
- **NPC**: Ephemeral agent. No Mind, no memory. Single task, fire & forget.
- **Observatory**: Visual dashboard. Real-time view of everything.

### The Physics
- **Physics**: Constants (CPU, memory, timeout), laws (network, max-processes), elements (@unix, @git, jq).
- **Elements**: Building blocks. @packs expand to collections. If not listed, doesn't exist.
- **Faculties**: Verified elements + gate bridges, auto-generated as `/world/faculties.md`.

### The Life
- **Mind**: Persistent identity — 6 layers: personas, skills, knowledge, playbooks, journal, sessions.
- **Soul**: Immutable core — purpose, values, bonds. Never changes.
- **Life Manifest**: `life.yaml` — declares tier, runtime, identity, body requirements.

### The Bridge
- **Gate**: Bridge between universe and host. Host-side (Go) manages element bridging. Container-side (Rivet) normalizes runtimes.
- **Rivet**: Runtime normalization layer. One API across all agent runtimes. Event streaming, session persistence.

### Evolution
- **Reflexion**: Review journal → promote successes to playbooks (auto-reflexion.md). `spwn agent reflect`
- **Sleep**: Archive stale files, prune old sessions. `spwn agent sleep`
- **Forking**: Clone a Mind from source to target agent. `spwn agent fork`

## CLI Commands

```bash
# Claw (the God)
spwn claw start                          # Start the Claw daemon
spwn claw stop                           # Stop the Claw daemon
spwn claw status                         # Show status, channels, active worlds
spwn claw connect <channel>              # Connect to a messaging channel
spwn claw "migrate auth to sessions"     # Talk to it (planned)

# World (a living workspace inside the universe)
spwn world                               # Spawn with defaults
spwn world -c acme-org                   # Named config
spwn world --governor morpheus -w ~/acme-org
spwn world list / inspect / logs / attach / destroy
spwn world send <id> --from <a> --to <b> "msg"  # Send message between agents
spwn world inbox <id> [agent]            # Show inbox messages
spwn world watch <id>                    # Watch for new messages (foreground)

# Agent (citizens + evolution)
spwn agent -n neo --world w-acme-84721
spwn agent init [name]
spwn agent list / inspect / export
spwn agent talk neo "how's the migration?"
spwn agent reflect <agent-id>            # Reflexion: journal → auto-reflexion.md
spwn agent sleep <agent-id>              # Sleep: archive stale, prune sessions
spwn agent fork <agent-id>               # Fork: clone Mind to new agent

# NPC (ephemeral)
spwn agent --npc "task" --world w-acme-84721

# Observatory
spwn observatory start / open

# Skills (marketplace)
spwn skill list                          # List available skills
spwn skill install <skill>               # Install a skill
spwn skill remove <skill>                # Remove a skill
```

**Design rules:**
- `spwn` IS the verb — no "create" or "spawn" subcommand
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
│       ├── soul/            # purpose.md, values.md, bonds.md
│       ├── mind/
│       │   ├── personas/
│       │   ├── skills/
│       │   ├── knowledge/
│       │   ├── playbooks/
│       │   └── journal/
│       ├── sessions/
│       └── life.yaml        # Tier, runtime, identity, body
└── skills/
    ├── local/               # Custom skills
    └── marketplace/         # Downloaded from marketplace
```

**Manifest hierarchy (cascading overrides):** `org.yaml` → `universe.yaml` → `life.yaml`. Each level inherits from parent and can override.

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
│   │       │   ├── colony.go        #       Multi-agent: SpawnAgents, Governor/Citizen
│   │       │   └── npc.go           #       Ephemeral: SpawnNPC
│   │       ├── backend/             #     Docker adapter (Backend port)
│   │       ├── runtime/             #     Claude Code adapter (Runtime port)
│   │       ├── provider/            #     Anthropic + OpenAI adapters (Provider port)
│   │       ├── channel/             #     CLI adapter (Channel port)
│   │       ├── skill/               #     LocalRegistry adapter (Skill port)
│   │       ├── observatory/         #     HTTP API server (/api/worlds, /api/agents)
│   │       ├── sync/                #     Git config sync (SyncToGit, PullFromGit)
│   │       ├── physics/             #     Physics/faculties generation
│   │       ├── manifest/            #     Config parsing (universe.yaml, life.yaml, org.yaml)
│   │       ├── state/               #     Universe + Claw state (JSON)
│   │       ├── models/              #     Domain types (World, Manifest, Status, AgentRecord)
│   │       └── ports/               #     8 port interfaces (Runtime, Backend, Provider, etc.)
│   │
│   ├── agent/                       #   go.mod — life management
│   │   ├── agent.go                 #   Public API (Info, InitMind, Reflect, Sleep, Fork)
│   │   └── internal/
│   │       ├── mind/                #     Mind CRUD (init, validate, list, inspect, export)
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
│       ├── constants.go             #     Defaults, MindLayers, BaseImage
│       ├── paths.go                 #     BaseDir(), WorldsDir(), AgentsDir(), OrgPath()
│       ├── identity.go              #     GenerateWorldID(), GenerateAgentID()
│       └── names.go                 #     RandomCosmosWord(), RandomAgentName()
│
├── apps/                            # Deployable consumers
│   ├── cli/                         #   go.mod — the spwn binary
│   │   ├── cmd/spwn/main.go         #     Entry point
│   │   ├── root.go                  #     Root cobra command
│   │   ├── init.go                  #     spwn init
│   │   ├── defaults.go              #     Auto-create defaults on first run
│   │   ├── world/                   #     World subcommands (thin wrappers)
│   │   ├── agent/                   #     Agent subcommands (+ reflect, sleep, fork)
│   │   ├── claw/                    #     Claw subcommands (start, stop, status, connect)
│   │   ├── skill/                   #     Skill subcommands (list, install, remove)
│   │   ├── observatory/             #     Observatory subcommands (start, open)
│   │   ├── ui/                      #     Stepper, table, style, format
│   │   └── tests/
│   │       └── integration/         #   Cross-domain flows (world + agent)
│   │
│   └── observatory/                 #   Visual dashboard (CLI placeholder, Next.js planned)
│       └── package.json
│
├── platform/                        # Build infrastructure
│   ├── images/                      #   Docker images
│   │   ├── Dockerfile               #     spwn-base production image
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
make build-image         # docker build spwn-base:latest from platform/images/
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

1. **Specify** — Define behavior in the blueprint (what the system SHOULD do)
2. **Encode** — Write tests that encode those specs (they fail initially)
3. **Implement** — Write code that makes the tests pass
4. **Verify** — The test suite IS the living specification

The E2E test suite is the behavioral specification of spwn. Each test describes a user-visible behavior:

```go
// GIVEN a universe with a governor and two citizens
// WHEN the governor delegates a task
// THEN both citizens receive work
// AND the governor aggregates results
```

### Test layers:
- **Behavioral specs** (`core/universe/tests/e2e/`, `tests/e2e/`) — what the system does (the specification)
- **CLI specs** (`apps/cli/cli_test.go`) — what the user sees (flag parsing, help, output)
- **Unit tests** (`*_test.go` next to source) — how the code works (implementation details)

The behavioral specs are the source of truth. If a spec fails, the implementation is wrong — not the spec.
