# Universe

Sandboxed realities for AI agents. Go CLI that provisions Docker containers with resource limits, mounts agent Minds and workspaces, then spawns Claude Code inside.

## Build & Test

```bash
make build          # → bin/universe
make build-image    # → universe-base:latest Docker image
make test           # go test ./...
make lint           # golangci-lint
```

## Project Structure

- `cmd/universe/` — entry point
- `cli/` — cobra commands (spawn, list, inspect, destroy, init, agent/*)
- `internal/config/` — types, defaults, constants
- `internal/manifest/` — YAML parsing, validation, defaults merging
- `internal/id/` — human-readable universe ID generation
- `internal/state/` — JSON file persistence (~/.universe/universes.json)
- `internal/mind/` — Mind directory management (init, validate, list, inspect)
- `internal/physics/` — physics.md generation
- `internal/backend/` — Backend interface + Docker SDK implementation
- `internal/architect/` — orchestrator wiring everything together
- `wordlist/` — adjective/noun lists for IDs
- `container/` — Dockerfile for base image

## Conventions

- Use Universe vocabulary: spawn (not create), destroy (not delete), origin (not image), physics (not config), elements (not tools), Mind (not profile), Gate (not bridge)
- Universe IDs: `u-{adj}-{noun}-{5digits}` (e.g. `u-bright-comet-84721`)
- Error format: `error: lowercase message.\nActionable hint.`
- No cgo dependencies. State stored as JSON files.
- One agent per universe. Mind persists after universe destruction.
