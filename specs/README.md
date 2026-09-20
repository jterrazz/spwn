# Test Infrastructure

This document describes the test layers, tooling, and conventions used across the spwn project.

## Test Layers

### 1. Go Unit Tests

Fast, isolated tests that run without Docker. Located next to their source files as `*_test.go`.

```bash
make test                        # all unit tests across the Go workspace
make test-pkg PKG=agent          # verbose go test for a single package
make test-pkg PKG=apps/cli       # path-form also works
```

Examples:

- `../packages/platform/paths_test.go` - path resolution logic
- `../packages/agent/agent_test.go` - agent lifecycle
- `../packages/project/internal/manifest/manifest_edge_test.go` - YAML parsing
- `../apps/cli/cli_test.go` - flag parsing + help output

### 2. Go E2E Tests

Integration tests that spawn real Docker containers using the `spwn-test:latest` image (a mock environment with the runtime simulators in `_simulators/` replacing the real Claude/Codex CLIs). Located in `../packages/world/tests/e2e/`.

```bash
make test-go-e2e              # builds test image, then runs the world E2E suite
make test-compile-e2e      # separate image-build E2E under packages/compile/e2e
```

These tests use the build tag `//go:build e2e` and are excluded from `make test`.

### 3. TypeScript E2E Tests

Behavioral specs that exercise the compiled `spwn` CLI binary end-to-end. Located in `cli/<domain>/`. They spawn processes, interact with Docker, and assert on CLI output.

Most of them are **documents** — a `<case>.spec.yaml` stating one terminal session — and the rest are chains in `<aspect>.spec.ts`. Which is which, and why, is [TypeScript E2E Setup](#typescript-e2e-setup-cli) below.

```bash
pnpm -C specs test               # run all TS E2E specs once
pnpm -C specs test:watch         # watch mode
cd specs && npx tsc --noEmit     # type-check only
```

## Prerequisites

- **Docker**: Required for all E2E tests (both Go and TypeScript).
- **Go 1.25+**: Required for Go tests.
- **Node.js 20+**: Required for TypeScript E2E tests.
- **Test image**: Run `make test-image` before E2E tests. This builds the `spwn-test:latest` Docker image from `_simulators/Dockerfile.test`.
- **Binary**: TypeScript E2E tests require `.artifacts/go/spwn`. Run `make build` first.

## How runtime simulators work

E2E tests do not call the real Claude Code or Codex CLIs. Instead, they use protocol-faithful **simulators** that ship inside the test image:

- `_simulators/claude/mock.sh` — installed as `/usr/local/bin/claude`
- `_simulators/codex/mock.sh` — installed as `/usr/local/bin/codex`
- `_simulators/Dockerfile.test` — builds `spwn-test:latest` with both pre-installed

The Claude simulator:

1. Accepts and ignores real Claude CLI flags (`--session-id`, `--resume`, etc.)
2. Inspects the container environment (checks for `/agents/<name>/CLAUDE.md`, `/workspaces`, etc.)
3. Writes its observations as JSON to `/tmp/claude-mock.json`
4. Optionally writes to `/workspaces/workspace0/mock-output.txt` to prove write access
5. Supports `--exit-code` and `--sleep` flags for testing error/timeout scenarios

The Go E2E framework reads this JSON via `TestContext.ReadMockOutput()` and exposes it through `MockAssertion` (e.g., `ExpectMock(func(m) { m.SawMind(); m.SawClaudeMD() })`).

## Test Infrastructure

### Go E2E Setup (`../packages/world/tests/e2e/setup/`)

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
        c.HasMount("/agents")
    })
}
```

Key design points:

- `NewTestContext(t)` creates an isolated SPWN_HOME in `t.TempDir()` and registers `t.Cleanup()` to destroy all spawned worlds.
- `SpawnBuilder.Execute()` returns an `AssertionChain` for fluent assertions.
- `WaitFor(t, timeout, interval, desc, conditionFn)` polls a condition instead of using `time.Sleep`.

### TypeScript E2E Setup (`cli/`)

All TypeScript E2E specs run under `@jterrazz/test` against one
specification runner:

| File                       | Purpose                                                                                              |
| -------------------------- | ---------------------------------------------------------------------------------------------------- |
| `cli/cli.specification.ts` | Exports `cli`, the single runner bound to `.artifacts/go/spwn`, docker-aware, with an `env` registry |

One runner, one mental model. Whether a spec happens to touch Docker is
a property of what it asserts on, not a choice made at setup time.
CLI-only specs reach for stdout/stderr/file accessors; specs that need
container assertions add `await using` and call `.container(name)` — the
first access lazily queries Docker, and a spec that never touches it
pays nothing.

There are two ways to write a spec, and the first is the default.

#### The document — `<case>.spec.yaml`

A terminal session is already a specification: a command, its exit code,
what it printed, what it left on disk. Most of this suite says exactly
that, in a YAML document that lives **beside** the spec it belongs to
(never under `_expected/`, which holds goldens, not scenarios):

```yaml
description: a valid project prints a clean success report
fixture: $FIXTURES/single-agent/
env: isolated
runs:
    - command: check
      exit: 0
      stdout: |4+
          
            ✓  Project is valid
               {{workdir}}/spwn.yaml

      files:
          spwn.yaml: { contains: 'name: demo' }
```

- `fixture:` is `.fixture()`: `$FIXTURES/<name>/` for the shared pool at
  `_fixtures/`, a bare name for a feature-local `_fixtures/`
  overlay, a **list** to layer them in order.
- `env:` names an environment set registered in
  `cli.specification.ts` — today only `isolated`, which moves
  `SPWN_HOME` onto the spec's temp cwd. Inline `KEY=value` also works.
- `runs:` is the session. Every run is asserted, and a non-zero exit does
  **not** end it — which is what `.exec([...])` could never do, since it
  stops at the first failure and keeps only the last output.
- `stdout:`/`stderr:` are byte-exact. `|` keeps the final newline, `|-`
  drops it. Absent asserts an EMPTY stream.
- `files:` asserts what the run left behind, in four forms: `contains`
  (one needle or a list), `equals`, `exists`, `absent`. Each run's
  mapping is read against the working directory as THAT run left it, so
  a path may be `absent` for one run and there for the next —
  `logs/lazily-created-log.spec.yaml` states exactly that.
- The whole [token vocabulary](https://github.com/jterrazz/package-test/blob/main/docs/15-tokens.md)
  works in the streams and in `files:` texts — `{{workdir}}`, `{{path}}`,
  `{{time}}`, `{{iso8601}}`, `{{hex}}`, and `{{string#ref}}` for a value
  that must be the same wherever it reappears.

The runner reports the document's own path as the test file and its
`description:` as the title, so a failing scenario opens where it is
written. `vitest.config.ts` wires the glob through the `literate()`
plugin; the shape of a document is checked by the conventions checker in
`pnpm -C specs lint`, which also has a `--fix` for key order and block
scalars.

**Regenerate a document's streams** with `TEST_UPDATE=1`:

```bash
TEST_UPDATE=1 pnpm -C specs exec vitest run cli/check
```

It rewrites `exit`, `stdout` and `stderr` per run — and nothing else.
The ground, the commands and the `files:` assertions are never touched,
and a token already in place survives by pattern match, so `{{workdir}}`
stays `{{workdir}}` wherever the line moved to. **Read every regenerated
document before committing it**: a value the framework could not
recognise as volatile (a temp path it did not substitute, a random agent
name, a clock) comes back as a literal and has to be tokenised by hand.

#### The chain — `<aspect>.spec.ts`

Reach for code when the format cannot state what the spec is about. Each
file that does says so in its own header; the reasons in this suite are:

| Reason                     | Example                                                                                                                  |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| **containers**             | `world/`, `colony/`, `architect/`, `hooks/` — `.container(name)`, `await using`                                          |
| **structural JSON**        | `check/json-report.spec.ts`, `agent/agent-list.spec.ts` — `result.json` by shape                                         |
| **an absence**             | `dependency/scoped-refs.spec.ts` — no entry may survive bare; `files:` says what a file contains, never what it does not |
| **a count**                | the same file — exactly one list entry after a repeated install                                                          |
| **two runs compared**      | `agent/agent-list.spec.ts` — the same header from two separate invocations                                               |
| **a host shell-out**       | `agent/export.spec.ts` (`tar tzf`), `build/build.spec.ts` (`docker run`)                                                 |
| **host-dependent output**  | `authentication/authentication.spec.ts` — the dashboard reads the operator's keychain                                    |
| **a long-running process** | `web/web.spec.ts` — `.exec(…, { waitFor })` plus a `pgrep` orphan check                                                  |

When only ONE assertion needs code, the session still belongs in a
document: `cli.run('<case>.spec.yaml')` runs it — its ground, its
servers, every run asserted — and resolves with the last run's result,
so the code adds only what it must:

```typescript
test('no manifest entry survives without an explicit scheme', async () => {
    // Given - init followed by three installs mixing bare, local and catalog refs
    const result = await cli.run('explicit-manifest-schemes.spec.yaml');

    // Then - every entry on disk carries its scheme
    const bare = bareEntries(result.file('spwn/agents/neo/agent.yaml').content);
    expect(bare, `manifest carries bare entries: ${bare?.join(', ')}`).toBeNull();
});
```

**Container-asserting pattern** — the same `cli`, plus `.container(...)`:

```typescript
import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

test('up provisions a running world', async () => {
    // Given - the docker-pilot world brought online
    await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

    // Then - the container is live with its agent home laid down
    expect(result.exitCode).toBe(0);
    expect(result.stderr).toContain('Created container');

    const neo = result.container('neo');
    expect(neo.running).toBe(true);
    expect(neo.file('/agents/neo/CLAUDE.md').exists).toBe(true);
});
```

- **`await using`** whenever a spec might spawn containers. The dispose
  hook force-removes every container tagged with this run's id so
  parallel runs never collide. Harmless no-op for specs that spawn
  nothing — and required by rule B5 on a docker-aware runner. A document
  needs none of this, which is the other reason a container spec stays a
  chain.
- `result.container('<world-key>')` resolves by the
  `sh.spwn.world.config` label — the key declared under `worlds.` in
  `spwn.yaml`, not the sometimes-empty `sh.spwn.world.name`.
- `result.container(name).file(path)` / `.exec(cmd)` / `.inspect.value` /
  `.stdout` / `.stderr` use the same accessor API as the host-side
  `result` — no new vocabulary.
- Follow-up CLI commands that need a container id (e.g.
  `spwn world inspect <id>`) get it via the world-id label.
- Always use the `docker-pilot` fixture (minimal agent without
  `spwn:python`). `single-agent` fails to spawn because the base image
  lacks `pip3`.
- Banners (`Created container`, `Agent is alive`, `Destroyed`,
  `World destroyed`) go to **stderr**, not stdout — spwn follows the Unix
  convention of data-on-stdout / status-on-stderr.

## Adding New Tests

### Go Unit Test

1. Create `your_file_test.go` next to the source file.
2. Use standard `testing.T` patterns.
3. Use table-driven tests where appropriate.
4. Run with `make test` or `go test ./...` in the module directory.

### Go E2E Test

1. Create `your_feature_test.go` in `../packages/world/tests/e2e/`.
2. Add `//go:build e2e` build tag at the top.
3. Use `setup.NewSpawnBuilder(t)` to create test infrastructure.
4. Follow GIVEN/WHEN/THEN comment structure.
5. Run with `make test-go-e2e`.

### TypeScript E2E Test

Write a **document** unless the spec needs one of the reasons listed
under [the chain](#the-chain--aspecttestts).

1. Create `cli/<domain>/<case>.spec.yaml` — `<case>` in
   kebab-case, naming the scenario, never the bare folder name and never
   carrying the words `test`, `spec` or `cli` the suffix already does.
2. State the ground (`fixture:`, `env:`), then `runs:` with a `command:`
   and a guessed `exit:` per step, plus any `files:` assertion.
3. `TEST_UPDATE=1 pnpm -C specs exec vitest run <path>` fills in the exit
   codes and the streams. **Read the result** and tokenise anything
   volatile the framework left literal — a temp path it did not
   substitute, a random agent name, a clock.
4. Run it again without the flag. It must pass twice in a row.

For a chain instead:

1. Create `cli/<domain>/<aspect>.spec.ts` and open its
   docblock with the reason the format cannot carry it.
2. Import `cli` from `../cli.specification.js`.
3. If only one ASSERTION needs code, put the session in a document and
   call `cli.run('<case>.spec.yaml')`.
4. Use `// Given -` and `// Then -` narration, in that order.
5. For Docker specs, always `await using` (rule B5).
6. Run with `pnpm -C specs exec vitest run <glob>`.

## Test File Naming Conventions

| Layer       | Pattern                                         | Example                   |
| ----------- | ----------------------------------------------- | ------------------------- |
| Go unit     | `*_test.go` (next to source)                    | `manifest_test.go`        |
| Go E2E      | `*_test.go` (in `../packages/world/tests/e2e/`) | `spawn_test.go`           |
| TS document | `<case>.spec.yaml` (in `cli/<domain>/`)         | `valid-project.spec.yaml` |
| TS chain    | `<aspect>.spec.ts` (in `cli/<domain>/`)         | `json-report.spec.ts`     |

The suffix says the KIND (`@jterrazz/test` rule C12): under `specs/<facet>/`
a chain meets the assembled product and is a `.spec.ts`; `.test.ts` is the
unit, beside its module. A first level that is NOT a facet is a repository
suite — it covers a tree rather than a product reached through an entry, and
its word stays `.test.ts`. Two suites stand here: `lint/`, the repo-wide
source guards, and `shapes/`, the wire payloads the Go API promises
(`shapes/activity/api.test.ts`). One file keeps `.test.ts` under a facet:
`cli/smoke/upgrade.e2e.test.ts`, which drives the binary with raw `execSync`
and so reaches no runner (C18, recorded on the ratchet). Both vitest includes
state the two suffixes for that reason.

Under this package, what a spec STANDS ON carries a leading underscore —
`_fixtures/` and `_expected/` — while a spec's own folder never does, and
the framework resolves no other name.

The pool at `_fixtures/` is what SEVERAL leaves share: a directory
only one spec folder reaches for belongs beside that folder as
`<leaf>/_fixtures/<name>/`, which `pnpm -C specs lint` reports and
`npx jterrazz-test-check specs --fix` moves.

## Test Function Naming

- **Go**: `TestFeature_Scenario` (e.g., `TestSpawn_CreatesRunningContainer`)
- **TypeScript document**: the `description:` line IS the vitest title —
  one lowercase line, no trailing period, unique within its folder.
- **TypeScript chain**: `test("scenario description")`, same rules.

## Vitest Configuration

`vitest.config.ts` is one project over `cli/**`, `shapes/**` and
`lint/**`, with the `literate()` plugin adding
`cli/**/*.spec.yaml` to the include and binding every document to
`cli/cli.specification.ts`.

- `testTimeout: 120_000` — the docker-aware specs spawn real containers;
  CLI-only specs finish in milliseconds, so the upper bound is harmless.
- `hookTimeout: 180_000` — setup hooks and the first container boot are
  slow on cold CI runners.
- `fileParallelism: true` — spwn scopes every world lookup by the
  `SPWN_TEST_LABEL` the framework injects per spec, and the framework
  force-removes each run's containers by that label at scope exit, so
  two parallel specs both spawning a "neo" world route to their own
  container.
- The real-build smoke spec runs separately, via
  `vitest.smoke.config.ts` (`make test-smoke`).
