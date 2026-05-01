# Test Architecture

This document explains **how** spwn is tested — the layers, where files live, the patterns each layer uses, the governance that prevents drift, and the cookbook for adding new tests.

For **how to run** tests, see [README.md](README.md). For the original design rationale and open issues, see the plan at `~/Desktop/spwn-test-architecture-plan.md`.

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
│     L4b TS: tests/cli/<noun>/<verb>/*.e2e.test.ts (compiled bin) │
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
│   ├── README.md                  ← How to run
│   ├── ARCHITECTURE.md            ← This file
│   │
│   ├── cli/                       ← L4b CLI E2E (one folder per verb)
│   │   ├── agent/
│   │   │   ├── talk/
│   │   │   │   ├── talk.e2e.test.ts
│   │   │   │   ├── seeds/         ← Files copied into the temp project
│   │   │   │   └── expected/
│   │   │   │       ├── stdout/
│   │   │   │       └── json/
│   │   │   ├── new/, fork/, dream/, sleep/
│   │   ├── world/
│   │   │   ├── up/, down/, inspect/, snap/, …
│   │   ├── build/, check/, init/, auth/, gate/, skill/, logs/, …
│   │   └── regressions/
│   │
│   ├── web/                       ← L5 Playwright (one folder per feature)
│   │   ├── playwright.config.ts
│   │   ├── _setup/
│   │   │   ├── global-setup.ts    ← make build (sidecar binary)
│   │   │   └── global-teardown.ts ← Container cleanup by label
│   │   ├── _fixtures/
│   │   │   └── app.ts             ← Playwright `test` extended with api + app helpers
│   │   ├── agents/
│   │   │   ├── list/list.spec.ts
│   │   │   └── detail/detail.spec.ts
│   │   ├── examples/
│   │   │   └── gallery/gallery.spec.ts
│   │   ├── navigation/
│   │   │   ├── sidebar/sidebar.spec.ts
│   │   │   ├── command-palette/command-palette.spec.ts
│   │   │   └── header/header.spec.ts
│   │   ├── system/
│   │   │   └── api-health/api-health.spec.ts
│   │   └── worlds/
│   │       ├── list/list.spec.ts
│   │       ├── detail/detail.spec.ts
│   │       └── lifecycle/lifecycle.spec.ts
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
│   ├── _setup/
│   │   └── cli.specification.ts   ← The single CLI E2E runner (`spec`)
│   │
│   ├── _fixtures/                 ← Project skeletons used by `spec` runner
│   │   ├── docker-pilot/
│   │   ├── codex-pilot/
│   │   ├── single-agent/
│   │   └── testdata/              ← Shared fixtures consumed by Go E2E too
│   │
│   ├── _smoke/                    ← Real-build cross-cutting smoke tests
│   │   ├── init-up/
│   │   └── upgrade/
│   │
│   ├── _catalog/                  ← Catalog-bundle goldens (Go module)
│   │
│   ├── vitest.config.ts           ← cli + docker projects
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

- **CLI tree** mirrors the CLI's noun-verb grammar: `tests/cli/agent/talk/talk.e2e.test.ts` reads like the command `spwn agent talk`. The folder holds the spec plus its `seeds/` (project skeleton copied into the temp dir) and `expected/` (stdout/JSON fixtures).
- **Web tree** mirrors web's UI grouping: `tests/web/<domain>/<feature>/<feature>.spec.ts` reads like the surface ("worlds/list", "agents/detail", "navigation/sidebar"). One feature = one folder = one spec file.
- **No false symmetry.** CLI features (`build`, `check`, `auth`, `gate`) often have no web counterpart, and vice versa. The trees don't pretend otherwise; the contract registry (`_contracts/`) is what ties them together when an underlying surface has both.
- **Underscore-prefixed infrastructure.** `_contracts/`, `_simulators/`, `_setup/`, `_fixtures/`, `_smoke/`, `_catalog/` sort above feature folders alphabetically and are visually distinct so no one mistakes them for a domain.

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

### CLI fixture tests: `tests/cli/<area>/<feature>/expected/`

The `spec` runner (see L4b) compares stdout/stderr/JSON output against fixture files under `expected/stdout/<name>.txt` and `expected/json/<name>.json`. Same regenerate flag.

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

### L4b · TypeScript: `tests/cli/<noun>/<verb>/<verb>.e2e.test.ts`

Exercise the compiled `bin/spwn` from a user's perspective via one runner: `tests/_setup/cli.specification.ts` (`spec`).

**Folder layout per feature:**

```
tests/cli/build/build/
├── build.e2e.test.ts
├── seeds/                ← Files copied into the temp project (spwn.yaml, agents/, …)
└── expected/
    ├── stdout/
    │   └── valid-build.txt
    └── json/
        └── invalid-tool.json
```

**CLI-only pattern** (no containers):

```ts
import { describe, expect, test } from 'vitest';
import { spec } from '../../../_setup/cli.specification.js';

describe('spwn check', () => {
    test('valid project prints a clean success report', async () => {
        const result = await spec('check valid').project('single-agent').exec('check').run();

        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('valid-project.txt');
    });
});
```

**Container-asserting pattern** (Docker):

```ts
test('up provisions a running world', async () => {
    await using result = await spec('up lifecycle').project('docker-pilot').exec('up').run();

    expect(result.exitCode).toBe(0);
    result.stderr.toContain('Created container');

    const neo = result.container('neo');
    expect(neo.running).toBe(true);
    expect(neo.file('/agents/neo/CLAUDE.md').exists).toBe(true);
});
```

- `await using` — the dispose hook force-removes every container tagged with this test's label so parallel runs never collide. Harmless no-op for CLI-only tests.
- `result.container('neo').file(...)` / `.exec(...)` / `.inspect.value` — same accessor API as host-side `result`. No new vocabulary.
- `JTERRAZZ_TEST_UPDATE=1` regenerates fixtures.
- The `.project('name')` source lives at `tests/_fixtures/<name>/`; the runner is configured with `root: '../_fixtures'`.

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

1. Pick a folder: `tests/cli/<noun>/<verb>/`.
2. Create `<verb>.e2e.test.ts`, plus `seeds/` (project skeleton) and `expected/` (fixtures).
3. Import `spec` from `tests/_setup/cli.specification.js`.
4. Use `await using` if any container might spawn.
5. Prefer structured assertions: `result.stdout.toMatch('file.txt')`, `result.json.toMatch('file.json')`, `result.container('neo').file(path)`.
6. Regenerate fixtures with `JTERRAZZ_TEST_UPDATE=1 pnpm -C tests exec vitest run <glob>`.
7. Add the command to `tests/_contracts/cli-commands.yaml`.

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
2. Add a behavior test under `tests/cli/<noun>/<verb>/<verb>.e2e.test.ts` with `expected/stdout/<verb>.txt`.
3. Add it to `tests/_contracts/cli-commands.yaml` (with `--help` snapshot path).
4. Generated docs under `docs/cli/spwn_<noun>_<verb>.md` are checked into the repo.

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
- **Golden** — A byte-level expected output committed to the repo. Regenerate with `JTERRAZZ_TEST_UPDATE=1`.
- **Simulator** — An executable protocol contract for an external CLI (under `tests/_simulators/`). Not a loose mock — a contract test guards the protocol shape.
- **`spec`** — The single CLI E2E runner exported from `tests/_setup/cli.specification.ts`.
- **Test label** — `SPWN_TEST_LABEL` (e.g. `web-e2e-<ts>-<rand>`) attached to every container created by a test run, so cleanup never affects unrelated containers.
- **Underscore-prefixed folder** — Test infrastructure (`_contracts/`, `_simulators/`, `_setup/`, `_fixtures/`, `_smoke/`, `_catalog/`). Sorts above feature folders and signals "this is not a feature."

---

## See Also

- [tests/README.md](README.md) — How to run tests
- [CLAUDE.md](../CLAUDE.md) — Project conventions, dependency graph, vocabulary
- [.github/workflows/validate.yaml](../.github/workflows/validate.yaml) — CI mapping
- `~/Desktop/spwn-test-architecture-plan.md` — Original design plan with open issues
