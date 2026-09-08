# Test Architecture

This document explains **how** spwn is tested — the layers, where files live, the patterns each layer uses, the governance that prevents drift, and the cookbook for adding new tests.

For **how to run** tests, see [README.md](README.md). For the original design rationale and open issues, see [`docs/03-testing.md`](../docs/03-testing.md). For manual QA passes that complement this suite, see [`docs/qa/`](../docs/qa/).

---

## Philosophy

1. **Tests follow architecture.** Every package and surface has a default proof type. Adding a new runtime/route/command without declaring its tests fails CI.
2. **Use real boundaries.** Real parsers, real file systems in temp dirs, real HTTP servers, real Docker, real Playwright. Mock vendors, not your own system.
3. **Mock at the network boundary, not on the global.** Go uses `httptest.Server`. Web uses MSW. We do not stub global `fetch`.
4. **Local equals CI.** Make targets are the source of truth. CI calls Make targets — never hand-written command variants.
5. **No fixed sleeps.** Web tests wait on locator state, API readiness, or `expect.poll`. CLI tests use the harness's deterministic waits.
6. **Coverage is a signal, not a goal.** We add thresholds where they make architectural sense; we don't chase 100%.
7. **Regression surfaces get golden or contract tests.** Runtime renderer output, CLI JSON output, generated docs, API schemas, and catalog manifests are byte-compared.
8. **Tree screams the product, not the framework.** Top-level tests folders read like the system (`world/`, `agent/`, `build/`), not like the test runner. Infrastructure (`_contracts/`, `_simulators/`, `_setup/`) is underscore-prefixed so it sorts above features and is unmistakable.

---

## The Pyramid

```
┌──────────────────────────────────────────────────────────────────┐
│ L0  Static Architecture Gates                                    │
│     Test contracts, depguard, lint                               │
│     ⏱  <1s   📦  No runtime infra                                │
│     ▶  make test-contracts, make lint                            │
├──────────────────────────────────────────────────────────────────┤
│ L1  Pure Unit                                                    │
│     Go *_test.go (next to source) + web vitest + gate vitest     │
│     ⏱  ~1s    📦  None (t.TempDir, mocks for vendor SDKs only)   │
│     ▶  make test, make test-web-unit, make test-gate-node        │
├──────────────────────────────────────────────────────────────────┤
│ L2  Contract & Golden                                            │
│     Runtime renderer goldens, CLI JSON fixtures, scaffold trees, │
│     compile cache invariants, hooks translation                  │
│     ⏱  ~5s   📦  Embedded fixtures + testdata/                   │
│     ▶  make test (folded into Go unit tier)                      │
├──────────────────────────────────────────────────────────────────┤
│ L3  Local Integration                                            │
│     API via production Handler() · Web via MSW · Gate over HTTP  │
│     ⏱  ~10s  📦  In-process httptest / local servers             │
│     ▶  make test, make test-web-unit, make test-gate-node        │
├──────────────────────────────────────────────────────────────────┤
│ L4  Docker E2E                                                   │
│     L4a Go: packages/world/tests/e2e (//go:build e2e)            │
│     L4b TS: tests/specs/cli/<domain>/ (compiled bin)             │
│     ⏱  ~30s–2m  📦  Real Docker + spwn-test:latest               │
│     ▶  make test-go-e2e, make test-compile-e2e, make test-cli        │
├──────────────────────────────────────────────────────────────────┤
│ L5  Web E2E (Playwright)                                         │
│     Real Next.js + real Go API + real Chromium + real Docker     │
│     Isolated SPWN_HOME + SPWN_PROJECT + SPWN_TEST_LABEL          │
│     ⏱  ~2m   📦  Docker + browser                                │
│     ▶  make test-web                                             │
├──────────────────────────────────────────────────────────────────┤
│ L6  Real-runtime Smoke                                           │
│     Real Claude/Codex CLIs against real provider APIs            │
│     ⏱  ~30s  📦  Live credentials + budget cap                   │
│     ▶  make test-smoke                                           │
└──────────────────────────────────────────────────────────────────┘
```

Each layer denies what's above: a unit test never spawns a container; a Docker E2E never makes a live API call. Lower layers run on every PR; higher layers run conditionally or on push to main.

---

## Repository Test Layout

Two organising principles:

- **Go tests are colocated** with the code they prove (idiomatic Go, tooling expects it).
- **TS tests live under `tests/`**, organised by feature: top-level folders scream the product (`cli/`, `web/`), and within each surface the structure mirrors the surface's own grammar (CLI uses noun-verb; web uses domain-feature). Infrastructure folders carry a leading underscore (`_contracts/`, `_simulators/`, …) so they sort above features and never get mistaken for a feature.

```
spwn/
├── apps/
│   ├── api/
│   │   └── *_test.go              ← API handler tests via production Handler()
│   ├── cli/
│   │   ├── cli_test.go            ← Cobra flag/help tests
│   │   └── <subcmd>/*_test.go     ← Per-command unit tests
│   ├── gate/
│   │   ├── package.json           ← Workspace root for gate tests
│   │   ├── vitest.config.mjs
│   │   └── sdk/
│   │       └── index.test.mjs     ← SDK + sidecar tests (local HTTP server)
│   └── web/
│       ├── vitest.config.ts
│       └── src/
│           ├── lib/__tests__/
│           │   └── stream-chat.test.ts   ← Network behaviour via MSW
│           └── test/
│               ├── msw/
│               │   ├── handlers.ts
│               │   └── server.ts
│               └── setup.ts              ← MSW startup, beforeEach reset
│
├── packages/
│   ├── <module>/
│   │   ├── *_test.go              ← L1 unit tests (alongside source)
│   │   └── internal/<sub>/*_test.go
│   ├── runtimes/
│   │   ├── claudecode/render_test.go
│   │   ├── codex/render_test.go
│   │   ├── gemini/render_test.go
│   │   ├── golden_test.go         ← L2: byte-compare against testdata/
│   │   └── testdata/              ← Embedded fixtures + expected output
│   │       ├── <case>/
│   │       │   ├── input/         ← agent.yaml + spwn.yaml + skills
│   │       │   └── output_<runtime>/
│   │       │       ├── AGENTS.md  (codex) or CLAUDE.md (claude)
│   │       │       ├── .codex/    or .claude/
│   │       │       └── …
│   ├── world/
│   │   └── tests/e2e/             ← L4a Docker E2E (//go:build e2e)
│   │       ├── setup/
│   │       │   ├── context.go     ← TestContext: temp SPWN_HOME, label, cleanup
│   │       │   ├── builders.go    ← SpawnBuilder fluent DSL
│   │       │   └── assertions.go  ← StateAssertion / ContainerAssertion / …
│   │       └── *_test.go
│   └── compile/
│       ├── e2e/                   ← L4 image-build E2E
│       └── builder_from_base_test.go   ← L2 cache-invariants
│
├── tests/                         ← TypeScript E2E + Web E2E + governance
│   ├── README.md                  ← How to run, and how to write a spec
│   ├── ARCHITECTURE.md            ← This file
│   │
│   ├── specs/                     ← L4b CLI E2E, one folder per domain
│   │   ├── cli/
│   │   │   ├── cli.specification.ts   ← The single runner (`cli`), docker-aware
│   │   │   ├── check/
│   │   │   │   ├── valid-project.spec.yaml   ← A document: one session, asserted whole
│   │   │   │   ├── json-report.test.ts       ← A chain: structural JSON
│   │   │   │   ├── manifest-grammar.test.ts  ← A chain: absences, bridged on documents
│   │   │   │   ├── _fixtures/                ← Feature-local overlays, this leaf's own
│   │   │   │   └── _expected/                ← Only what a document cannot carry
│   │   │   └── agent/, world/, init/, build/, dependency/, logs/, …
│   │   ├── _fixtures/             ← Shared pool, reached via `$FIXTURES/<name>/`
│   │   │   └── docker-pilot/, codex-pilot/, single-agent/, empty/, …
│   │   └── lint/
│   │       └── guards/            ← Repo-wide source guards (plain vitest, no runner)
│   │
│   ├── web/                       ← L5 Playwright (one folder per feature)
│   │   ├── playwright.config.ts
│   │   ├── _setup/
│   │   │   ├── global-setup.ts    ← make build (sidecar binary)
│   │   │   └── global-teardown.ts ← Container cleanup by label
│   │   ├── _fixtures/
│   │   │   └── app.ts             ← Playwright `test` extended with api + app helpers
│   │   ├── agents/, examples/, navigation/, system/, worlds/
│   │
│   ├── _contracts/                ← L0 governance (the registry)
│   │   ├── api-routes.yaml
│   │   ├── runtimes.yaml
│   │   ├── cli-commands.yaml
│   │   ├── catalog.yaml
│   │   ├── web-routes.yaml
│   │   ├── node-packages.yaml
│   │   └── assert-contracts.mjs   ← One script that fails CI on drift
│   │
│   ├── _simulators/               ← Runtime simulators (vendor CLI stubs)
│   │   ├── Dockerfile.test        ← Builds spwn-test:latest
│   │   ├── claude/mock.sh         ← `claude` simulator inside test image
│   │   └── codex/mock.sh          ← `codex` simulator
│   │
│   ├── _fixtures/                 ← Shared seed dirs the GO E2E suite reads
│   │   └── testdata/              ← Looked up by setup.TestdataDir()
│   │
│   ├── _smoke/                    ← Real-build cross-cutting smoke tests
│   ├── _catalog/                  ← Catalog-bundle goldens (Go module)
│   │
│   ├── vitest.config.ts           ← One project + the literate() plugin
│   ├── vitest.smoke.config.ts
│   ├── package.json, tsconfig.json
│   ├── oxfmt.config.ts, oxlint.config.ts
│   └── node_modules/
│
└── .github/workflows/
    ├── validate.yaml              ← lint, test, contracts, web-unit, gate-node, e2e
    └── release.yaml
```

### Why this shape

- **CLI tree** is one folder per DOMAIN of the CLI (`specs/cli/agent/`, `specs/cli/world/`, `specs/cli/check/`), and inside it one file per scenario: `specs/cli/agent/forked-agent.spec.yaml` reads like the session it describes. The folder holds its scenarios, the chains that need code, and — only where those chains need them — a `_fixtures/` overlay and an `_expected/` golden.
- **Ground carries a leading underscore.** Under `specs/`, what a spec STANDS ON is `_fixtures/` and `_expected/`; a spec's own folder never is, so a domain and its material can never be mistaken for each other. The framework resolves no other name, and `c13-underscored-ground` fails a bare one.
- **The pool is what SEVERAL leaves share.** A directory of `specs/_fixtures/` reached from exactly one spec folder belongs beside that folder as `<leaf>/_fixtures/<name>/`, because a one-reader fixture parked in the pool reads as shared ground nobody dares change; `c14-pool-fixture-shared` names it and `jterrazz-test-check --fix` moves it.
- **Web tree** mirrors web's UI grouping: `tests/web/<domain>/<feature>/<feature>.spec.ts` reads like the surface ("worlds/list", "agents/detail", "navigation/sidebar"). One feature = one folder = one spec file.
- **No false symmetry.** CLI features (`build`, `check`, `auth`, `gate`) often have no web counterpart, and vice versa. The trees don't pretend otherwise; the contract registry (`_contracts/`) is what ties them together when an underlying surface has both.
- **Underscore-prefixed infrastructure.** `_contracts/`, `_simulators/`, `_fixtures/`, `_smoke/`, `_catalog/` sort above feature folders alphabetically and are visually distinct so no one mistakes them for a domain. The convention the TS suite adopted by hand at the `tests/` root is the same one the framework now enforces inside `specs/`; the `tests/_fixtures/` at this level is unrelated to the CLI pool and holds only the seed dirs the GO E2E suite reads.

---

## Layer 0 — Static Architecture Gates

The governance layer. Catches drift between code and tests _before_ anything runs.

### Test contracts: `tests/_contracts/`

YAML registries declare every "test-bearing surface" in the codebase:

- **api-routes.yaml** — every `/api/*` route → at least one Go server test (and optionally a Playwright spec).
- **runtimes.yaml** — every adapter (claude-code, codex, gemini) → render+spawn+tool unit tests + golden output dir + docs.
- **cli-commands.yaml** — every `spwn` Cobra command → help snapshot + at least one behavior spec.
- **catalog.yaml** — every catalog entry → manifest parses, dependencies resolve, smoke test exists.
- **web-routes.yaml** — every Next.js route → Playwright spec or component test.
- **node-packages.yaml** — `apps/web` and `apps/gate` → has a `test` script wired into make.

`tests/_contracts/assert-contracts.mjs` walks each YAML and fails CI if:

- a referenced test/doc file doesn't exist on disk
- a runtime's `adapter.go` lacks the declared facet (`Tool:`, `Render:`, `Spawn:`)
- a runtime's `goldenOutput` dir is missing for any embedded test case
- a Node package referenced by `node-packages.yaml` is missing its test script

```bash
make test-contracts        # runs node tests/_contracts/assert-contracts.mjs
```

### Depguard (Go imports): `.golangci.yml`

Seven layers, each denies imports from the layer above (see [CLAUDE.md](../CLAUDE.md) "Dependency Graph"):

```
L1 platform, container          → (nothing above)
L2 activity, auth, upgrade      → L1
L3 dependency, agent            → L1-L2
L4 project                      → L1-L3
L5 compile, transpile, runtimes → L1-L4
L6 world, architect             → L1-L5
L7 apps/cli, apps/api           → anything
```

Adding `import "spwn.sh/packages/world"` from `packages/agent/` fails `make lint`.

### Lint: `make lint`

`make lint` invokes `go vet` across every workspace module + `pnpm -r lint` (oxlint + oxfmt + knip). Lint, formatting, and unused-code checks are PR-blocking.

---

## Layer 1 — Pure Unit Tests

Fast, no Docker, no real network, no real `~/.spwn`.

### Go: `*_test.go` next to source

```go
// packages/platform/paths_test.go
func TestWorldsDir_RespectsProjectRoot(t *testing.T) {
    SetProjectRoot("/tmp/test-spwn")
    t.Cleanup(func() { SetProjectRoot("") })

    got := WorldsDir()
    want := "/tmp/test-spwn/worlds"
    if got != want {
        t.Errorf("WorldsDir() = %q, want %q", got, want)
    }
}
```

**Rules:**

- No Docker, no live network.
- Use `t.TempDir()` and `t.Setenv("SPWN_HOME", ...)` — never the user's home.
- Table-driven tests for rule sets.
- A failing test is a violation of the contract documented in the package's `README.md`.

### Web: `apps/web/src/**/*.test.ts`

Vitest with happy-dom. Pure logic and component tests run here; **network behavior is tested at the HTTP boundary via MSW**, not by stubbing `fetch`.

```ts
// apps/web/src/lib/__tests__/stream-chat.test.ts
import { http, HttpResponse } from 'msw';
import { server } from '@/test/msw/server';

test('falls back to JSON when SSE returns 404', async () => {
    server.use(
        http.post('/api/chat/stream', () =>
            HttpResponse.json({ error: 'not found' }, { status: 404 }),
        ),
        http.post('/api/chat', () => HttpResponse.json({ message: 'hi' })),
    );

    const result = await streamChat('hello');
    expect(result).toBe('hi');
});
```

### Gate: `apps/gate/sdk/*.test.mjs`

The Gate SDK is CommonJS but tests are ESM via `createRequire` (Vitest 4 doesn't allow `require('vitest')` from CJS). Tests cover MCP manifest generation, CLI dispatch, JSON-RPC method registration, and HTTP error propagation against a local `http.createServer` simulating the sidecar.

### Automations: `packages/automation/*_test.go`

The automation engine is intentionally testable without Docker, fsnotify, or real time. Four collaborator interfaces (`Clock`, `Dispatcher`, `ReceiptWriter`, `StateStore`) plus the `RawFSSource` for the watcher all have memory/fake implementations alongside the production ones. Tests drive the engine directly:

- **`FakeClock.Advance(d)`** moves time forward and fires every pending `After` channel whose deadline elapsed — cron schedules tick deterministically.
- **`FakeFSSource.Emit(ev)`** + `watcher.handle(ctx, ev)` injects synthetic filesystem events; debounce + pattern + recursive logic runs without fsnotify.
- **`MockDispatcher.Hold`** blocks each Dispatch on a channel — lets concurrency tests assert "exactly one in flight per agent" without timing flakes.
- **`MemoryReceiptWriter` / `MemoryStateStore`** keep receipt + last-fired state in slices/maps for trivial assertion.

72 tests cover the engine; 12 more cover the architect-side dispatcher + command resolver; 13 cover the CLI helpers. All `-race` clean.

---

## Layer 2 — Contract & Golden Tests

Lock the byte-level shape of outputs that users (or other systems) depend on.

### Runtime golden tests: `packages/runtimes/golden_test.go`

For each subdirectory under `packages/runtimes/testdata/<case>/`, the test:

1. Loads `input/spwn.yaml` + `input/agents/<name>/agent.yaml`
2. Runs the adapter's `Render()` for every facet (claude-code, codex, gemini)
3. Compares the produced tree against `output_<runtime>/` byte-for-byte

```
packages/runtimes/testdata/minimal-single-agent/
├── input/
│   ├── spwn.yaml
│   └── agents/neo/
│       ├── agent.yaml
│       └── SOUL.md
├── output_claude_code/
│   ├── CLAUDE.md
│   └── .claude/
│       ├── settings.json
│       └── skills/
└── output_codex/
    ├── AGENTS.md
    └── .codex/
        ├── config.toml
        └── hooks.json
```

Regenerate goldens with `JTERRAZZ_TEST_UPDATE=1 go test ./packages/runtimes/...`.

### CLI goldens: the document's own streams

Most CLI output is goldened INSIDE the spec document that produced it — `stdout:` and `stderr:` are byte-exact block scalars in the `<case>.spec.yaml`, regenerated with `TEST_UPDATE=1` (see L4b). A sibling `_expected/` survives only for the assertions a document cannot carry: `_expected/<name>.json` for the structural `result.json` comparisons under `check/`, `agent/` and `world/`.

### Compile cache invariants: `packages/compile/builder_from_base_test.go`

Builds the tar context that goes to Docker without actually running Docker. Asserts:

- The `Dockerfile` ARGs are present
- Project files land at the right path inside the tar
- Cache labels change when policy/runtime/Dockerfile change (and _don't_ change otherwise)

---

## Layer 3 — Local Integration Tests

Real subsystems wired together, but vendor APIs are simulated at the network boundary.

### API: `apps/api/server_test.go`

The pivot is that **tests share the production handler**:

```go
// apps/api/server.go
func (s *Server) Handler() http.Handler {
    mux := http.NewServeMux()
    s.registerRoutes(mux)
    return mux
}

// apps/api/server_test.go
srv := httptest.NewServer(server.Handler())
defer srv.Close()
res, _ := http.Get(srv.URL + "/api/health")
```

A new route is added by editing `registerRoutes` — _both_ production and tests pick it up. There is no parallel router definition that can drift.

### Web client: MSW

`apps/web/src/test/msw/server.ts` starts an MSW server in `setupFiles` and resets handlers between tests. Network-behavior tests (SSE streams, JSON fallback, HTTP errors, network errors, fallback URLs) all go through real `fetch` and an HTTP-level interceptor.

### Gate: local `http.createServer`

Gate sidecar tests stand up a real HTTP server in-process to exercise the SDK's request shape, retry logic, and error propagation without a Playwright browser pool.

---

## Layer 4 — Docker E2E

### L4a · Go: `packages/world/tests/e2e/`

Build tag `//go:build e2e`. Excluded from `make test`; runs only via `make test-go-e2e`.

**Pattern (fluent builder + assertion chain):**

```go
//go:build e2e
package e2e

import (
    "testing"
    "spwn.sh/packages/world/tests/e2e/setup"
)

func TestSpawn_CreatesRunningContainer(t *testing.T) {
    chain := setup.NewSpawnBuilder(t).
        WithAgent("neo").
        WithProject("docker-pilot").
        Execute()

    chain.ExpectState(func(s *setup.StateAssertion) {
        s.WorldCount(1)
        s.HasAgent("neo")
    })

    chain.ExpectContainer(func(c *setup.ContainerAssertion) {
        c.IsRunning()
        c.HasMount("/agents")
        c.FileExists("/agents/neo/CLAUDE.md")
    })
}
```

**Setup primitives** (`packages/world/tests/e2e/setup/`):

- `NewTestContext(t)` — creates `t.TempDir()` SPWN_HOME, registers `t.Cleanup()` to destroy every world, attaches a unique label so parallel runs cannot collide.
- `SpawnBuilder` — fluent DSL.
- `ContainerAssertion`, `MindAssertion`, `MockAssertion`, `JournalAssertion`, etc. — every observable surface has its own assertion type.
- `WaitFor(t, timeout, interval, desc, cond)` — replaces `time.Sleep`.

The simulator inside `spwn-test:latest` writes its observations as JSON to `/tmp/claude-mock.json` so `MockAssertion` can read it back: `m.SawMind()`, `m.SawClaudeMD()`, `m.SawSkill("focus")`. The shared seed dirs live at `tests/_fixtures/testdata/<case>/` and are looked up by `setup.TestdataDir()`.

### L4b · TypeScript: `tests/specs/cli/<domain>/`

Exercise the compiled `.artifacts/go/spwn` from a user's perspective through one runner, `tests/specs/cli/cli.specification.ts` (`cli`), which is docker-aware and carries the `env` registry a document names by bare word.

A spec is written in one of two forms, and the first is the default.

**The document** — `<case>.spec.yaml`, beside the specs it belongs to. One scenario per file: the ground it stands on, the session, and what the session left on disk.

```yaml
description: an incomplete lockfile is flagged by name
fixture: $FIXTURES/docker-pilot/
runs:
    - command: install spwn:unix
      exit: 0
      stdout: |4
            ✓  installed spwn:unix
               1 agent updated, spwn.lock pinned
    - command: check
      exit: 1
      stdout: |4+
          
            {{workdir}}/spwn.yaml

            errors
              •  "spwn:git" is not recorded in spwn.lock
                 spwn.yaml#deps
                 → run `spwn install spwn:git` to sync the lockfile

            1 error(s), 0 warning(s), 0 info

```

Every run is asserted, and a non-zero exit does not end the session — which is what `.exec([...])` never could, since it stops at the first failure and keeps only the last output. `files:` adds on-disk assertions in four forms (`contains`, `equals`, `exists`, `absent`), token-aware like the streams. `vitest.config.ts` collects the glob through the `literate()` plugin, so a failing scenario opens in the document itself, under its `description:`.

Regenerate a document's `exit`, `stdout` and `stderr` — and nothing else — with `TEST_UPDATE=1`; the full workflow, including what to read before committing a regenerated document, is in [README.md](README.md#the-document--casespecyaml).

**The chain** — `<aspect>.test.ts`, for what the format cannot state: containers, structural JSON, an absence, a count, two runs compared to each other, a host shell-out, host-dependent output, a long-running process. Each such file opens with the reason in its docblock. When only one ASSERTION needs code, the session still belongs in a document and `cli.run('<case>.spec.yaml')` runs it whole:

```ts
test('the exported archive carries every mind layer', async () => {
    // Given - the create + export session, stated in the document
    const result = await cli.run('exported-archive.spec.yaml');

    // Then - the tarball listing carries the Soul plus the two Mind layer dirs
    const listing = execSync(`tar tzf ${join(result.filesystem.cwd, 'neo.tar.gz')}`, {
        encoding: 'utf8',
    });
    expect(listing).toContain('SOUL.md');
});
```

**Container-asserting pattern** (Docker):

```ts
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

- `await using` — the dispose hook force-removes every container tagged with this run's id so parallel runs never collide. Required by rule B5 on a docker-aware runner, and a harmless no-op for a chain that spawns nothing. A document needs none of it, which is the other reason a container spec stays a chain.
- `result.container('neo').file(...)` / `.exec(...)` / `.inspect.value` — same accessor API as the host-side `result`. No new vocabulary.
- `.fixture('$FIXTURES/<name>/')` spreads a project from the shared pool at `specs/_fixtures/`; a bare name resolves to the feature's own `_fixtures/` overlay, and chained calls layer in order.

---

## Layer 5 — Web E2E (Playwright)

End-to-end via real Chromium against real Next.js + real Go API + real Docker.

### Tree shape

Each feature owns a folder that holds exactly one spec file:

```
tests/web/<domain>/<feature>/<feature>.spec.ts
```

- `<domain>` is the high-level slice of the UI (`worlds`, `agents`, `examples`, `navigation`, `system`).
- `<feature>` is one cohesive surface inside it (`list`, `detail`, `lifecycle`, `gallery`, `sidebar`, `command-palette`, `api-health`).
- The folder is symmetric with the CLI tree: feature-named files inside a feature-named folder. If a feature ever needs its own seeds or fixtures, they sit beside the spec without restructuring.

### Isolation: `tests/web/playwright.config.ts`

Every run gets:

- **`SPWN_HOME`** = `tmpdir()/spwn-web-e2e-XXXX` (so `~/.spwn` is never touched)
- **`SPWN_PROJECT`** = `tmpdir()/spwn-web-e2e-project-XXXX` (the cwd for `spwn web`)
- **`SPWN_TEST_LABEL`** = `web-e2e-<timestamp>-<rand>` (Docker label for cleanup)
- **`.onboarding-complete`** marker so the welcome wizard is skipped
- **`SPWN_BASE_IMAGE=spwn-test:latest`** so worlds spawn against the simulator image

The fixture project is hydrated from `catalog/matrix/` and `catalog/startup/` so the API has agents and inline `spwn.yaml#worlds` to spawn from.

The Playwright config sets `testDir: '.'` with `testMatch: ['**/*.spec.ts']` and `testIgnore: ['_setup/**', '_fixtures/**']` so adding a new feature folder requires zero config change.

### Fixture: `tests/web/_fixtures/app.ts`

Extends Playwright's `test` with two helpers:

```ts
test('selecting a planet shows agent details', async ({ page, api, app }) => {
    await api.installExample('matrix'); // POST /api/examples/matrix/install
    await api.spawnWorld('matrix', 'Neo'); // POST /api/worlds
    await page.goto('/');
    await app.waitForWorlds(); // Locator-based, not setTimeout
    await app.selectWorld('matrix');
    await expect(page.getByText('Neo').first()).toBeVisible();
});
```

- `api.*` — direct calls to the Go API (faster than UI for setup).
- `app.*` — page-object methods named in user language (`goToAgents`, `selectWorld`, `enterWorld`, `openCommandPalette`).
- `await using` is implicit: the fixture's teardown destroys every world it spawned.
- Spec import path: `import { expect, test } from '../../_fixtures/app.js';` (from the feature folder, `../../` reaches the web root).

### Cleanup: `tests/web/_setup/global-teardown.ts`

Removes only containers that match `SPWN_TEST_LABEL`. The dev's local containers (and the always-on `spwn-gate`) are never touched.

### Rules

- **Zero `waitForTimeout`** — replaced with locator visibility, `expect.poll`, or API readiness probes. The contract checker fails CI if it spots one.
- **Page objects expose user-language methods**, not raw selectors.
- **Tests are independent**: each test does its own setup; there is no global "after each test installs matrix" assumption.
- **One feature per file.** When a describe block grows past one cohesive feature, split it into a sibling folder.

---

## Layer 6 — Real-Runtime Smoke

Real Claude/Codex CLIs, real provider APIs. Currently:

- `make test-smoke` — `tests/_smoke/init-up` and `tests/_smoke/upgrade` exercise `spwn init` → `spwn up` against a live build (no live LLM call).
- **Planned** (Phase 9 of the test architecture plan): `tests/real-runtime/` with `SPWN_REAL_RUNTIME=1` opt-in for live `spwn agent talk` against Claude/Codex/Gemini APIs, with hard timeout + cleanup.

---

## Make Targets and CI

The Makefile is the source of truth. CI calls Make targets — never raw `go test` or `vitest`. There is **no aggregate meta-target** (`test-pr`, `test-release`, …) by design — `.github/workflows/validate.yaml` enumerates the granular targets and is itself the aggregate. To know what CI runs, read the workflow.

### Targets

| Target                     | Layer    | What it runs                                                       |
| -------------------------- | -------- | ------------------------------------------------------------------ |
| `make lint`                | L0       | `go vet` + `pnpm -r lint` (oxlint + oxfmt + knip)                  |
| `make test`                | L1+L2+L3 | `go test ./...` across every workspace module                      |
| `make test-contracts`      | L0       | `node tests/_contracts/assert-contracts.mjs`                       |
| `make test-web-unit`       | L1+L3    | `pnpm -C apps/web test` (vitest + MSW)                             |
| `make test-gate-node`      | L1+L3    | `pnpm -C apps/gate test`                                           |
| `make test-cli`            | L4b      | `pnpm -C tests exec vitest run` (full CLI E2E)                     |
| `make test-go-e2e`         | L4a      | Go world E2E with `//go:build e2e`                                 |
| `make test-compile-e2e`    | L4       | Image-build E2E in `packages/compile/e2e`                          |
| `make test-web`            | L5       | Playwright (depends on `make build` + `make test-image`)           |
| `make test-smoke`          | L6       | Real-build init→up→probe                                           |
| `make test-pkg PKG=<name>` | L1       | Verbose go test for one module                                     |
| `make test-image`          | infra    | Builds `spwn-test:latest` from `tests/_simulators/Dockerfile.test` |

### CI: `.github/workflows/validate.yaml`

One job per Make target. Every job sets up Go + pnpm and calls `make <target>`. If CI ever runs commands that local Make doesn't, that's a bug.

```
PR jobs:        lint, test, test-contracts, test-web-unit, test-gate-node,
                test-cli, test-go-e2e, test-compile-e2e, build, web-build
Push to main:   + test-smoke, test-web
```

---

## Runtime Simulators (Mock Vendors)

E2E tests cannot call real Claude/Codex on every PR — too slow, costs money, can flake. Instead, the test image (`spwn-test:latest`) ships **simulators** under `tests/_simulators/` that follow the same protocol as the real CLIs.

### `tests/_simulators/claude/mock.sh`

Installed as `/usr/local/bin/claude` inside the test image. Behavior:

1. Accepts and ignores real Claude flags (`--session-id`, `--resume`, `--print`, …).
2. Inspects the container (`/agents/<name>/CLAUDE.md`, `/workspaces`, `/world/knowledge/`).
3. Writes JSON observations to `/tmp/claude-mock.json` for `MockAssertion` to read.
4. Optionally writes to `/workspaces/workspace0/mock-output.txt` to prove write access.
5. Supports `--exit-code` and `--sleep` for failure/timeout testing.

### `tests/_simulators/codex/mock.sh`

Same idea for Codex. Accepts `codex exec --json`, `codex exec resume <session-id>`, and writes session IDs to a file the architect can resume from. Supports `AGENTS.md`/`.codex/config.toml` introspection.

### Why "simulator" not "mock"

These scripts are protocol contracts. If the real Codex CLI changes its resume syntax, the simulator must update too — and a contract test catches the drift before E2E does. They're not loose mocks; they're executable specs of the vendor protocol we depend on. Putting them under `_simulators/` (separate from `_fixtures/`) makes that distinction visible.

---

## Cookbook

### Add a Go unit test

1. Create `your_feature_test.go` next to `your_feature.go`.
2. Use `t.TempDir()` and `t.Setenv("SPWN_HOME", ...)` for any filesystem state.
3. Run `make test` (or `make test-pkg PKG=<module>` for verbose).
4. If touching a domain module, update its `README.md` if behavior changed.

### Add a Go E2E test

1. Create `tests/e2e/your_feature_test.go` with `//go:build e2e` at the top.
2. Use `setup.NewSpawnBuilder(t)` to spawn a world.
3. Follow GIVEN/WHEN/THEN comment structure.
4. Run `make test-go-e2e`.

### Add a CLI E2E test

1. Pick a folder: `tests/specs/cli/<domain>/`.
2. Write a document, `<case>.spec.yaml`: the ground (`fixture:`, `env:`), the `runs:`, and any `files:` assertion.
3. `TEST_UPDATE=1 pnpm -C tests exec vitest run <path>` fills in the exit codes and the streams; read the result, tokenise what stayed literal, and run it again clean.
4. Reach for a chain (`<aspect>.test.ts`) only for what the format cannot state — containers, structural JSON, an absence, a count, two runs compared, a host shell-out. Open the file with the reason, use `await using` if any container might spawn, and put the session in a document called through `cli.run()` when only one assertion needs code.
5. Add the command to `tests/_contracts/cli-commands.yaml`.

### Add a Web E2E test

1. Pick a folder: `tests/web/<domain>/<feature>/`. Create it if the feature is new.
2. Create `<feature>.spec.ts` inside.
3. Import the fixture: `import { expect, test } from '../../_fixtures/app.js';`
4. Use `api.*` for setup (faster than UI) and `app.*` page-object methods for UI assertions.
5. Add the route to `tests/_contracts/web-routes.yaml`.

### Add a runtime adapter

1. Create `packages/runtimes/<runtime>/` with `adapter.go`, `render.go`, `spawn.go`, `tool.go`.
2. Add unit tests next to source: `render_test.go`, `spawn_test.go`, `<runtime>_test.go`.
3. For each test case in `packages/runtimes/testdata/<case>/`, generate `output_<runtime>/` (regenerate with `JTERRAZZ_TEST_UPDATE=1`).
4. Update `packages/runtimes/README.md`.
5. Add the runtime to `tests/_contracts/runtimes.yaml` with its facets, tests, docs, and golden output dir.
6. `make test-contracts` will now require all of the above to exist.

### Add an API route

1. Add `mux.HandleFunc("METHOD /api/path", cors(s.handleX))` in `apps/api/server.go#registerRoutes`.
2. Add a handler test in `apps/api/server_test.go` using `httptest.NewServer(server.Handler())`.
3. Optionally add a Playwright spec under `tests/web/<domain>/<feature>/`.
4. Add the route to `tests/_contracts/api-routes.yaml`.

### Add a CLI command

1. Add the Cobra command under `apps/cli/<noun>/<verb>.go`.
2. Add a behaviour spec under `tests/specs/cli/<domain>/<case>.spec.yaml`, and its `--help` page under `tests/specs/cli/help/`.
3. Add it to `tests/_contracts/cli-commands.yaml` (with `--help` snapshot path).
4. Generated docs under `docs/reference/spwn_<noun>_<verb>.md` are checked into the repo.

### Add a catalog entry

1. Create `catalog/<slug>/spwn.yaml` (+ optional `agents/`, `skills/`, `tools/`, `hooks/`).
2. Add it to `tests/_contracts/catalog.yaml` with declared smoke coverage.
3. The contract checker verifies the manifest parses and dependencies resolve.

### Add a runtime simulator

1. Create `tests/_simulators/<runtime>/mock.sh` that follows the vendor CLI's protocol.
2. Update `tests/_simulators/Dockerfile.test` to `COPY <runtime>/mock.sh /usr/local/bin/<runtime>`.
3. Add a contract test under `packages/runtimes/<runtime>/` that exercises the simulator over the real entry points (json mode, resume, prompt files, exit codes).

### Add a web route

1. Create the page under `apps/web/src/app/<route>/page.tsx`.
2. Add either a component test (vitest) or a Playwright spec under `tests/web/<domain>/<feature>/<feature>.spec.ts`.
3. Add the route to `tests/_contracts/web-routes.yaml`.

---

## Anti-Patterns

These will fail CI or get caught in review:

- **`time.Sleep` in Go tests, `waitForTimeout` in TS tests.** Use `WaitFor` / `expect.poll` / locator state.
- **Stubbing `global.fetch` in web tests.** Use MSW.
- **Writing to the dev's `~/.spwn`.** Always isolate with `t.TempDir()` + `SPWN_HOME` (Go) or the Playwright fixture (TS).
- **Skipping cleanup.** Use `t.Cleanup()` (Go) or `await using` (TS). Containers must be labeled with the test-run id.
- **Testing internals via reflection.** Test the contract documented in the package's `README.md` instead.
- **Hardcoded counts that break when the catalog grows** (`expect(examples).toHaveLength(5)`). Use `>=` or `toContain`.
- **Casing-sensitive assertions on user-facing text without checking what the UI actually renders.** Read the page snapshot in the failure first.
- **Adding a new runtime/route/command/catalog entry without updating `tests/_contracts/`.** `make test-contracts` will fail.
- **Bundling multiple features into one web spec file.** Split into `<domain>/<feature>/<feature>.spec.ts` so each feature owns its proof.
- **Hand-written CI commands.** CI calls Make targets. The validate.yaml workflow IS the aggregate; if you want a new gate, add a job there, not a meta-target in the Makefile.

---

## Glossary

- **Layer** — A horizontal slice of the test pyramid (L0 governance, L1 unit, …, L6 real-runtime smoke).
- **Surface** — Anything users or other systems depend on: an API route, a CLI command, a runtime adapter, a catalog entry, a web route. Every surface must declare its tests in `tests/_contracts/`.
- **Contract** — A registry entry in `tests/_contracts/*.yaml` that names a surface and lists the proofs it requires.
- **Golden** — A byte-level expected output committed to the repo. For Go, regenerate with `JTERRAZZ_TEST_UPDATE=1`; for the CLI E2E documents, with `TEST_UPDATE=1`.
- **Simulator** — An executable protocol contract for an external CLI (under `tests/_simulators/`). Not a loose mock — a contract test guards the protocol shape.
- **`cli`** — The single CLI E2E runner exported from `tests/specs/cli/cli.specification.ts`.
- **Document** — A `<case>.spec.yaml` stating one terminal session: the ground, the `runs:`, the streams and `files:`. The default form for a CLI E2E spec.
- **Chain** — A `<aspect>.test.ts` for what a document cannot state. Its docblock opens with the reason.
- **Test label** — `SPWN_TEST_LABEL` (e.g. `web-e2e-<ts>-<rand>`) attached to every container created by a test run, so cleanup never affects unrelated containers.
- **Underscore-prefixed folder** — Test infrastructure (`_contracts/`, `_simulators/`, `_fixtures/`, `_smoke/`, `_catalog/`). Sorts above feature folders and signals "this is not a feature."

---

## See Also

- [tests/README.md](README.md) — How to run tests
- [CLAUDE.md](../CLAUDE.md) — Project conventions, dependency graph, vocabulary
- [.github/workflows/validate.yaml](../.github/workflows/validate.yaml) — CI mapping
- `~/Desktop/spwn-test-architecture-plan.md` — Original design plan with open issues
