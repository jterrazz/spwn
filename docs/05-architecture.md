# Architecture

spwn is a polyglot monorepo — Go domain modules plus a Next.js/Tauri web UI — wired together with Go workspaces (`go.work`), pnpm (`pnpm-workspace.yaml`), and a top-level `Makefile` that is the single entry point for both toolchains. CI calls the Makefile targets directly; the canonical aggregate is [`.github/workflows/validate.yaml`](../.github/workflows/validate.yaml) — there is no `test-pr` meta-target by design.

## Repository layout

```
spwn/
├── apps/                            # end-user surfaces (L7)
│   ├── cli/                         #   Go — the `spwn` binary (cmd/spwn, cobra commands, ui)
│   ├── api/                         #   Go — HTTP server backing the web UI
│   ├── web/                         #   Next.js (src/) + Tauri shell (src-tauri/, Rust)
│   ├── gate/                        #   host-side broker + Playwright browser sidecar (see 06-gate)
│   └── spwn-cookie-sync/            #   Chrome extension feeding the gate
├── packages/                        # Go domain modules (shared libraries)
│   ├── platform/                    #   cross-cutting primitives (paths, IDs, env)
│   ├── activity/  auth/  upgrade/   #   platform utilities (event log, credentials, self-update)
│   ├── migration/                   #   ~/.spwn schema migrations
│   ├── dependency/  agent/          #   domain (dep schema/refs/lockfile; agent mind + evolution)
│   ├── project/                     #   spwn.yaml parser, validation, scaffolding, teams/orgs
│   ├── compile/  transpile/ runtimes/  # build (render Input→Tree; provider-neutral tree; runtime adapters)
│   ├── container/                   #   Docker backend adapter
│   ├── automation/                  #   trigger engine (cron + fs), receipts, catch-up
│   ├── gate/                        #   host-side gate logic
│   ├── world/  architect/           #   runtime (container lifecycle/labels; orchestration daemon)
├── catalog/                         # shipped example worlds + block bundles (spwn:* entries)
├── tests/                           # TypeScript vitest E2E + Playwright + governance (see 07-testing)
├── docs/                            # this corpus (+ generated cli/ man pages)
├── go.work · pnpm-workspace.yaml · Makefile
```

Each `packages/` module exposes its public API in the root `.go` file; implementation details live under `internal/`, which the Go compiler itself keeps private.

## Layered dependency graph

Imports flow **downward only**, across seven layers. This is enforced at three levels, in order of strength:

1. **depguard** ([`.golangci.yml`](../.golangci.yml)) — lint-time deny rules per layer. **This file is the mechanical source of truth**; a violation fails lint and blocks PRs (e.g. `L3 (domain) must not import L5 (build)`).
2. **Go `internal/`** — compile-time privacy for implementation packages.
3. **This document** — review-time ground truth for intent.

```
L7 Surface     apps/cli · apps/api · apps/web · apps/gate      (imports anything)
   ──────────────────────────────────────────────────────────────────────────
L6 Runtime     world · architect                               (orchestration hub)
   ──────────────────────────────────────────────────────────────────────────
L5 Build       compile · transpile · runtimes · container      (project → image)
   ──────────────────────────────────────────────────────────────────────────
L4 Project     project                                         (manifest + validation)
   ──────────────────────────────────────────────────────────────────────────
L3 Domain      dependency · agent                              (deps + agents)
   ──────────────────────────────────────────────────────────────────────────
L2 Platform    activity · auth · upgrade · migration           (platform utilities)
   ──────────────────────────────────────────────────────────────────────────
L1 Foundation  platform                                        (constants + IDs)
```

`automation` and `gate` are cross-cutting subsystems consumed at the runtime/surface layers. When the exact deny rules or a package's layer is in question, read `.golangci.yml` — it is what actually runs.

## Data flow: `spwn up`

Every arrow crosses a layer boundary; every layer has one job.

```
spwn.yaml + agents/**  →  project.Load       →  project.Manifest + []AgentRef
project.Manifest       →  project/resolve    →  []string (merged deps)
merged deps            →  dependency.Parse   →  *dependency.Parsed (schema + files)
*dependency.Parsed     →  compile ToolFrom…  →  tool set
tool set               →  image build        →  Docker image
Docker image           →  compile.Render     →  compile.Tree (in-memory files)
compile.Tree           →  world.Spawn        →  running container + synced agents (architect)
```

## Container architecture: Docker-outside-of-Docker (DooD)

spwn uses **DooD**, not DinD. The host's Docker daemon is shared via socket mount (`/var/run/docker.sock`); every container is a **sibling** on the same daemon — no nesting, no privilege escalation, no performance overhead.

```
Host machine
└── Docker daemon (/var/run/docker.sock)
    ├── Architect container (always-on, socket-mounted)
    ├── World containers (siblings, created by the Architect)
    └── Desktop App container (sibling)
```

Two modes:

- **Local CLI (direct)** — `spwn up` calls Docker directly from the host; no Architect container needed.
- **Hosted Architect (containerized)** — `spwn architect start` launches the Architect in a long-lived socket-mounted container that creates and manages world containers as siblings. Channels connect here.

The [gate](06-gate.md) is a separate long-running host container that owns cookie-bearing tools and the shared browser primitive.

## Key invariants

- **Per-repository.** Agents and local blocks live in `./spwn/`, not `~/.spwn/`.
- **Declared deps only.** An agent can use only the dependencies in its (unioned) `dependencies:` list — unlisted means physically absent from the image.
- **Transitive resolution.** Dependency dependencies are resolved recursively and topologically sorted.
- **Lock file is text.** `spwn.lock` — one dep per line, trivially diffable.
- **Labels are truth.** World state comes from Docker labels, not on-disk state.
- **Compile is deterministic.** Same input → same output, covered by golden tests.
- **Layers flow downward.** Enforced by depguard; no upward imports.

## Code style

- No cgo.
- Errors: `error: lowercase message.\nActionable hint.`
- Domain modules own all business logic; the CLI is a thin wrapper (parse flags → call domain API → format output).
- Types avoid stutter: `world.World` not `world.WorldInstance`, `agent.Info` not `agent.AgentInfo` — the package name provides context.

## Related

- [Concepts](02-concepts.md) — the abstractions these packages implement.
- [Gate](06-gate.md) — the host-side broker container.
- [Testing](07-testing.md) — how the layers are covered.
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) · [`contributing/releasing.md`](contributing/releasing.md) · [`contributing/update-system.md`](contributing/update-system.md) — contributor runbooks.
