# Universe — Project Conventions

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

- Named universe configs: `~/.universe/universes/{name}.yaml`
- Agent Minds: `~/.universe/agents/{name}/`
- Life manifest: `~/.universe/agents/{name}/life.yaml` (optional)
- State: `~/.universe/state.json`
- Gate bridges: top-level `gate:` key in universe YAML, or `--gate "source:as:caps"` CLI flag
- No project-local manifests — configs are infrastructure, not project code

## Code Style

- No cgo
- Errors: `error: lowercase message.\nActionable hint.`
- One agent per universe
- `internal/` for all business logic — CLI is a thin wrapper
- Backend interface abstracts Docker — no direct Docker calls outside `internal/backend/`
- Container-side Gate is Rust (`container/gate/`) — separate binary, communicates via Unix socket

## Build

```
make build        # → bin/universe
make build-image  # → universe-base:latest
make build-gate   # → container/gate/target/release/universe-gate
make test
make test-e2e     # requires Docker + universe-test:latest image
make lint
make clean
```
