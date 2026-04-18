# packages/world

The world domain — live container lifecycle primitives.

## Role

A "world" is one Docker container deploying one or more agents into a sandboxed workspace. This package owns the primitives that sit *below* the orchestrator (see `packages/architect`) and *above* the Docker daemon (see `packages/container/backend`): world manifest parsing, state persistence (`~/.spwn/state.json`), the docker-cp + label-as-truth scheme, and the typed model of worlds/workspaces/agent-records every other layer consumes. The top-level `world.go` is a thin public facade over the sub-packages so external callers never reach into internals.

Runtime adapters live in `packages/runtimes/` — they were moved out of `world/runtime/` when the spawn port (BuildCommand, credential sync, prelaunch shell) was consolidated next to its implementations.

## Key types

- `World`, `Workspace`, `Manifest`, `Status`, `AgentRecord` — re-exports of the `models/` types. `World.Runtime` holds the runtime adapter selected at spawn time ("claude-code", "codex") so hot-deploy and talk routes can resolve the right spawner.
- `Store` / `NewStore` — JSON-persisted world state at `~/.spwn/state.json`.
- `Backend`, `NewDocker()` — convenience re-exports of `packages/container/backend`'s Docker adapter.
- `LoadManifest` / `LoadManifestPath` / `ListConfigs` / `CreateDefaultConfig` / `CreateConfig` / `ValidateManifest` — CRUD over the legacy `~/.spwn/worlds/<name>.yaml`.

## Sub-packages

- `labels/` — docker label key constants (`sh.spwn.*`).
- `runtimestate/` — per-world mutable state (session IDs, agent roster).
- `state/` — JSON `~/.spwn/state.json` persistence.
- `manifest/` — legacy global world-config parser (`~/.spwn/worlds/<name>.yaml`).
- `models/` — the typed shape of worlds, workspaces, agents.
- `tests/e2e/` — end-to-end integration tests.

## Related

- **Imported by** — `apps/api`, `apps/cli`, `packages/architect`
- **Imports** — `packages/agent`, `packages/container` (Docker backend), `packages/platform`, `packages/activity`
