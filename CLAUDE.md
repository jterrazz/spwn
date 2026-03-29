# Spwn — Project Conventions

## Core Principle: Framework, Not a Product

Spwn is a **framework for orchestrating artificial life**. Every layer is an interface (port) with swappable implementations (adapters). If a tool dies tomorrow, swap one adapter. Core logic never changes.

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
- **Claw**: The God. Always-on ZeroClaw daemon in Docker. Connected to all channels. Creates/destroys universes. Self-manages via spwn.
- **Universe**: A contained reality. Org-scale, cross-repo. Has physics, elements, and inhabitants. Multiple agents collaborate inside.
- **Governor**: Leader agent inside a universe. Decomposes tasks, delegates to citizens, aggregates results.
- **Citizen**: Persistent worker agent. Has a Mind — remembers, learns, evolves.
- **Visitor**: Ephemeral agent. No Mind, no memory. Single task, fire & forget.
- **Observatory**: Visual dashboard. Real-time view of everything.

### The World
- **Physics**: Constants (CPU, memory, timeout), laws (network, max-processes), elements (@unix, @git, jq).
- **Elements**: Building blocks. @packs expand to collections. If not listed, doesn't exist.
- **Faculties**: Verified elements + gate bridges, auto-generated as `/universe/faculties.md`.

### The Life
- **Mind**: Persistent identity — 6 layers: personas, skills, knowledge, playbooks, journal, sessions.
- **Soul**: Immutable core — purpose, values, bonds. Never changes.
- **Life Manifest**: `life.yaml` — declares tier, runtime, identity, body requirements.

### The Bridge
- **Gate**: Bridge between universe and host. Host-side (Go) manages element bridging. Container-side (Rivet) normalizes runtimes.
- **Rivet**: Runtime normalization layer. One API across all agent runtimes. Event streaming, session persistence.

### Evolution (future)
- **Reflexion**: Review journal → promote successes to playbooks.
- **Sleep**: Consolidate experience into durable knowledge.
- **Forking**: Clone a Mind, run experiments, keep the best branch.

## CLI Commands

```bash
# Claw (the God)
spwn claw start                          # Start the Claw daemon
spwn claw stop                           # Stop the Claw daemon
spwn claw status                         # Show status, channels, active universes
spwn claw connect <channel>              # Connect to a messaging channel
spwn claw "migrate auth to sessions"     # Talk to it (planned)

# Universe (the world)
spwn universe                            # Spawn with defaults
spwn universe -c acme-org                # Named config
spwn universe --governor morpheus -w ~/acme-org
spwn universe list / inspect / logs / attach / destroy

# Agent (citizens + evolution)
spwn agent -n neo --universe u-acme-84721
spwn agent init [name]
spwn agent list / inspect / export
spwn agent talk neo "how's the migration?"
spwn agent reflect <agent-id>            # Reflexion: journal → auto-reflexion.md
spwn agent sleep <agent-id>              # Sleep: archive stale, prune sessions
spwn agent fork <agent-id>               # Fork: clone Mind to new agent

# Visitor (ephemeral)
spwn visitor "task" --universe u-acme-84721

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

- Universe: `u-{config-name}-{5digits}` (e.g. `u-default-84721`)
- Agent: `a-{agent-name}-{5digits}` (e.g. `a-leonardo-52103`)
- Generated with `crypto/rand`

## Config Paths

```
~/.spwn/
├── org.yaml                 # Organization manifest (source of truth)
├── claw/
│   ├── state.json           # Active universes, channels
│   └── claw.yaml            # Claw runtime config
├── universes/
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
│   │   ├── universe.go              #   Public API (World, Manifest, Architect)
│   │   ├── internal/                #   Private implementation
│   │   │   ├── architect/           #     Orchestration (spawn, destroy, list)
│   │   │   ├── backend/             #     Docker adapter
│   │   │   ├── physics/             #     Physics/faculties generation
│   │   │   ├── manifest/            #     Config parsing + validation
│   │   │   ├── state/               #     Universe registry (state.json)
│   │   │   └── models/              #     Domain types (World, Manifest, Status)
│   │   └── tests/
│   │       ├── unit/
│   │       └── integration/         #   Universe-only E2E (Docker required)
│   │
│   ├── agent/                       #   go.mod — life management
│   │   ├── agent.go                 #   Public API (Info, InitMind, ExportMind)
│   │   ├── internal/
│   │   │   ├── mind/                #     Mind CRUD (init, validate, list, inspect, export)
│   │   │   ├── journal/             #     Episodic memory (append, list)
│   │   │   └── session/             #     Session persistence (load, save)
│   │   └── tests/
│   │       ├── unit/
│   │       └── integration/         #   Agent-only E2E (filesystem, no Docker)
│   │
│   ├── gate/                        #   go.mod — bridge protocol
│   │   ├── gate.go                  #   Public API (Server, Bridge, SetupBridges)
│   │   └── internal/
│   │       ├── bridge/              #     Wrapper script generation
│   │       └── server/              #     HTTP-over-TCP gate server
│   │
│   └── foundation/                  #   go.mod — cross-cutting primitives
│       ├── constants.go             #     Defaults, MindLayers, BaseImage
│       ├── paths.go                 #     BaseDir(), UniversesDir(), AgentsDir()
│       ├── identity.go              #     GenerateUniverseID(), GenerateAgentID()
│       └── names.go                 #     RandomCosmosWord(), RandomAgentName()
│
├── apps/                            # Deployable consumers
│   ├── cli/                         #   go.mod — the spwn binary
│   │   ├── cmd/spwn/main.go         #     Entry point
│   │   ├── root.go                  #     Root cobra command
│   │   ├── init.go                  #     spwn init
│   │   ├── defaults.go              #     Auto-create defaults on first run
│   │   ├── universe/                #     Universe subcommands (thin wrappers)
│   │   ├── agent/                   #     Agent subcommands (thin wrappers)
│   │   ├── ui/                      #     Stepper, table, style, format
│   │   └── tests/
│   │       └── integration/         #   Cross-domain flows (universe + agent)
│   │
│   └── observatory/                 #   Future: visual dashboard (Next.js)
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
apps/cli ──→ core/universe, core/agent, core/gate, core/foundation
core/universe ──→ core/agent, core/gate, core/foundation
core/agent ──→ core/foundation
core/gate ──→ core/foundation
```

Each `core/` module exposes a public API in its root `.go` file. The `internal/` packages are private per module — no cross-module access.

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
make test-universe       # cd core/universe && go test ./...
make test-agent          # cd core/agent && go test ./...
make test-gate           # cd core/gate && go test ./...

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
| **Unit** | `core/*/tests/unit/` | ~1s | None |
| **Domain integration** | `core/*/tests/integration/` | ~30s | Docker (universe) or filesystem (agent) |
| **Cross-domain** | `apps/cli/tests/integration/` | ~2min | Docker |

Each domain tests only its own contract. Cross-domain flows (spawn universe + agent → verify journal) are the CLI's responsibility.
