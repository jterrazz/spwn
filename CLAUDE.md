# Spwn — Project Conventions

## Vocabulary

- **Universe**: An isolated Docker container — a reality for an agent
- **Spawn**: Create a universe and bring an agent to life inside it
- **Physics**: The reality definition — constants, laws, and elements
- **Elements**: Building blocks of the world (@packs or individual binaries), declared under `physics.elements`
- **Faculties**: What the agent can actually do (verified elements + gate bridges)
- **Mind**: The agent's persistent identity (6 layers of markdown)
- **Gate**: Two-sided bridge between Host and Universe. Host-side (Go) manages mounts and element bridging. Container-side (Rust) speaks ACP to the agent CLI.
- **Gate Bridge**: An MCP server on the Host exposed as a CLI command inside the universe via wrapper scripts at `/gate/bin/`
- **Life Manifest**: Optional `life.yaml` in agent dir — declares identity (soul/mind) and body requirements
- **Host**: The host machine
- **Operator**: Any entity (human, agent, or code) that interacts with an agent at runtime

## IDs

- Universe: `u-{config-name}-{5digits}` (e.g. `u-default-84721`)
- Agent: `a-{agent-name}-{5digits}` (e.g. `a-leonardo-52103`)
- Generated with `crypto/rand`

## Config

- Named universe configs: `~/.spwn/universes/{name}.yaml`
- Agent Minds: `~/.spwn/agents/{name}/`
- Life manifest: `~/.spwn/agents/{name}/life.yaml` (optional)
- State: `~/.spwn/state.json`
- Gate bridges: top-level `gate:` key in universe YAML, or `--gate "source:as:caps"` CLI flag on `spwn universe`
- No project-local manifests — configs are infrastructure, not project code

## Project Layout

Multi-module Go monorepo managed with `go.work`:

```
spwn/
├── go.work
├── cli/                 # go.mod — CLI consumer (cobra commands, entry point at cmd/spwn/)
├── domains/
│   ├── universe/        # go.mod — world management (architect, backend, physics, manifest, state)
│   ├── agent/           # go.mod — life management (mind, journal, session)
│   └── gate/            # go.mod — bridge protocol (server, bridge)
├── shared/              # go.mod — cross-cutting (config paths, constants, IDs)
├── container/           # Build infra (Dockerfile.test, Rust gate)
└── __tests__/mock/      # Test fixtures
```

**Dependency graph:** `cli` -> `universe`, `agent`, `gate`, `shared` / `universe` -> `agent`, `gate`, `shared` / `agent` -> `shared` / `gate` -> `shared`

## Code Style

- No cgo
- Errors: `error: lowercase message.\nActionable hint.`
- One agent per universe
- Domain modules own their business logic — CLI is a thin wrapper
- Backend interface abstracts Docker — no direct Docker calls outside `domains/universe/backend/`
- Container-side Gate is Rust (`container/`) — separate binary, communicates via Unix socket

## Build

```
make build            # cd cli && go build -o ../bin/spwn ./cmd/spwn
make build-image      # → spwn-base:latest
make build-gate       # → container Rust gate binary
make test             # go vet all modules
make test-e2e         # per-domain E2E tests
make test-universe    # cd domains/universe && go test ./...
make test-agent       # cd domains/agent && go test ./...
make lint
make clean
```
