# Test Infrastructure

This document describes the test layers, tooling, and conventions used across the spwn project.

## Test Layers

### 1. Go Unit Tests

Fast, isolated tests that run without Docker. Located next to their source files as `*_test.go`.

```bash
make test                  # all unit tests
make test-foundation       # packages/base only
make test-agent            # packages/mind only
make test-world            # packages/world only
make test-cli              # apps/cli only
make test-messenger        # packages/mailbox only
```

Examples:

- `packages/base/paths_test.go` - path resolution logic
- `packages/mind/mind_test.go` - agent lifecycle
- `packages/world/internal/manifest/manifest_test.go` - YAML parsing
- `apps/cli/ui/table_test.go` - table formatting

### 2. Go E2E Tests

Integration tests that spawn real Docker containers using the `spwn-test:latest` image (a mock environment with `mock-claude` replacing the real Claude binary). Located in `packages/world/tests/cli/`.

```bash
make test-e2e              # builds test image, then runs E2E suite
make test-e2e-world     # world E2E only
```

These tests use the build tag `//go:build e2e` and are excluded from `make test`.

### 3. TypeScript E2E Tests

Behavioral specs that exercise the compiled `spwn` CLI binary end-to-end. Located in `tests/cli/`. They spawn processes, interact with Docker, and assert on CLI output.

```bash
cd tests && npx vitest           # run all TS E2E tests
cd tests && npx vitest run       # run once (no watch)
cd tests && npx tsc --noEmit     # type-check only
```

## Prerequisites

- **Docker**: Required for all E2E tests (both Go and TypeScript).
- **Go 1.25+**: Required for Go tests.
- **Node.js 20+**: Required for TypeScript E2E tests.
- **Test image**: Run `make build-test-image` before E2E tests. This builds the `spwn-test:latest` Docker image from `tests/fixtures/Dockerfile.test`.
- **Binary**: TypeScript E2E tests require `bin/spwn`. Run `make build` first.

## How mock-claude Works

E2E tests do not call the real Claude Code CLI. Instead, they use a mock:

**`tests/fixtures/mock-claude/mock-claude.sh`** is a bash script installed as `/usr/local/bin/claude` inside the test Docker image. It:

1. Accepts and ignores real Claude CLI flags (`--session-id`, `--resume`, etc.)
2. Inspects the container environment (checks for `/agents`, `/world/physics.md`, `/work`, etc.)
3. Writes its observations as JSON to `/tmp/claude-mock.json`
4. Optionally writes to `/workspace/mock-output.txt` to prove write access
5. Supports `--exit-code` and `--sleep` flags for testing error/timeout scenarios

The Go E2E framework reads this JSON via `TestContext.ReadMockOutput()` and exposes it through `MockAssertion` (e.g., `ExpectMock(func(m) { m.SawMind(); m.SawPhysics() })`).

## Test Infrastructure

### Go E2E Setup (`packages/world/tests/cli/setup/`)

| File            | Purpose                                                                                                                                                   |
| --------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `context.go`    | `TestContext` - creates isolated temp SPWN_HOME, connects to Docker, registers cleanup                                                                    |
| `builders.go`   | `SpawnBuilder` - fluent builder for spawning worlds with config/agent/workspace                                                                           |
| `assertions.go` | Assertion chains: `StateAssertion`, `ContainerAssertion`, `MindAssertion`, `MockAssertion`, `SessionAssertion`, `JournalAssertion`, `GateAssertion`, etc. |

**Pattern:**

```go
func TestSomething(t *testing.T) {
    chain := setup.NewSpawnBuilder(t).
        WithAgent("test-agent").
        Execute()

    chain.ExpectState(func(s *setup.StateAssertion) {
        s.WorldCount(1)
        s.HasAgent("test-agent")
    })

    chain.ExpectContainer(func(c *setup.ContainerAssertion) {
        c.IsRunning()
        c.HasMount("/mind")
    })
}
```

Key design points:

- `NewTestContext(t)` creates an isolated SPWN_HOME in `t.TempDir()` and registers `t.Cleanup()` to destroy all spawned worlds.
- `SpawnBuilder.Execute()` returns an `AssertionChain` for fluent assertions.
- `WaitFor(t, timeout, interval, desc, conditionFn)` polls a condition instead of using `time.Sleep`.

### TypeScript E2E Setup (`tests/setup/`)

All TypeScript E2E tests run under `@jterrazz/test` via one
specification runner:

| File                   | Purpose                                               |
| ---------------------- | ----------------------------------------------------- |
| `cli.specification.ts` | Exports `spec`, the single runner bound to `bin/spwn` |

One runner, one mental model. Whether a test happens to touch
Docker is a property of what it asserts on, not a choice you make
at setup time. CLI-only tests use `.exec(...)` and reach for
stdout/stderr/file accessors. Tests that need container assertions
add `await using` and call `.container(name)` - the first access
lazily queries Docker; CLI-only tests never touch it.

The runner ships with a `transform` that strips ANSI and collapses
`/tmp/spec-*` paths to `<PROJECT>`, plus seed handlers for
`spwn.yaml/`, `agent/`, `state/`, and `activity/` fragments under
each test's `seeds/` directory.

**CLI-only pattern** - no containers, no docker cost:

```typescript
import { describe, expect, test } from 'vitest';
import { spec } from '../../../setup/cli.specification.js';

describe('spwn check', () => {
    test('valid project prints a clean success report', async () => {
        const result = await spec('check valid').project('single-agent').exec('check').run();

        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('valid-project.txt');
    });
});
```

- `.project('name')` copies `tests/fixtures/<name>/` into a fresh
  temp dir before the command runs.
- `result.stdout.toMatch('name.txt')` compares against
  `<test-dir>/expected/stdout/name.txt`. Regenerate with
  `JTERRAZZ_TEST_UPDATE=1 pnpm -C tests exec vitest run ...`.
- `result.json.toMatch('name.json')` parses stdout and deep-equals
  against a JSON fixture - pair with `spwn check --json` etc.
- `result.file('.spwn/state.json').exists` / `.content` reads the
  host-side working dir.

**Container-asserting pattern** - same `spec`, plus `.container(...)`:

```typescript
import { describe, expect, test } from 'vitest';
import { spec } from '../../../setup/cli.specification.js';

describe('world lifecycle', () => {
    test('up provisions a running world', async () => {
        await using result = await spec('up lifecycle').project('docker-pilot').exec('up').run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Created container');

        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        expect(neo.file('/world/physics.md').exists).toBe(true);

        const ls = await neo.exec('ls /world');
        ls.stdout.toContain('physics.md');
    });
});
```

- **`await using`** whenever a test might spawn containers. The
  dispose hook force-removes every container tagged with this
  test's run id so parallel runs never collide. Harmless no-op
  for tests that don't spawn anything.
- `result.container('<world-key>')` resolves by the
  `sh.spwn.world.config` label - the key declared under `worlds.`
  in `spwn.yaml`, not the sometimes-empty `sh.spwn.world.name`.
- `result.container(name).file(path)` / `.exec(cmd)` /
  `.inspect.value` / `.stdout` / `.stderr` use the same accessor
  API as the host-side `result` - no new vocabulary.
- Follow-up CLI commands that need a container id (e.g.
  `spwn world inspect <id>`) get it via `neo.id`.
- Always use the `docker-pilot` fixture (minimal agent without
  `@spwn/python`). `single-agent` fails to spawn because the
  base image lacks `pip3`.
- Banners (`Created container`, `Agent is alive`, `Destroyed`,
  `World destroyed`) go to **stderr**, not stdout - spwn follows
  the Unix convention of data-on-stdout / status-on-stderr.

## Adding New Tests

### Go Unit Test

1. Create `your_file_test.go` next to the source file.
2. Use standard `testing.T` patterns.
3. Use table-driven tests where appropriate.
4. Run with `make test` or `go test ./...` in the module directory.

### Go E2E Test

1. Create `your_feature_test.go` in `packages/world/tests/cli/`.
2. Add `//go:build e2e` build tag at the top.
3. Use `setup.NewSpawnBuilder(t)` to create test infrastructure.
4. Follow GIVEN/WHEN/THEN comment structure.
5. Run with `make test-e2e`.

### TypeScript E2E Test

1. Create `tests/cli/<area>/<feature>/<feature>.e2e.test.ts` (one
   per-feature folder per test file, siblings: `expected/`, `seeds/`).
2. Import `spec` from `tests/setup/cli.specification.js`.
3. Use `describe`/`test` with clear behavioral names.
4. Prefer structured assertions: `.toMatch('file.txt')`,
   `.json.toMatch('file.json')`, `.container(name).file(path)`,
   `.container(name).exec(cmd)`. Reach for `.toContain(substring)`
   when the intent is "some substring appears somewhere".
5. For Docker tests, always use `await using` to get automatic
   container cleanup.
6. Run with `pnpm -C tests exec vitest run --project {cli|docker} <glob>`.

## Test File Naming Conventions

| Layer   | Pattern                                    | Example             |
| ------- | ------------------------------------------ | ------------------- |
| Go unit | `*_test.go` (next to source)               | `manifest_test.go`  |
| Go E2E  | `*_test.go` (in `tests/cli/`)              | `spawn_test.go`     |
| TS E2E  | `*.e2e.test.ts` (in `tests/cli/{domain}/`) | `spawn.e2e.test.ts` |

## Test Function Naming

- **Go**: `TestFeature_Scenario` (e.g., `TestSpawn_CreatesRunningContainer`)
- **TypeScript**: `describe("feature")` + `test("scenario description")` (e.g., `describe("world spawn")` + `test("creates a running Docker container")`)

## Vitest Configuration

Tests use `tests/vitest.config.ts` with two projects:

- **`cli`** - CLI-mode tests. Parallel within the project, per-test
  `testTimeout: 120_000`, `hookTimeout: 60_000`.
- **`docker`** - Docker-mode tests. Also `testTimeout: 120_000` and
  `hookTimeout: 60_000`, but `fileParallelism: false` because Docker
  tests within a single file must run sequentially. Cross-file
  isolation is still handled by the framework's label-based test-run
  id, so the docker project is safe to parallelize if you ever drop
  `fileParallelism: false`.
