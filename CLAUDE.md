# Universe

Sandboxed AI agent environments powered by Docker.

## Architecture

Go project with standard layout:
- `cmd/universe/` — CLI entry point (cobra)
- `internal/architect/` — Architect: main orchestrator (create, spawn, list, inspect, destroy)
- `internal/backend/` — Backend interface + Docker implementation (Docker SDK for Go)
- `internal/mind/` — Mind path resolution + directory scaffolding
- `internal/physics/` — physics.md generation
- `internal/agent/` — Claude Code CLI spawning
- `internal/config/` — UniverseConfig and Universe types

## Conventions

- Standard Go error handling (fmt.Errorf with %w wrapping)
- Context passed through all async operations
- Container labels prefixed with `universe.` for filtering
- `internal/` for all non-exported packages

## Commands

```
go build ./cmd/universe
go vet ./...
go test ./...
./universe <subcommand>
```
