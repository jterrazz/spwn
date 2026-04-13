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
  world/                  World lifecycle (architect, backend, runtime, api)
  agent/                  Mind lifecycle (journal, session, evolution)
  imagebuilder/           Composable Docker image builder + tool catalog
  messenger/              Inter-agent messaging (inbox, models)
  migration/              ~/.spwn schema migrations
  foundation/             Cross-cutting primitives (paths, IDs, auth, activity)

apps/                   Deployable consumers
  cli/                    The spwn binary (Cobra → domain APIs → output)
  web/                    Next.js + Tauri web/desktop UI

examples/               Bundled example worlds
fixtures/               Test fixtures (mock-claude, testdata, Dockerfile.test)

tests/                  TypeScript E2E test suite
  e2e/                    Behavioral specs (world, agent, messaging, etc.)
  setup/                  Test infrastructure (runners, assertions, mock LLM)
  ui/                     Playwright specs for the web UI
```

## Adding a Runtime Adapter

1. Create `packages/world/internal/runtime/{name}/{name}.go`
2. Implement the `runtime.Runtime` interface
3. Call `runtime.Register(&YourRuntime{})` in `init()`
4. The adapter auto-registers via blank import in `architect.go`

```go
package myruntime

import rt "spwn.sh/packages/world/internal/runtime"

type MyRuntime struct{}
func init() { rt.Register(&MyRuntime{}) }
func (r *MyRuntime) Name() string { return "my-runtime" }
func (r *MyRuntime) BuildCommand(cfg rt.SpawnConfig) []string { ... }
func (r *MyRuntime) BaseImage() string { return "node:20" }
// ... implement remaining interface methods
```

## Adding a CLI Command

1. Create `apps/cli/{domain}/{command}.go`
2. Register with `Cmd.AddCommand(yourCmd)` in `init()`
3. Command calls domain APIs — no business logic in CLI layer
4. Use `ui.New()` stepper for output (✓/✗/→)
5. Add to custom help in `apps/cli/root.go`

## Testing

### Running Tests

```bash
# Unit tests — fast, no Docker required
make test

# Go E2E tests — requires Docker + test image
make build-test-image   # build spwn-test:latest (once)
make test-e2e           # run Go E2E suite

# TypeScript E2E tests — requires built binary + Docker + test image
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
- **Go E2E tests**: `packages/world/tests/e2e/`. Build tag `//go:build e2e`. Needs Docker.
- **TypeScript E2E**: `tests/e2e/`. Runs against real `spwn` binary. Needs Docker.
- **Output assertions**: use `expectLine()`, `expectTableHeader()` — not weak `toContain()`
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

- [Knowledge Wiki](https://github.com/jterrazz/spwn-wiki) — ADRs, domain models, epoch plans
- [CLI Reference](https://spwn.sh/docs) — auto-generated from source
- [CLAUDE.md](./CLAUDE.md) — full project conventions
