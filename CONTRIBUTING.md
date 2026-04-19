# Contributing to Spwn

## Prerequisites

- **Go 1.25+** (monorepo uses `go.work`)
- **Docker** (for E2E tests and world provisioning)
- **Node.js 20+** (for TypeScript E2E tests)
- **Rust** (only for `apps/web/src-tauri`)

## Getting Started

```bash
git clone https://github.com/jterrazz/spwn.git
cd spwn
go work sync
make build              # builds bin/spwn
make test               # runs all Go unit tests
make lint               # go vet across all modules

# TypeScript E2E tests
cd tests && pnpm install && pnpm test
```

## Project Structure

```
packages/               Domain libraries (Go modules)
  agent/                  Mind lifecycle (journal, session, evolution)
  architect/              World orchestration daemon
  compile/                Docker image builder (dependency resolution → image)
  dependency/             tool.yaml parser, refs, lockfile, resolver
  project/                spwn.yaml parser, scaffolding, validation
  runtimes/               Agent runtimes (claude-code, codex)
  transpile/              Provider-neutral source → runtime-specific Tree
  platform/               Cross-cutting primitives (paths, IDs, config)
  world/                  World-state + container labels
  auth/                   Provider resolution + credential storage
  migration/              ~/.spwn schema migrations
  activity/               Append-only event log

apps/                   Deployable consumers
  cli/                    The spwn binary (Cobra → domain APIs → output)
  api/                    HTTP API for the web UI
  web/                    Next.js + Tauri web/desktop UI

catalog/                Shipped catalog, one directory per entry
  <dep-slug>/             Installable dependency (tools/tool.yaml + optional skills/, files/)
  <template-slug>/        Scaffoldable project (agents/, skills/, knowledge/, spwn.yaml)

tests/                  E2E test suites (Go + TypeScript)
  cli/                    TypeScript CLI E2E (vitest)
  catalog/                Catalog invariant tests (Go)
  smoke/                  Full-build smoke tests
  fixtures/               Project fixtures used by the specs above
```

## Adding a Runtime Adapter

Runtimes live under `packages/runtimes/<name>/`. A runtime can ship any subset of three facets:

| Facet | Interface | Purpose |
|---|---|---|
| `Tool` | `tool.Tool` | Install recipe (apt/curl/npm, user config) — runs at image build time |
| `Render` | `transpile.Runtime` | Translates a provider-neutral source tree into runtime-specific output files |
| `Spawn` | `runtimes.Spawner` | Host-side spawn-time behavior — `BuildCommand`, credential sync, prelaunch shell, default config files, container config path |

1. Create `packages/runtimes/{name}/` with `tool.go`, `spawn.go` (and optionally `render.go`).
2. Implement the facets you need. For `Render`, read runtime-neutral prose from `packages/transpile/worldbook` — don't duplicate it.
3. Create `packages/runtimes/{name}/adapter.go` bundling the facets and registering via `init()`:

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

4. Add a blank import to `packages/runtimes/defaults/defaults.go` so production binaries pick up the new runtime automatically.

## Adding a CLI Command

1. Create `apps/cli/{domain}/{command}.go`
2. Register with `Cmd.AddCommand(yourCmd)` in `init()`
3. Command calls domain APIs - no business logic in CLI layer
4. Use `ui.New()` stepper for output (✓/✗/→)
5. Add to custom help in `apps/cli/root.go`

## Testing

### Running Tests

```bash
# Unit tests - fast, no Docker required
make test

# Go E2E tests - requires Docker + test image
make build-test-image   # build spwn-test:latest (once)
make test-e2e           # run Go E2E suite

# TypeScript E2E tests - requires built binary + Docker + test image
make build              # build bin/spwn
cd tests && npx vitest run

# Type-check TypeScript tests (no execution)
cd tests && npx tsc --noEmit

# Lint all Go modules
make lint
```

### Test Pyramid

| Layer | Command | Docker? | Speed | What it tests |
|-------|---------|---------|-------|---------------|
| Unit | `make test` | No | Fast | Pure logic, parsers, helpers |
| Go E2E | `make test-e2e` | Yes | Medium | Architect API, containers, state |
| TS E2E | `cd tests && npx vitest run` | Yes | Slow | Full CLI binary, end-to-end |

### Writing New Tests

- **Spec-first**: write the test before the implementation
- **Go unit tests**: `*_test.go` next to source. No Docker needed.
- **Go E2E tests**: `packages/world/tests/e2e/` and `packages/compile/e2e/`. Build tag `//go:build e2e`. Needs Docker.
- **TypeScript E2E**: `tests/cli/`. Runs against real `spwn` binary. Needs Docker.
- **Output assertions**: use `expectLine()`, `expectTableHeader()` - not weak `toContain()`
- See [tests/README.md](./tests/README.md) for full testing documentation and patterns

## Commit Messages

Imperative mood, lowercase:
```
feat: add world snapshot restore
fix: agent talk skips dead containers
test: add messaging inbox E2E specs
docs: update CLI reference
```

## Resources

- [Knowledge Wiki](https://github.com/jterrazz/spwn-wiki) - ADRs, domain models, epoch plans
- [CLI Reference](https://spwn.sh/docs) - auto-generated from source
- [CLAUDE.md](./CLAUDE.md) - full project conventions
