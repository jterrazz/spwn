# Contributing to Spwn

## Prerequisites

- **Go 1.25+** (monorepo uses `go.work`)
- **Docker** (for E2E tests and world provisioning)
- **Node.js 20+** (for TypeScript E2E tests)
- **Rust** (only for `platform/gate-runtime`)

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
core/                   Domain libraries (pure logic, no I/O at boundary)
  universe/               World management (architect, backend, runtime adapters)
  agent/                  Agent lifecycle (mind, journal, session, evolution)
  gate/                   Host↔container bridge (server, bridge scripts)
  messenger/              Inter-agent messaging (inbox, models)
  foundation/             Cross-cutting primitives (paths, IDs, constants)

apps/                   Deployable consumers
  cli/                    The spwn binary (Cobra → domain APIs → output)
  observatory/            Dashboard (planned)

platform/               Build infrastructure
  images/                 Docker images (base, test)
  gate-runtime/           Container-side Rust gate
  fixtures/               Mock claude, test data

tests/                  TypeScript E2E test suite
  e2e/                    Behavioral specs (world, agent, messaging, etc.)
  setup/                  Test infrastructure (runners, assertions, mock LLM)
```

## Adding a Runtime Adapter

1. Create `core/universe/internal/runtime/{name}/{name}.go`
2. Implement the `runtime.Runtime` interface
3. Call `runtime.Register(&YourRuntime{})` in `init()`
4. The adapter auto-registers via blank import in `architect.go`

```go
package myruntime

import rt "spwn.sh/core/universe/internal/runtime"

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

- **Spec-first**: write the test before the implementation
- **Go unit tests**: `*_test.go` next to source. No Docker needed.
- **Go E2E tests**: `core/universe/tests/e2e/`. Build tag `//go:build e2e`. Needs Docker.
- **TypeScript E2E**: `tests/e2e/`. Runs against real `spwn` binary. Needs Docker.
- **Output assertions**: use `expectLine()`, `expectTableHeader()` — not weak `toContain()`

```bash
make test               # Go unit tests (fast, no Docker)
make test-e2e           # Go E2E tests (Docker required)
cd tests && pnpm test   # TypeScript E2E (Docker required)
```

## Commit Messages

Imperative mood, lowercase:
```
feat: add world snapshot restore
fix: agent talk skips dead containers
test: add messaging inbox E2E specs
docs: update CLI reference
```

## Resources

- [Blueprint Wiki](https://github.com/jterrazz/spwn-wiki) — ADRs, domain models, epoch plans
- [CLI Reference](https://spwn.sh/docs) — auto-generated from source
- [CLAUDE.md](./CLAUDE.md) — full project conventions
