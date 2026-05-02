# Spwn - Project Conventions

## Core Principle: The Building Blocks of Agent Intelligence

Spwn is the **operating system for autonomous agent worlds**. Compose tools, skills, and identity into agents, then spawn them into isolated worlds where they wake up, find their tools, and get to work.

The domain has three main abstractions, each owning one concern:

| Abstraction | What it owns | Implementation |
|---|---|---|
| **Runtime** | How an agent actually runs (CLI invocation, session capture, credential plumbing) | `packages/runtimes` - Claude Code today, codex next, others plug in as a ~50 LOC Adapter |
| **Backend** | Where worlds run | `packages/container/backend` - Docker; container labels are the source of truth for world state |
| **Mind** | How an agent persists across worlds | `packages/agent` - two markdown layers (`playbooks/`, `journal/`) on the host filesystem plus a single `SOUL.md` at the agent root. Knowledge is world-scoped, not in the Mind — declare a host path via `worlds.<name>.knowledge` in `spwn.yaml` (default `./spwn/knowledge`) and it gets bind-mounted into `/world/knowledge/`. Omit the key to spawn a world whose agents are never told a knowledge base exists. |

## Vocabulary

### Entities
- **Agent**: A persistent mind. Composed from tools, skills, and an identity. Has memory and evolution history. The main thing you create and ship.
- **World**: A runtime instance. Ephemeral. Where an agent actually runs - Docker container with filesystem, tools, and lifecycle. Dies when stopped.
- **Architect**: The always-on orchestration daemon. Connected to all channels. Creates/destroys worlds. Self-manages via spwn.

### Building blocks (composable, reusable)
- **Dependency**: The distribution unit. A `spwn.yaml` manifest (catalog or GitHub repo) that ships any combination of tools, skills, hooks, and agents. Installed via `spwn install`, pinned in `spwn.lock`. Agents reference them as external deps.
- **Skill (bare form)**: A `spwn/skills/<name>.md` file. Simplest authoring path for "write a paragraph of instructions."
- **Automation**: A `worlds.<name>.automations.<id>` map entry that wakes one agent on a trigger (cron expression or filesystem watch). Receipts land at `<root>/.spwn/runs.jsonl`. Engine in `packages/automation`; user guide at [`docs/automations.md`](docs/automations.md).

### Agent internals
- **Soul**: Who the agent is - purpose, voice, values, in a single file at `spwn/agents/<name>/SOUL.md`. Persists across world restarts. (Formerly split across `identity/profile.md`, `purpose.md`, `traits.md`; collapsed in 2026-04.)
- **Memory**: Journal and sessions. Persists across worlds, grows with experience. (Knowledge is world-scoped, not agent-scoped.)
- **Composition**: An agent's active dependencies (tools, skills, hooks), declared as a unified `dependencies:` list in `agent.yaml` using the `spwn:`/`skill:`/`tool:`/`hook:` schemes.

### Hierarchy (inside a world - "coming soon" on landing page)
- **Chief**: Lead agent inside a world. Decomposes tasks, delegates to workers, aggregates results.
- **Worker**: Persistent worker agent. Has its own identity and memory.
- **NPC**: Ephemeral agent. No persistent memory. Single task, fire & forget.

### Evolution
- **Dream**: Analyze experience → discover patterns → promote successes to playbooks. `spwn agent dream <name>`
- **Sleep**: Graceful shutdown - save state, consolidate, prune. `spwn agent sleep <name>`
- **Fork**: Clone an agent with everything it knows. `spwn agent fork <src> <dst>`

## CLI Commands

**Grammar: `spwn <noun> <verb>`** plus compose-style shortcuts
(`spwn up`, `spwn ls`, `spwn down`) and name-only shortcuts
(`spwn agent neo`, `spwn world default`). With no args, the
shortcuts act on every world declared in `spwn.yaml`.

```bash
# ── Project workflow ─────────────────────────────────────────────
spwn init                                      # Scaffold spwn.yaml + ./spwn/ + .spwn/
spwn check                                     # Validate the tree
spwn build --tree-only                         # Render the project tree to ./dist (preview/debug)
spwn build                                     # Transpile + compile into a project-specific Docker image
spwn up                                        # Spawn a world from the current project

# ── Compose-style shortcuts ──────────────────────────────────────
spwn up                                        # Bring up every world in spwn.yaml
spwn up default                                # Bring up one world by name
spwn agent neo                                 # Start the world that contains neo
spwn ls                                        # Agent-centric status (running/stopped/orphan)
spwn down                                      # Stop every world

# ── Agents ───────────────────────────────────────────────────────
spwn agent new neo                             # Create a blank agent in ./spwn/agents/
spwn agent ls                                  # List project agents
spwn agent inspect neo                         # Inspect composition, memory, history
spwn agent fork neo neo-v2                     # Clone memory + composition
spwn agent rm neo                              # Delete an agent

# Compose (via the project-level install / uninstall verbs)
spwn install python                            # Catalog dep, every agent
spwn install python --agent neo                # Catalog dep, only neo
spwn install skill/paper-reading --agent neo   # Local skill, only neo
spwn install tool/ffmpeg --agent neo           # Local tool, only neo
spwn install hook/pre-spawn --agent neo        # Local hook, only neo
spwn install command/refactor --agent neo      # Local command, only neo
spwn uninstall python --agent neo              # Detach from one agent

# Talk + messaging
spwn agent talk  neo "refactor auth"           # Full form of `spwn talk`
spwn agent send  neo "do this" --from morpheus # Async message to an agent's inbox
spwn agent inbox neo                           # Show neo's inbox
spwn agent watch neo                           # Tail neo's inbox live

# Evolution
spwn agent dream neo                           # Analyze experience, promote playbooks
spwn agent sleep neo                           # Consolidate memory, prune stale patterns

# ── Worlds ───────────────────────────────────────────────────────
# Worlds are inline map entries in spwn.yaml#worlds; there is no
# spwn/worlds/ directory any more.
spwn world start [name]                        # Start a world (no arg: every world in spwn.yaml)
spwn world stop  [name]                        # Stop a world
spwn world ls                                  # List running worlds
spwn world inspect <id>                        # Inspect a running world
spwn world enter   <id>                        # Interactive shell inside the world
spwn world snap save|ls|restore|rm             # World snapshots

# ── Dependencies ────────────────────────────────────────────
spwn install spwn:python              # Install a dep (adds to every agent + lockfile)
spwn uninstall spwn:python          # Remove a dep

spwn skill   new|edit|show|rm <name>           # Bare-markdown skill authoring (./spwn/skills/<name>.md)

# ── Automations ──────────────────────────────────────────────────
spwn automation ls                             # List declared automations + last-fired
spwn automation status                         # Per-automation rollup (fires/ok/fail)
spwn automation logs [-f] [-n N]               # Tail .spwn/runs.jsonl receipts
spwn automation daemon                         # Run the engine until interrupted

# ── Registry (planned) ───────────────────────────────────────────
spwn agent   get github:community/sci          # Install a shared agent     [planned]
spwn install github:acme/fuzzer                # Install from GitHub [planned]
spwn *       publish <name>                    # Push to registry           [planned]

# ── System ───────────────────────────────────────────────────────
spwn architect start|stop|status|talk|logs     # Always-on orchestration daemon
spwn web                                       # Open the local web UI
spwn auth login|logout|token                   # Provider credentials
```

**Design rules:**
- Strict noun-first grammar: `spwn <noun> <verb>`. Three shortcuts exist: `up`, `ls`, `talk`. No other top-level verbs.
- `rm` is contextual: `spwn agent rm neo` deletes the agent; `spwn agent rm neo --dependency X` removes a dep from it.
- Inside a project, commands resolve against `./spwn/` first. Outside a project, they operate on user-level paths (legacy).

## IDs

- World: `world-{planet}-{5digits}` (e.g. `world-rhea-84721`)
- Agent: `agent-{name}-{5digits}` (e.g. `agent-neo-52103`)
- Generated with `crypto/rand`.

## Config layout (per-repo)

A spwn project is **in the repo**, not in your home directory. `~/.spwn/` holds user-level credentials and daemon state only.

```
my-project/
├── spwn.yaml                    # manifest - version, name, inline worlds map
├── spwn/                        # committed project assets
│   ├── agents/
│   │   └── neo/
│   │       ├── agent.yaml       # composition: dependencies + runtime.backend
│   │       ├── AGENTS.md         # entry point (provider-neutral, compiled per runtime)
│   │       ├── SOUL.md          # who the agent is (one file: purpose, voice, values)
│   │       ├── playbooks/       # promoted patterns (name:/description: header = auto-indexed in CLAUDE.md)
│   │       └── journal/         # per-run history
│   ├── knowledge/               # world-scoped facts, bind-mounted to /world/knowledge/ (default path)
│   ├── skills/                  # project-scoped skill files (skill/<name> → spwn/skills/<name>.md)
│   ├── tools/                   # project-scoped tool dependencies (tool/<name> → spwn/tools/<name>/)
│   ├── hooks/                   # project-scoped runtime hooks (hook/<name> → spwn/hooks/<name>.yaml)
│   └── commands/                # project-scoped slash commands (command/<name> → spwn/commands/<name>.md)
└── .spwn/                       # gitignored local state
    ├── state.json               # live world IDs bound to this project
    ├── runs.jsonl               # automation receipts (one line per fire)
    ├── automations/state.json   # last-fired cursor per automation (catch-up math)
    └── cache/
```

Worlds are declared **inline** under `spwn.yaml#worlds` - the
world record (agents, workspaces, tool overrides) lives in yaml,
not in separate yaml files. A world optionally owns one filesystem
artifact: the directory referenced by its `knowledge:` key (default
`./spwn/knowledge`), which gets bind-mounted at `/world/knowledge/` inside
the running container. Omit the key and no mount happens — the agent
is never told a knowledge base exists. Each world entry names the
agents it deploys, the workspace mounts, and optional tool
overrides.

```
~/.spwn/                         # USER-LEVEL only, not per-project
├── credentials/                 # auth material surfaced to containers at /credentials
├── activity.jsonl               # global activity log
└── state/                       # architect daemon state
```

**Config hierarchy:** `agent.yaml` declares composition via a unified `dependencies:` list. The grammar splits **source** (the colon prefix) from **type** (the leading path segment): `spwn:<name>` is a catalog dep; `github:<owner>/<repo>` is a remote dep (planned); `skill/<name>`, `tool/<name>`, `hook/<name>`, `command/<name>` are local blocks authored under `spwn/skills/<name>.md`, `spwn/tools/<name>/`, `spwn/hooks/<name>.yaml`, `spwn/commands/<name>.md`. All four local schemes are iso: a path-style ref selected per agent, resolving to one file or directory on disk. Hooks are runtime-fired (PreToolUse, SessionStart, …); commands are slash-invoked prompt shortcuts (`/<name>`); each agent inherits only the blocks it explicitly subscribes to. Plus `runtime.backend`. `spwn.yaml#worlds[<name>]` declares the runtime environment (agents + workspaces). The union of project-wide and agent-specific dependencies is what actually materializes inside the container.

## Repository Structure

Polyglot monorepo: Go modules + Next.js/Tauri web UI, wired together
with Go workspaces, pnpm, and a top-level Makefile.

```
spwn/
├── go.work                          # Go workspace
├── pnpm-workspace.yaml              # JS workspace (apps/*, tests)
├── Makefile                         # Single entry point for Go + JS tasks
│
├── apps/                            # End-user binaries
│   ├── cli/                         #   go.mod - the `spwn` binary
│   │   ├── cmd/spwn/main.go         #     Entry point
│   │   ├── root.go                  #     Root cobra command
│   │   ├── world/                   #     spwn world (up, down, ls, inspect, logs, enter)
│   │   ├── agent/                   #     spwn agent (new, ls, rm, talk, fork, export…)
│   │   ├── snap/                    #     spwn world snap (save, ls, restore, rm)
│   │   ├── architect/               #     spwn architect (start, stop, status)
│   │   ├── web/                     #     spwn web (launches the web UI)
│   │   ├── auth/                    #     spwn auth (login, logout, token)
│   │   ├── skill/                   #     spwn skill (bare-markdown authoring)
│   │   ├── team/                    #     spwn team
│   │   ├── organization/            #     spwn organization
│   │   ├── logs/                    #     spwn logs
│   │   └── ui/                      #     Stepper, table, style
│   │
│   └── web/                         #   Next.js + Tauri desktop/web app
│       ├── src/                     #     Next.js app (React)
│       └── src-tauri/               #     Tauri shell (Rust)
│
├── packages/                        # Go domain modules (shared libraries)
│   ├── world/                       #   World lifecycle state, manifest, labels, models
│   ├── architect/                   #   Orchestration (spawn, destroy, deploy, NPCs)
│   ├── automation/                  #   Trigger engine (cron + fs) — receipts, catch-up
│   ├── runtimes/                    #   Runtime adapters (claude-code, codex, …)
│   ├── transpile/                   #   Source → Tree rendering (worldbook, source)
│   ├── compile/                     #   Docker image assembly (base + derived)
│   ├── container/                   #   Docker backend adapter
│   ├── agent/                       #   Agent/mind management (playbooks + journal)
│   ├── dependency/                  #   Dep resolution + catalog mirror
│   ├── project/                     #   spwn.yaml parser + project scaffold
│   ├── platform/                    #   Cross-cutting primitives (paths, ids)
│   ├── activity/                    #   Activity log
│   ├── auth/                        #   Credential resolution
│   ├── upgrade/                     #   Self-update logic
│   └── migration/                   #   ~/.spwn schema migrations
│
├── catalog/                         # Shipped example worlds + block bundles
├── tests/                           # TypeScript vitest E2E + Playwright + governance
│   ├── cli/                         #   Behavioral specs against the compiled binary
│   ├── web/                         #   Playwright specs (one folder per feature)
│   ├── _contracts/                  #   Governance YAMLs + assert-contracts.mjs
│   ├── _simulators/                 #   Runtime simulators (claude/codex stubs + test image)
│   ├── _setup/                      #   Shared CLI E2E harness (`spec` runner)
│   ├── _fixtures/                   #   Project skeletons consumed by `spec`
│   ├── _smoke/                      #   Real-build end-to-end (spwn init → up → probe)
│   └── _catalog/                    #   Catalog-bundle smoke + Dockerfile goldens
│   #
│   # See tests/ARCHITECTURE.md for layer pyramid + cookbook
├── docs/                            # Prose docs (architecture, releasing, CLI man pages)
│
├── Makefile
├── README.md
└── CLAUDE.md                        # (this file)
```

## Gate Architecture: shared host-side broker for cookie-bearing tools

The **gate** is a long-running Docker container on the host (`spwn gate
start`). It owns three concerns no individual world should:

1. **Cookie sync** — receives session cookies from the
   `spwn-cookie-sync` Chrome extension at `/sync/<provider>` and
   persists them under `~/.spwn/credentials/<provider>/cookies.json`.
2. **MCP routing** — exposes `/mcp/<element>/*` for every registered
   element (Notion proxy, Gmail/Gcal via `gws`, every catalog tool
   loaded from `~/.spwn/gate/tools/`).
3. **Browser primitive** — a Playwright Chromium sidecar
   (`apps/gate/browser/`, in-container `127.0.0.1:9001`) that
   catalog tools call to drive a cookie-loaded browser without
   shipping their own Chromium.

```
Host
└── spwn-gate container (port 9000 → host)
    ├── spwn-gate (Go)              ← cookie sync + MCP routing
    │     └── supervises:
    │         ├── gate-browser (Node, :9001)   ← Playwright pool
    │         └── catalog tools (Node, :9100+) ← per-tool MCP server
    ├── @spwn/gate-tool SDK          /usr/lib/node_modules/@spwn/gate-tool
    └── /gate/tools/<name>/         ← bind-mounted from ~/.spwn/gate/tools/
        └── tool.yaml + index.js
```

### Catalog tools that plug into the gate

A catalog entry under `catalog/<name>/tools/<name>/tool.yaml` becomes a
gate element by adding a `gate:` section:

```yaml
name: "spwn:x"
gate:
  cookies:
    domains: [x.com, twitter.com]
    cookies: [auth_token, ct0]
  mcp:
    entry: ["node", "index.js", "mcp-serve"]
install:
  commands:
    - cat > /usr/local/bin/x-mcp <<'WRAPPER'
      #!/bin/bash
      spwn-policy-check x "${1:-}" || exit 1
      exec mcp2cli --mcp "http://host.docker.internal:9000/mcp/x" "$@"
      WRAPPER
```

At startup the gate scans `/gate/tools/`, auto-registers each tool's
`CookieProvider` with cookie-sync (the extension picks it up next
refresh), spawns its MCP subprocess on a port from `9100+`, and
reverse-proxies `/mcp/<name>/*` into it. Adding a new site (LinkedIn,
Reddit, …) is one new directory — no edits to `packages/gate/`.

### The Node SDK

Catalog tools `require('@spwn/gate-tool')` and use:

- `new Tool({ name }).method(name, { description, schema, handler })` —
  register MCP methods. Same `handler` is invoked for both MCP calls
  (HTTP) and direct CLI invocation (`node index.js <method> --flags`).
- `openSession(provider)` — open a Playwright session in the sidecar
  with the provider's cookies pre-loaded. Returns a `Session` with
  `.goto / .click / .type / .scroll / .waitResponse / .eval / .end`.

Direct CLI mode is how host scripts (e.g. `publish.sh`) invoke writes
without going through the agent's MCP wrapper — keeping HITL methods
out of agent reach by construction.

### Generic browser escape hatch

Beyond per-site catalog tools, the gate exposes the sidecar directly
as `/mcp/browser` — agents that need ad-hoc browsing call
`browser-open / browser-goto / browser-click / browser-eval / …` for
sites without a dedicated tool. Heavier on tokens; reserve for
exploration, not scheduled scrapes.

### Per-agent allow/deny

Agents can constrain which methods of a dependency they may call:

```yaml
# agent.yaml
dependencies:
  - spwn:unix
  - name: spwn:x
    deny: [post-tweet, reply-tweet]   # read-only marketer
```

The compile pipeline materializes this as
`/etc/spwn/policy/<short>.json` in the agent's image. The catalog
tool's wrapper consults it via `spwn-policy-check <tool> <method>`
(installed by `spwn:mcp2cli`) and rejects denied calls before they
hit the gate. Deny-takes-precedence merging when multiple agents in
one world have conflicting policies.

## Container Architecture: Docker-outside-of-Docker (DooD)

spwn uses **DooD (Docker-outside-of-Docker)**, not DinD (Docker-in-Docker). The host's Docker daemon is shared via socket mount (`/var/run/docker.sock`). All containers are **siblings** on the same daemon - no nesting, no privilege escalation, no performance overhead.

```
Host machine
└── Docker daemon (/var/run/docker.sock)
    ├── Architect container (always-on, socket-mounted)
    ├── World containers (siblings, created by Architect)
    └── Desktop App container (sibling)
```

**Two modes:**
- **Local CLI (direct)** - `spwn up` calls Docker directly from the host. No Architect container needed.
- **Hosted Architect (containerized)** - `spwn architect start` launches the Architect in a long-lived container with the Docker socket mounted. It creates/manages world containers as siblings. Channels connect here.

## Dependency Graph

Enforced via depguard (`.golangci.yml`). Seven layers, each denies
imports from layers above:

```
L1 Foundation  →  (nothing)              platform, container
L2 Platform    →  L1                     activity, auth, upgrade
L3 Domain      →  L1-L2                  dependency, agent
L4 Project     →  L1-L3                  project
L5 Build       →  L1-L4                  compile, transpile, runtimes
L6 Runtime     →  L1-L5                  world, architect
L7 Surface     →  anything               apps/cli, apps/api
```

Each `packages/` module exposes a public API in its root `.go` file.
Implementation details live under `internal/`.

## Code Style

- No cgo
- Errors: `error: lowercase message.\nActionable hint.`
- Domain modules own all business logic - CLI is a thin wrapper
  (parse flags → call domain API → format output)
- Types avoid stutter: `world.World` not `world.WorldInstance`,
  `agent.Info` not `agent.AgentInfo`. Package name provides context.

## Build

```bash
make build               # cd apps/cli && go build -o ../../bin/spwn ./cmd/spwn
make test-image    # docker build spwn-test:latest for E2E

make test                # Unit tests across the Go workspace
make test-pkg PKG=agent  # Verbose go test for a single package

make test-go-e2e         # Go world E2E against Docker
make test-compile-e2e    # Image-build E2E under packages/compile/e2e
make test-cli            # TypeScript CLI E2E (Docker + Node 22)
make test-smoke          # Real-build smoke tests (spwn init → up → probe)
make test-web            # Playwright web E2E (Docker + browser)

make lint                # go vet across every module in go.work + pnpm lint
make clean               # rm -rf bin/

# Run `make` (no args) for the full target list with descriptions.
# CI lives in .github/workflows/validate.yaml — there's no aggregate
# meta-target on purpose; the workflow IS the aggregate.
```

## Testing Strategy

Three-layer pyramid:

| Layer | Location | Speed | Infra |
|-------|----------|-------|-------|
| **Unit** | `*_test.go` next to source files | ~1s | None |
| **E2E (Go)** | `packages/world/tests/e2e/`, `packages/compile/e2e/` | ~30s | Docker |
| **E2E (TS)** | `tests/cli/`, `tests/_smoke/` | ~2min | Built binary |

Each domain tests only its own contract. Cross-domain flows (spawn universe + agent → verify journal) are the CLI's responsibility.

## Development Methodology: Spec-First

Spwn follows a **spec-first** development process:

1. **Specify** - Define behavior in the knowledge (what the system SHOULD do)
2. **Encode** - Write tests that encode those specs (they fail initially)
3. **Implement** - Write code that makes the tests pass
4. **Verify** - The test suite IS the living specification

The E2E test suite is the behavioral specification of spwn. Each test describes a user-visible behavior:

```go
// GIVEN a world with a chief and two workers
// WHEN the chief delegates a task
// THEN both workers receive work
// AND the chief aggregates results
```

### Test layers:
- **Behavioral specs** (`packages/world/tests/e2e/`, `tests/cli/`) - what the system does (the specification)
- **CLI specs** (`apps/cli/cli_test.go`) - what the user sees (flag parsing, help, output)
- **Unit tests** (`*_test.go` next to source) - how the code works (implementation details)

The behavioral specs are the source of truth. If a spec fails, the implementation is wrong - not the spec.
