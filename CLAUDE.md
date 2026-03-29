# Spwn — Project Conventions

## Vocabulary

### The World
- **Universe**: An isolated Docker container — a reality for an agent. Defined by physics (constants, laws, elements).
- **Physics**: The reality definition. Constants (CPU, memory, timeout) are finite resources. Laws (network, max-processes) are structural constraints. Elements (@unix, @git, jq) are the building blocks.
- **Elements**: Building blocks of the world. @packs expand to collections (`@unix` → bash, coreutils, grep…). Individual binaries can be added. If it's not in the element list, it doesn't exist in this reality.
- **Faculties**: What the agent can actually do — verified elements + gate bridges, auto-generated as `/universe/faculties.md` inside the container.

### The Life
- **Agent**: A living entity with persistent identity. Has a Soul (immutable purpose/values), Mind (evolving knowledge), and Body (physical capabilities).
- **Mind**: The agent's persistent identity — 6 layers of markdown files: personas, skills, knowledge, playbooks, journal, sessions. Mounted at `/mind` inside every universe.
- **Soul**: Immutable core — purpose, values, bonds. Never changes. Defined in `soul/` directory.
- **Life Manifest**: Optional `life.yaml` in agent dir — declares identity (soul/mind) and body requirements.

### The Bridge
- **Gate**: Two-sided bridge between Host and Universe. Host-side (Go) manages mounts and element bridging. Container-side (Rust) speaks ACP to the agent CLI.
- **Gate Bridge**: An MCP server on the Host exposed as a CLI command inside the universe via wrapper scripts at `/gate/bin/`.

### The Infrastructure
- **Host**: The machine running spwn — physical reality.
- **Architect**: The orchestrator that creates, manages, and destroys universes.
- **Operator**: Any entity (human, agent, or code) that interacts with an agent at runtime.

### Evolution (future)
- **Reflexion**: After each session, review journal → promote successes to playbooks. Natural selection for behavior.
- **Sleep**: Consolidate raw experience into durable knowledge. Prune stale strategies. Resolve contradictions.
- **Forking**: Clone a Mind, run experiments, keep the best branch.

## CLI Commands

```bash
spwn init [name]                       # First-time setup (~/.spwn/)

spwn universe                          # Spawn universe with default config
spwn universe -c node-dev              # Spawn with named config
spwn universe --agent neo -w .         # Spawn with agent + workspace
spwn universe list                     # List active universes
spwn universe inspect <id>             # Show details
spwn universe logs <id>                # Stream agent output
spwn universe attach <id>              # Interactive shell
spwn universe destroy <id>             # Destroy (agent survives)

spwn agent                             # Spawn default agent
spwn agent -n neo                      # Spawn named agent
spwn agent init [name]                 # Create new agent identity
spwn agent list                        # List all agents
spwn agent inspect <name>              # Show agent details
spwn agent export <name>               # Export as tar.gz
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
├── universes/           # Named universe configs (YAML)
│   ├── default.yaml
│   └── node-dev.yaml
├── agents/              # Agent Minds (persistent)
│   └── neo/
│       ├── soul/        # purpose.md, values.md, bonds.md
│       ├── mind/
│       │   ├── personas/
│       │   ├── skills/
│       │   ├── knowledge/
│       │   ├── playbooks/
│       │   └── journal/
│       ├── sessions/
│       └── life.yaml    # Optional life manifest
└── state.json           # Active universe registry
```

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
- One agent per universe
- Domain modules own all business logic — CLI is a thin wrapper (parse flags → call domain API → format output)
- Backend interface abstracts Docker — no direct Docker calls outside `core/universe/internal/backend/`
- Container-side Gate is Rust (`platform/gate-runtime/`) — separate binary, TCP to host
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
