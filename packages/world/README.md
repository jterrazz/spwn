# packages/world

The world domain — live container lifecycle and runtime abstractions.

## Role

A "world" is one Docker container deploying one or more agents into a sandboxed workspace. This package owns the primitives that sit *below* the orchestrator (see `packages/architect`) and *above* the Docker daemon (see `packages/compile/backend`): world manifest parsing, state persistence (`~/.spwn/state.json`), the docker-cp bind-mount label scheme, runtime-adapter registry, and the typed model of worlds/workspaces/agent-records that every other layer consumes. The top-level `world.go` is a thin public facade over the sub-packages so external callers never reach into internals.

## Key types

- `World`, `Workspace`, `Manifest`, `Status`, `AgentRecord` — re-exports of the `models/` types.
- `Store` / `NewStore` — `state.Store` wrapper: JSON-persisted world state at `~/.spwn/state.json`.
- `Backend`, `NewDocker()` — convenience re-exports of `packages/compile/backend`'s Docker adapter (worlds use the same backend as the image builder).
- `Runtime` / `GetRuntime(name)` / `BuildRuntimeCommand` — the runtime adapter port (spawn-time concerns: `BuildCommand`, `SyncHostCredentials`, `PrelaunchShell`, `DefaultConfigFiles`).
- `LoadManifest` / `LoadManifestPath` / `ListConfigs` / `CreateDefaultConfig` / `CreateConfig` / `ValidateManifest` — CRUD over `~/.spwn/worlds/<name>.yaml`.
- Sub-packages: `labels/` (docker label keys), `runtimestate/` (container-state tracking), `state/` (JSON store), `runtime/` (adapter port), `manifest/`, `models/`.

## Related

- **Imported by** — `apps/api`, `apps/cli`, `packages/architect`, `packages/runtimes` (runtime port)
- **Imports** — `packages/agent`, `packages/compile`, `packages/platform`, `packages/activity`
