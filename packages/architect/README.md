# packages/architect

World orchestrator — spawn, destroy, and coordinate live containers.

## Role

The Architect is the layer that sits on top of `packages/world` + `packages/image` and turns a declarative spwn project into running Docker containers. It owns the spawn pipeline (image build → render tree → docker-cp agent files → start container → sync credentials), the Architect daemon (the always-on container that runs the CLI itself), and the colony logic that wires multi-agent worlds together. Below it: backend primitives, image build, compile tree. Above it: CLI and HTTP API.

## Key types

- `Architect` — the orchestrator. `New(backend, store)` or `NewFromEnv()`.
- `Spawn(ctx, SpawnOpts) → SpawnResult` — the top-level verb: builds the image if needed, materialises the compile tree, starts the container, and returns the world ID. `AgentSpec` describes each agent in the request.
- `Destroy(ctx, worldID)` — stop + remove container, persist mind state, emit activity.
- `StartDaemon` / `StopDaemon` / `GetDaemonStatus` / `TalkExecArgs` — lifecycle for the always-on `spwn-architect` container that runs the CLI inside Docker-over-Docker.
- `BuildArchitectImage` — builds the architect image from `packages/image/base/architect.Dockerfile`.

## Related

- **Imported by** — `apps/api`, `apps/cli`
- **Imports** — `packages/world` (state, runtime, labels), `packages/image` (backend, build), `packages/compile`, `packages/runtimes`, `packages/agent`, `packages/activity`, `packages/auth`, `packages/platform`, `catalog`
