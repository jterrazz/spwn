# Universe — Project Conventions

## Vocabulary

- **Universe**: An isolated Docker container — a reality for an agent
- **Spawn**: Create a universe and bring an agent to life inside it
- **Physics**: Constraints of a universe (constants, laws, elements)
- **Technologies**: Evolved capabilities available to the agent (@packs or individual binaries)
- **Faculties**: What the agent can actually do (verified technologies + gate bridges)
- **Mind**: The agent's persistent identity (6 layers of markdown)
- **Gate**: Bridge between Substrate and Universe (Epoch 3)
- **Substrate**: The host machine

## IDs

- Universe: `u-{config-name}-{5digits}` (e.g. `u-default-84721`)
- Agent: `a-{agent-name}-{5digits}` (e.g. `a-leonardo-52103`)
- Generated with `crypto/rand`

## Config

- Named universe configs: `~/.universe/universes/{name}.yaml`
- Agent Minds: `~/.universe/agents/{name}/`
- State: `~/.universe/state.json`
- No project-local manifests — configs are infrastructure, not project code

## Code Style

- No cgo
- Errors: `error: lowercase message.\nActionable hint.`
- One agent per universe
- `internal/` for all business logic — CLI is a thin wrapper
- Backend interface abstracts Docker — no direct Docker calls outside `internal/backend/`

## Build

```
make build        # → bin/universe
make build-image  # → universe-base:latest
make test
make lint
make clean
```
