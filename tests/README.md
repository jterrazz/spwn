# Test Infrastructure

This document describes the test layers, tooling, and conventions used across the spwn project.

## Test Layers

### 1. Go Unit Tests

Fast, isolated tests that run without Docker. Located next to their source files as `*_test.go`.

```bash
make test                  # all unit tests
make test-foundation       # packages/foundation only
make test-agent            # packages/agent only
make test-world         # packages/world only
make test-cli              # apps/cli only
make test-messenger        # packages/messenger only
```

Examples:
- `packages/foundation/paths_test.go` - path resolution logic
- `packages/agent/agent_test.go` - agent lifecycle
- `packages/world/internal/manifest/manifest_test.go` - YAML parsing
- `apps/cli/ui/table_test.go` - table formatting

### 2. Go E2E Tests

Integration tests that spawn real Docker containers using the `spwn-test:latest` image (a mock environment with `mock-claude` replacing the real Claude binary). Located in `packages/world/tests/e2e/`.

```bash
make test-e2e              # builds test image, then runs E2E suite
make test-e2e-world     # world E2E only
```

These tests use the build tag `//go:build e2e` and are excluded from `make test`.

### 3. TypeScript E2E Tests

Behavioral specs that exercise the compiled `spwn` CLI binary end-to-end. Located in `tests/e2e/`. They spawn processes, interact with Docker, and assert on CLI output.

```bash
cd tests && npx vitest           # run all TS E2E tests
cd tests && npx vitest run       # run once (no watch)
cd tests && npx tsc --noEmit     # type-check only
```

## Prerequisites

- **Docker**: Required for all E2E tests (both Go and TypeScript).
- **Go 1.25+**: Required for Go tests.
- **Node.js 20+**: Required for TypeScript E2E tests.
- **Test image**: Run `make build-test-image` before E2E tests. This builds the `spwn-test:latest` Docker image from `fixtures/Dockerfile.test`.
- **Binary**: TypeScript E2E tests require `bin/spwn`. Run `make build` first.

## How mock-claude Works

E2E tests do not call the real Claude Code CLI. Instead, they use a mock:

**`fixtures/mock-claude/mock-claude.sh`** is a bash script installed as `/usr/local/bin/claude` inside the test Docker image. It:

1. Accepts and ignores real Claude CLI flags (`--session-id`, `--resume`, etc.)
2. Inspects the container environment (checks for `/agents`, `/world/physics.md`, `/work`, etc.)
3. Writes its observations as JSON to `/tmp/claude-mock.json`
4. Optionally writes to `/workspace/mock-output.txt` to prove write access
5. Supports `--exit-code` and `--sleep` flags for testing error/timeout scenarios

The Go E2E framework reads this JSON via `TestContext.ReadMockOutput()` and exposes it through `MockAssertion` (e.g., `ExpectMock(func(m) { m.SawMind(); m.SawPhysics() })`).

## Test Infrastructure

### Go E2E Setup (`packages/world/tests/e2e/setup/`)

| File | Purpose |
|------|---------|
| `context.go` | `TestContext` - creates isolated temp SPWN_HOME, connects to Docker, registers cleanup |
| `builders.go` | `SpawnBuilder` - fluent builder for spawning worlds with config/agent/workspace |
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

| File | Purpose |
|------|---------|
| `helpers.ts` | `createSpwnHome()`, `createAgent()`, `createWorldConfig()`, `waitForContainer()`, `retry()` |
| `spwn.specification.ts` | `createTestContext()` - creates isolated SPWN_HOME, provides `ctx.spwn()` runner with env overrides |
| `output-helpers.ts` | `expectLine()`, `expectNoLine()`, `expectTableHeader()`, `expectTableRow()`, `stripAnsi()` |
| `world-assertion.ts` | `WorldAssertion` - asserts on container state, files, mounts |
| `mind-assertion.ts` | `MindAssertion` - asserts on agent Mind directory structure |
| `state-assertion.ts` | `StateAssertion` - asserts on `state.json` contents |
| `mock-llm/` | Mock LLM server for testing agent talk flows |
| `mock-api/` | Mock API server (marketplace, auth) |

**Pattern:**
```typescript
describe("world spawn", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("creates a running Docker container", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(["world", "--agent", "neo", "-w", ctx.home], 60_000);

    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Created container\s+w-\w+-\d{5}/);

    const id = parseWorldId(result.output)!;
    ctx.world(id).toBeRunning();
  });
});
```

Key design points:
- Each test gets its own `createTestContext()` with an isolated temp dir.
- `afterEach` calls `ctx.cleanup()` to destroy Docker containers and remove temp files.
- Use `expectLine()` for structured assertions on CLI output - never weak `toContain()`.
- `vitest.config.ts` sets `fileParallelism: false` because Docker tests must run sequentially.

## Adding New Tests

### Go Unit Test

1. Create `your_file_test.go` next to the source file.
2. Use standard `testing.T` patterns.
3. Use table-driven tests where appropriate.
4. Run with `make test` or `go test ./...` in the module directory.

### Go E2E Test

1. Create `your_feature_test.go` in `packages/world/tests/e2e/`.
2. Add `//go:build e2e` build tag at the top.
3. Use `setup.NewSpawnBuilder(t)` to create test infrastructure.
4. Follow GIVEN/WHEN/THEN comment structure.
5. Run with `make test-e2e`.

### TypeScript E2E Test

1. Create `your-feature.e2e.test.ts` in the appropriate `tests/e2e/{domain}/` directory.
2. Use `describe`/`test` with clear behavioral names.
3. Use `createTestContext()` and `afterEach(() => ctx?.cleanup())`.
4. Use output helpers (`expectLine`, `expectTableHeader`) for CLI assertions.
5. Run with `cd tests && npx vitest`.

## Test File Naming Conventions

| Layer | Pattern | Example |
|-------|---------|---------|
| Go unit | `*_test.go` (next to source) | `manifest_test.go` |
| Go E2E | `*_test.go` (in `tests/e2e/`) | `spawn_test.go` |
| TS E2E | `*.e2e.test.ts` (in `tests/e2e/{domain}/`) | `spawn.e2e.test.ts` |

## Test Function Naming

- **Go**: `TestFeature_Scenario` (e.g., `TestSpawn_CreatesRunningContainer`)
- **TypeScript**: `describe("feature")` + `test("scenario description")` (e.g., `describe("world spawn")` + `test("creates a running Docker container")`)

## Vitest Configuration

Tests use `tests/vitest.config.ts`:
- `testTimeout: 120_000` - 2 minutes per test (Docker is slow)
- `hookTimeout: 60_000` - 1 minute for setup/teardown hooks
- `fileParallelism: false` - sequential execution (Docker resource constraints)
