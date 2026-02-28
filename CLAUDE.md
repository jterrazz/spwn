# Universe

Sandboxed AI agent environments powered by Docker.

## Architecture

Go project with standard layout:
- `cmd/universe/` — CLI entry point (cobra): create, spawn, list, inspect, destroy, mind
- `internal/architect/` — Architect: main orchestrator (create, spawn, list, inspect, destroy)
- `internal/agent/` — Claude Code CLI spawning with session resume (--resume)
- `internal/backend/` — Backend interface + Docker implementation (auto-pulls images)
- `internal/config/` — UniverseConfig and Universe types
- `internal/journal/` — Automatic spawn journal (markdown entries per session)
- `internal/mind/` — Mind path resolution, validation, listing (6 subdirs)
- `internal/physics/` — physics.md generation + container element introspection
- `internal/session/` — Session persistence (JSON per mind+universe pair)

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
