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

spwn uses **DooD**, not DinD ([ADR-013](decisions/013-dood-over-dind.md)). The host's Docker daemon is shared via socket mount (`/var/run/docker.sock`); every container is a **sibling** on the same daemon — no nesting, no privilege escalation, no performance overhead.

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

The [gate](08-gate.md) is a separate long-running host container that owns cookie-bearing tools and the shared browser primitive.

## Key invariants

- **Per-repository.** Agents and local blocks live in `./spwn/`, not `~/.spwn/`.
- **Declared deps only.** An agent can use only the dependencies in its (unioned) `dependencies:` list — unlisted means physically absent from the image.
- **Transitive resolution.** Dependency dependencies are resolved recursively and topologically sorted.
- **Lock file is text.** `spwn.lock` — one dep per line, trivially diffable.
- **Labels are truth.** World state comes from Docker labels, not on-disk state.
- **Compile is deterministic.** Same input → same output, covered by golden tests.
- **Layers flow downward.** Enforced by depguard; no upward imports.
- **External systems sit behind ports.** Every dependency on the outside world is a Go interface defined in the domain that uses it, with the adapter injected at startup — no domain package names a concrete backend ([ADR-011](decisions/011-ports-and-adapters.md)).

## What the architecture does not do

The boundaries are as load-bearing as the layers:

- **It does not implement agent reasoning.** The runtime does the thinking, planning, and tool use; spwn creates the reality it operates in.
- **It does not run inference.** No model call happens in the machinery — the runtime manages its own.
- **It does not schedule work.** Worlds are created on demand, by an operator or by a declared trigger ([Automations](13-automations.md)).
- **It does not network worlds together.** Each world is isolated; anything between two worlds goes through the host.
- **It does not persist worlds.** The Soul and the Mind persist; worlds are disposable.
- **It does not restrict behaviour through rules.** What an agent cannot do, it cannot do because the capability is absent from the image ([Physics](10-physics.md)).

## Planned surfaces

Thin SDKs are designed, not shipped: language wrappers (TypeScript first, then Python) over the same domain APIs the CLI consumes, so code can act as an operator — spawn a world, assign a task, collect the result — without shelling out. Today the shipped operator surfaces are the CLI and the web UI.

The runtime stack has further designed-but-unshipped layers, each with its own record: a runtime normalization layer inside worlds ([ADR-008](decisions/008-rivet-runtime-layer.md)), multi-provider and subscription auth behind it ([ADR-010](decisions/010-pi-mono-multi-provider.md)), and the Architect's messaging channels ([ADR-009](decisions/009-zeroclaw-as-claw.md), [ADR-012](decisions/012-organization-manifest.md)).

## Related

- [Developing](02-developing.md) — the toolchain, which file a change opens, the code style these layers hold to.
- [Testing](03-testing.md) — how the layers are covered.
- [Operating](04-operating.md) · [Update system](16-update-system.md) — how the binary ships and updates itself.
- [Concepts](05-concepts.md) — the abstractions these packages implement.
- [Gate](08-gate.md) — the host-side broker container.
- [Decisions](decisions/) — why the stack, the layering, and the container model are what they are.
