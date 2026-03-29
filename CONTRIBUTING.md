# Contributing to Spwn

## Prerequisites

- **Go 1.25+** (monorepo uses `go.work`)
- **Docker** (for E2E tests and universe provisioning)
- **Rust** (for `platform/gate-runtime` only)

## Getting Started

```bash
git clone https://github.com/jterrazz/spwn.git
cd spwn
go work sync
make build        # builds bin/spwn
make test         # runs all unit tests
make lint         # go vet across all modules
```

## Project Structure

```
core/               Domain libraries (no CLI, no IO at the boundary)
  foundation/         Cross-cutting primitives (paths, IDs, constants)
  agent/              Agent lifecycle (mind, journal, session, evolution)
  gate/               Host-container bridge (server, bridge scripts)
  universe/           World management (architect, backend, manifest, state)
apps/               Deployable binaries
  cli/                The spwn binary (Cobra commands -> domain APIs -> output)
  observatory/        Dashboard (planned)
platform/           Build infrastructure
  images/             Docker images (base, test)
  gate-runtime/       Container-side Rust gate
  fixtures/           Test fixtures
```

## Adding a Port Adapter

1. Define the port interface in the relevant `core/` module (e.g., `core/universe/internal/backend/backend.go`).
2. Create an adapter package implementing that interface (e.g., `backend/docker.go`).
3. Wire it in the module's public API file (`core/universe/universe.go`).
4. Add unit tests covering the adapter contract.

## Adding a CLI Command

1. Create a new file in `apps/cli/cmd/` (e.g., `cmd_foo.go`).
2. Register it in the parent command's `init()`.
3. The command should call domain APIs from `core/` and format output -- no business logic in the CLI layer.

## Testing Conventions

- **Spec-first**: write the test before or alongside the implementation.
- **Naming**: use `GIVEN_WHEN_THEN` style -- `TestArchitect_Spawn_CreatesContainer`.
- **Unit tests**: `go test ./...` in each module. No Docker required.
- **E2E tests**: `make test-e2e`. Requires Docker and the test image (`make build-test-image`).
- **Tags**: E2E tests use `//go:build e2e` so they don't run in CI by default.

## Commit Messages

Use imperative mood, lowercase, no period:

```
add visitor fire command
fix universe destroy when agent is running
refactor mind export to support layer filtering
```

Prefix with the domain when helpful: `universe: add attach command`.

## Architecture Decisions

See the [blueprint wiki](https://github.com/jterrazz/spwn-wiki/blob/main/domains/) for ADRs, domain models, and epoch plans.
