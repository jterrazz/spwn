# packages/architect

World orchestrator — spawn, destroy, and coordinate live containers.

## Role

The Architect sits on top of `packages/world` + `packages/compile` + `packages/container` and turns a declarative spwn project into running Docker containers. It owns:

- **Spawn pipeline** — image build → render tree → docker-cp agent files → start container → sync credentials → materialise runtime config.
- **Architect daemon** — the always-on `spwn-architect` container that runs the CLI inside Docker-over-Docker.
- **Colony logic** — multi-agent world wiring (chief + managers + workers, hot-deploy, NPC flows).

Below it: container backend, image build, render tree. Above it: CLI and HTTP API.

## Runtime routing

The spawn pipeline is data-driven on runtime choice. `SpawnOpts.RuntimeName` selects which adapter drives the spawn (empty defaults to `"claude-code"`), and `models.World.Runtime` persists the choice on the world record so hot-deploy + talk can resolve the right spawner later.

## Key types

- `Architect` — the orchestrator. `New(backend, store)` or `NewFromEnv()`.
- `Spawn(ctx, SpawnOpts) → SpawnResult` — the top-level verb: builds the image if needed, materialises the compile tree, starts the container, and returns the world ID. `AgentSpec` describes each agent in the request.
- `Destroy(ctx, worldID)` — stop + remove container, persist mind state, emit activity.
- `DeployAgent(ctx, worldID, name, role)` — hot-deploy into a running world.
- `StartDaemon` / `StopDaemon` / `GetDaemonStatus` / `TalkExecArgs` — lifecycle for the architect daemon container.
- `BuildArchitectImage` — cross-compiles `spwn` and builds the architect image from `packages/compile/base/architect.Dockerfile`.

## Sub-packages

- `internal/deploy/` — tree materialisation + per-agent docker-cp sync-in/sync-out (moved here from `world/deploy` because only architect orchestrates these).

## Related

- **Imported by** — `apps/api`, `apps/cli`
- **Imports** — `packages/world` (state, labels, models, runtimestate), `packages/container/backend` (Docker adapter), `packages/compile` (image build), `packages/dependency` + `packages/dependency/resolver` (dep-resolution), `packages/transpile` + `packages/transpile/worldbook` (render + spwn content), `packages/runtimes` (+ `runtimes/defaults` for the built-in adapter set), `packages/agent`, `packages/activity`, `packages/auth`, `packages/platform`
