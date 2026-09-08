# Developing

How spwn is changed: the toolchain a clone needs, the loop a change runs through, the file a given change opens, and what it owes before it lands. What proves a change is [Testing](03-testing.md); the shape it must respect is [Architecture](01-architecture.md).

## The toolchain

| Tool         | Why                                                          |
| ------------ | ------------------------------------------------------------ |
| **Go 1.25+** | Every domain package and the `spwn` binary; wired by `go.work` |
| **Docker**   | Worlds are containers, and every E2E layer needs a daemon     |
| **Node 20+** | The TypeScript E2E suites and the web UI, driven by pnpm      |
| **Rust**     | Only `apps/web/src-tauri`, the desktop shell                  |

```bash
git clone https://github.com/jterrazz/spwn.git
cd spwn
go work sync
pnpm install --frozen-lockfile
make build              # .artifacts/go/spwn
```

## The loop

The `Makefile` is the single entry point for both toolchains, and CI calls its targets directly — [`.github/workflows/validate.yaml`](../.github/workflows/validate.yaml) *is* the aggregate, so there is no `test-pr` meta-target to keep in sync. Run `make` with no arguments for the annotated list; the four gates a change runs locally before it is pushed are:

```bash
make lint            # go vet across go.work + pnpm -r lint (oxlint + oxfmt + knip)
make test            # Go unit tests across the workspace (~5s)
make test-contracts  # every surface declared the proof it needs
make test-cli        # the TypeScript CLI E2E against the compiled binary (Docker)
```

Adding a module to `go.work` is the only thing needed to bring it under lint and test coverage: `GO_MODS` is read from `go work edit -json`, so the Makefile never lists a package.

## Which file a change opens

| Change                                  | Opens                                                                             |
| --------------------------------------- | ---------------------------------------------------------------------------------- |
| Business logic of a domain              | `packages/<domain>/` — the root `.go` file is the public API, `internal/` is private |
| A CLI command                           | `apps/cli/<domain>/<command>.go`, registered with `Cmd.AddCommand` in `init()`       |
| A runtime (Claude Code, codex, …)       | `packages/runtimes/<name>/` — see below                                             |
| A shipped tool or template              | `catalog/<slug>/`, one directory per entry                                          |
| The layer a package may import          | [`.golangci.yml`](../.golangci.yml), the depguard deny rules                        |
| A world's on-disk shape                 | `packages/compile/` and `packages/transpile/`                                       |

A CLI command carries no business logic: it parses flags, calls a domain API, and formats output through the `ui.New()` stepper (✓/✗/→). A new top-level command is also added to the custom help in `apps/cli/root.go`.

### Adding a runtime adapter

A runtime lives at `packages/runtimes/<name>/` and ships any subset of three facets:

| Facet    | Interface           | Does                                                                                                       |
| -------- | ------------------- | ----------------------------------------------------------------------------------------------------------- |
| `Tool`   | `tool.Tool`         | The install recipe (apt/curl/npm, user config) — runs at image-build time                                    |
| `Render` | `transpile.Runtime` | Translates the provider-neutral source tree into runtime-specific output files                               |
| `Spawn`  | `runtimes.Spawner`  | Host-side spawn behaviour — `BuildCommand`, credential sync, prelaunch shell, default config, container path |

Create `tool.go`, `spawn.go` and optionally `render.go`; a `Render` facet reads its runtime-neutral prose from `packages/transpile/worldbook` rather than restating it. Bundle the facets in `adapter.go` and register them from `init()`:

```go
package myruntime

import "spwn.sh/packages/runtimes"

var Adapter = runtimes.Adapter{
    Name:            "my-runtime",
    DefaultProvider: "openai", // or "anthropic", "google", ""
    Tool:            Tool,     // *myTool implementing tool.Tool (optional)
    Render:          Renderer, // *renderer implementing transpile.Runtime (optional)
    Spawn:           Spawner,  // *spawner implementing runtimes.Spawner (optional)
}

func init() { runtimes.Register(Adapter) }
```

Then add a blank import to `packages/runtimes/defaults/defaults.go`, which is what makes a production binary pick the runtime up.

## Conventions

- **No cgo.**
- **Errors read as two lines** — `error: lowercase message.\nActionable hint.`
- **Types avoid stutter** — `world.World`, not `world.WorldInstance`; `agent.Info`, not `agent.AgentInfo`. The package name already carries the context.
- **Domain modules own all business logic**; a surface is a wrapper.
- **Commit messages are imperative and lowercase**, prefixed by the kind of change: `feat: add world snapshot restore`, `fix: agent talk skips dead containers`, `test: add messaging inbox E2E specs`, `docs: update CLI reference`.

## What a change owes

Four things land in the same commit as the change that makes them true:

1. **The guard.** A discovery grows a test, a `spwn check` rule, or a runtime error in the same change — the suite is the specification ([Testing](03-testing.md)).
2. **The contract entry.** A new runtime, route, command or catalog entry declares the proof it needs; `make test-contracts` refuses a surface that declared none.
3. **The regenerated projection.** [`reference/`](reference/) is projected from Cobra by `make docs` and is never hand-edited; the embedded catalog is projected by `make generate`, which `build`, `lint` and `test` already run.
4. **The chapter the behaviour falsified.** A page of this corpus that a change makes untrue is repaired by that change, not by a follow-up.

## Decision records

A decision this repository alone took is written to [`decisions/`](decisions/) as `NNN-kebab.md`, cut from [`decisions/_template.md`](decisions/_template.md). The status is `Proposed` until the owner writes `Accepted` — an agent never accepts its own record.

Numbers are historical and never reused, which is why 003 is absent: *security as physics* is a decision about the product rather than about this codebase, and it lives in spwn's product knowledge outside this repository. A decision spanning two repositories is recorded in the corpus that spans them, and linked from here.

## Related

- [Architecture](01-architecture.md) — the layers a change must not cross upward.
- [Testing](03-testing.md) — the pyramid, and what a test may assume.
- [Operating](04-operating.md) — cutting a release once the change has landed.
