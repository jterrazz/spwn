# spwn Test Architecture Plan

Date: 2026-04-30
Repo audited: `/Users/jterrazz/Developer/spwn/spwn`

## Goal

Build a reusable, scalable, hard-to-forget test architecture for spwn.

The target state is not "more tests everywhere". The target state is:

1. Every architectural surface declares the proof it needs.
2. CI verifies those proofs automatically.
3. Local commands match CI exactly.
4. E2E tests read like executable user stories.
5. Mocks are avoided where real boundaries are cheap.
6. When replacement is necessary, test doubles behave like contract simulators, not loose mocks.
7. Network behavior is tested at the network boundary with tools like `httptest`, MSW, or real local servers.
8. Manual QA becomes rare, explicit, documented, and backed by automated regression tests when possible.

## Current Snapshot

Observed during audit:

- 178 Go `*_test.go` files.
- 61 TypeScript CLI E2E specs under `tests/cli`.
- 23 Go Docker E2E files under `packages/world/tests/e2e`.
- 31 runtime golden fixture cases under `packages/runtimes/testdata`.
- 6 Playwright web specs.
- 1 web Vitest unit test file under `apps/web/src/lib/__tests__`.
- Strong CLI test harness in `tests/setup/cli.specification.ts`.
- Good test docs in `tests/README.md`.
- Go layer architecture enforced by depguard in `.golangci.yml`.

Commands verified during audit:

```bash
make test
```

Passed.

Coverage spot checks showed weak areas:

- `packages/architect`: about 12 percent.
- `apps/api`: about 32 percent.
- `apps/cli/world`: about 14 percent.
- `packages/compile`: package root 0 percent, most useful coverage in `internal/dockerfile`.
- `packages/container/backend`: about 0.4 percent.
- `packages/project/internal/scaffold`: 0 percent.
- `apps/web`: one Vitest test file, but not wired into a package test script or CI.

## Core Diagnosis

spwn already has many useful tests. The missing piece is governance.

Today, a contributor can add a runtime, route, command, catalog tool, or web feature without a central mechanism forcing them to declare:

- which test layers apply,
- which fixture proves the happy path,
- which fixture proves the failure path,
- which CI job runs it,
- which docs must stay in sync,
- which manual QA remains accepted and why.

That is the gap this plan resolves.

## Non-Negotiable Principles

1. **Tests follow architecture.**
   Every package/layer gets a default proof type.

2. **No hidden manual gates.**
   If something is manual, it must be listed in a debt file with owner, reason, and automation path.

3. **Use real boundaries.**
   Use real parsers, real file systems in temp dirs, real HTTP servers, real Docker, real Playwright where practical.

4. **Mock external vendors, not your own system.**
   Avoid mocking spwn internals in E2E. Mock Anthropic/OpenAI/OS keychain/payment-style boundaries only when the real thing is costly, stateful, or unsafe.

5. **Prefer network-level test doubles.**
   In Go, use `httptest.Server`. In web and Node, use MSW. Do not stub `fetch` directly except for tiny pure parser tests.

6. **Every E2E reads like a spec.**
   Test helpers should expose descriptive operations like `givenProject`, `whenWorldStarts`, `thenAgentSeesSkill`, not raw process plumbing.

7. **No fixed sleeps in browser tests.**
   Replace `waitForTimeout` with locator expectations, API polling, event probes, or app readiness markers.

8. **Local equals CI.**
   Make targets are the source of truth. CI calls Make targets, not hand-written command variants.

9. **Coverage is a signal, not a goal.**
   Add thresholds only where they make architectural sense. Do not chase 100 percent.

10. **Regression surfaces get golden or contract tests.**
    Runtime render output, CLI output, generated docs, API schemas, and catalog manifests should be machine-compared.

## Target Test Pyramid

### Layer 0: Static Architecture Gates

Purpose:

- Prevent import drift.
- Prevent stale docs.
- Prevent new untested surfaces.
- Prevent accidental direct network calls in tests where a local server should be used.

Tools:

- depguard for Go imports.
- contract linter for test coverage declarations.
- generated-doc drift checks.
- lint rules or repo scans for forbidden patterns.

Command:

```bash
make test-contracts
```

Must check:

- Every runtime has required renderer/tool/spawn tests.
- Every API route is listed in API route contract tests.
- Every CLI command has at least help coverage and one behavior spec or declared exemption.
- Every catalog entry has manifest validation and, if executable, install/probe coverage.
- Every web route has either component/network tests or Playwright coverage.
- Docs that describe runtime layouts match generated facts.

### Layer 1: Pure Unit Tests

Purpose:

- Fast feedback.
- Pure logic, parser rules, path math, validation rules, small formatters.

Rules:

- No Docker.
- No live network.
- No real home directory.
- Use `t.TempDir()` and `t.Setenv("SPWN_HOME", ...)`.
- Table-driven tests for rule sets.

Command:

```bash
make test
```

### Layer 2: Contract and Golden Tests

Purpose:

- Lock byte-level and schema-level contracts.
- Catch drift before E2E.

Surfaces:

- Runtime renderer outputs.
- CLI JSON output.
- CLI help output.
- API response schemas.
- Hook translation.
- Dockerfile generation.
- Catalog manifest expansion.

Commands:

```bash
make test-golden
make test-api-contracts
```

### Layer 3: Local Integration Tests

Purpose:

- Real subsystems, local fake vendors.
- No Docker unless required.

Examples:

- API handler with `httptest.Server`.
- Auth OAuth flows with local `httptest.Server`.
- Web network behavior with MSW.
- Gate proxy with local upstream server.

Commands:

```bash
make test-integration
```

### Layer 4: Docker E2E

Purpose:

- Verify container lifecycle, filesystem, labels, runtime launch commands, mount behavior, docker-cp behavior, tool probes.

Rules:

- Use real Docker.
- Use `spwn-test:latest` for PR E2E.
- Test simulators for runtime binaries are allowed only as protocol simulators.
- Always use per-test labels and cleanup.

Commands:

```bash
make test-e2e
make test-e2e-compile
```

### Layer 5: CLI E2E

Purpose:

- Verify the compiled `bin/spwn` from a user's perspective.
- Assert stdout/stderr/files/containers with descriptive helpers.

Rules:

- Use the existing `tests/setup/cli.specification.ts` harness.
- Prefer JSON assertions for machine contracts.
- Use fixture files for exact output.
- Use `await using` whenever containers may spawn.

Command:

```bash
make test-ts
```

### Layer 6: Web E2E

Purpose:

- Verify UI workflows against local API and real browser.

Rules:

- Use isolated `SPWN_HOME`.
- Use `SPWN_TEST_LABEL`.
- No `waitForTimeout`.
- Use page objects with user-language methods.
- Only use real Docker for workflows whose value is Docker.
- Use MSW or a local API fixture mode for pure UI route tests.

Command:

```bash
make test-web
```

### Layer 7: Real Runtime Smoke

Purpose:

- Validate that Claude/OpenAI/Codex integration still works against real vendor CLIs and auth.

Rules:

- Never required on normal PRs.
- Scheduled or manual.
- Budget and timeout capped.
- Must print exact tested model/runtime/provider.
- Must write artifacts with session IDs and prompt snippets.

Command:

```bash
make test-real-runtime
```

## Issue Register and Resolution Plan

### Issue 1: CI Does Not Run the Whole Declared Pyramid

Problem:

- PR CI runs `make test`, `make test-ts`, `make build`, `make lint`, `make web-build`.
- `make test-e2e`, `make test-e2e-compile`, and `make test-web` are not PR-blocking.
- Smoke runs only on push to main.

Fix:

1. Add Make targets:

```makefile
test-pr: lint test test-contracts test-ts web-build
test-docker-pr: test-e2e test-e2e-compile
test-release: test-pr test-docker-pr test-smoke test-web
test-nightly: test-release test-real-runtime
```

2. Update `.github/workflows/validate.yaml`:

- `lint`: unchanged.
- `unit`: `make test`.
- `contracts`: `make test-contracts`.
- `cli-e2e`: `make test-ts`.
- `docker-e2e`: run `make test-e2e` and `make test-e2e-compile`.
- `web-build`: `make web-build`.
- `web-e2e`: initially path-aware or nightly, then PR-blocking once isolated and stable.
- `smoke`: push to main and nightly.

3. Add concurrency limits for Docker jobs:

```yaml
concurrency:
  group: docker-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: false
```

Acceptance:

- A PR cannot merge if a Go package, CLI command, runtime renderer, or API contract breaks.
- Docker E2E is at least required for PRs touching `packages/architect`, `packages/world`, `packages/compile`, `packages/runtimes`, `apps/cli/world`, `apps/cli/agent`, `tests/fixtures`, or `catalog`.

### Issue 2: No Central Test Contract Registry

Problem:

- New surfaces can be added without test declarations.
- Existing tests are spread across Go, Vitest, Playwright, and docs.

Fix:

Add:

```text
tests/contracts/
  features.yaml
  api-routes.yaml
  cli-commands.yaml
  runtimes.yaml
  catalog.yaml
  web-routes.yaml
  assert-contracts.ts
```

Example `runtimes.yaml`:

```yaml
runtimes:
  claude-code:
    facets: [tool, render, spawn]
    requires:
      - go-unit: packages/runtimes/claudecode
      - golden: packages/runtimes/testdata/*/output_claude_code
      - docker-e2e: tests/cli/agent/talk
      - docs: packages/runtimes/README.md
  codex:
    facets: [tool, render, spawn]
    requires:
      - go-unit: packages/runtimes/codex
      - golden: packages/runtimes/testdata/*/output_codex
      - cli-tree-only: tests/cli/build/tree-only
      - docker-e2e: tests/cli/agent/talk or dedicated codex spec
      - docs: packages/runtimes/README.md
```

Example `api-routes.yaml`:

```yaml
routes:
  GET /api/health:
    tests:
      - apps/api/server_test.go
      - tests/web/specs/api-health.spec.ts
  POST /api/worlds:
    tests:
      - apps/api/create_world_body_test.go
      - tests/web/specs/world-lifecycle.spec.ts
```

Acceptance:

- `make test-contracts` fails when a route, CLI command, runtime, or catalog entry is missing declared tests.
- Exemptions require a reason and expiration.

### Issue 3: API Tests Can Drift From Real Route Registration

Problem:

- `apps/api/server_test.go` manually recreates routes.
- `Server.Start()` is the real route source.
- Tests can pass while production routes are missing or wired differently.

Fix:

1. Add method:

```go
func (s *Server) Handler() http.Handler {
    mux := http.NewServeMux()
    s.registerRoutes(mux)
    return mux
}
```

2. Refactor `Start()`:

```go
s.srv = &http.Server{Addr: s.addr, Handler: s.Handler()}
```

3. Refactor tests to use `srv.Handler()`.

4. Add route inventory test:

- Either a declarative route table used by registration.
- Or a test that sends representative requests for every route in `tests/contracts/api-routes.yaml`.

Acceptance:

- No test manually recreates the API router.
- Adding a route without contract coverage fails.

### Issue 4: Web Tests Use Real User State

Problem:

- `tests/web/playwright.config.ts` says tests use real `~/.spwn`.
- That can corrupt local state and creates non-reproducible tests.

Fix:

1. In Playwright global setup:

- Create temp `SPWN_HOME`.
- Write a test config JSON for teardown.
- Export `SPWN_HOME`, `SPWN_TEST_LABEL`, `SPWN_SKIP_AUTH_VALIDATION=1`.

2. Pass env to both web servers:

- `spwn dash start`
- `next dev`

3. Cleanup by label, not broad `spwn.kind=world` only.

4. Make fixture API expose current test IDs for debugging.

Acceptance:

- `make test-web` never reads or writes real `~/.spwn`.
- Teardown only destroys containers labeled with that test run.

### Issue 5: Web Tests Use Fixed Sleeps

Problem:

- Many `page.waitForTimeout(...)` calls in `tests/web/specs` and `tests/web/fixtures/app.ts`.
- These are slow and flaky.

Fix:

1. Add page object methods that wait on actual UI state:

```ts
await worldsPage.expectLoaded();
await worldsPage.expectWorldVisible("matrix");
await worldPage.expectAgentVisible("Neo");
await app.expectToast("World destroyed");
```

2. Replace sleep after API calls with `expect.poll`:

```ts
await expect.poll(async () => {
  const worlds = await api.worlds();
  return worlds.some((w) => w.id === worldId);
}).toBe(true);
```

3. Add API readiness endpoint if needed:

- `/api/health`
- `/api/status`
- `/api/test/readiness` only in test mode if absolutely needed.

Acceptance:

- `rg "waitForTimeout" tests/web` returns zero or only documented unavoidable cases.
- Web tests are deterministic on CI.

### Issue 6: Web Unit Tests Are Not Properly Wired

Problem:

- `apps/web/src/lib/__tests__/stream-chat.test.ts` exists.
- `apps/web/package.json` has no `test` script.
- Root tests config excludes `web/**`.

Fix:

1. Add `apps/web/vitest.config.ts`.
2. Add scripts:

```json
{
  "test": "vitest --run",
  "test:watch": "vitest"
}
```

3. Add `make test-web-unit`.
4. Add CI job or include in `make test-pr`.

Acceptance:

- `pnpm -C apps/web test` runs and passes.
- `make test-pr` includes web unit tests.

### Issue 7: Web Network Tests Stub `fetch` Directly

Problem:

- `stream-chat.test.ts` stubs global `fetch`.
- This is okay for tiny parser tests but weak for client behavior.
- You prefer network-level mocking like MSW.

Fix:

1. Add MSW to `apps/web`.
2. Create:

```text
apps/web/src/test/msw/server.ts
apps/web/src/test/msw/handlers.ts
apps/web/src/test/setup.ts
```

3. Convert network behavior tests to MSW:

- SSE stream.
- JSON fallback.
- HTTP errors.
- Network errors.
- Fallback URL.

4. Keep pure stream parser tests with direct stream helpers only if parser is extracted.

Acceptance:

- Client tests exercise actual `fetch` calls through MSW.
- Direct `vi.stubGlobal("fetch")` usage is limited and justified.

### Issue 8: Gate Browser and SDK Have No Test Runner

Problem:

- `apps/gate/browser/package.json` and `apps/gate/sdk/package.json` are not workspace packages.
- They have no tests.
- `docs/gate-test-debt.md` lists manual/deferred coverage.

Fix:

1. Add them to `pnpm-workspace.yaml`, or create a root `apps/gate/package.json` that owns Node tests.
2. Use `node:test` or Vitest. Prefer Vitest for repo consistency.
3. Add tests:

SDK:

- tool manifest generation.
- CLI dispatch.
- MCP method registration.
- browser session client request shape.
- error propagation.

Browser sidecar:

- `/healthz`.
- session create/close.
- goto/click/type/eval request validation.
- captured responses.
- timeout errors.
- concurrent sessions with small load.

4. For browser-side tests:

- Unit tests can use fake Playwright objects.
- Integration tests can use real Playwright with local static server.

Acceptance:

- `make test-gate-node` exists.
- Gate debt file shrinks.
- Node sidecar breakage fails CI.

### Issue 9: Runtime Simulators Are Under-Specified

Problem:

- `mock-claude` and `mock-codex` are useful, but the word "mock" hides that they are protocol contracts.
- The Codex simulator previously had resume syntax drift.

Fix:

1. Rename conceptually in docs to "runtime simulators".
2. Add contract tests for simulator behavior:

- command parsing.
- session IDs.
- resume syntax.
- JSON output shape.
- prompt file discovery.
- working directory.
- exit code behavior.

3. Add a simulator spec:

```text
tests/fixtures/runtime-simulators/
  claude-contract.md
  codex-contract.md
```

4. Keep scripts named as-is if renaming causes churn, but document contract.

Acceptance:

- Simulator drift is caught before E2E.
- Codex resume syntax is verified against the current expected CLI shape.

### Issue 10: Runtime Docs Drift

Problem:

- `packages/runtimes/README.md` was stale about Codex renderer support.
- `packages/runtimes/claudecode/RENDER.md` also appears to mention old skill paths.

Fix:

1. Update docs immediately.
2. Add docs contract tests:

- Parse registered runtime adapters.
- Assert README facet table matches code.
- Assert render paths documented match golden output paths.

3. Add generated snippet:

```bash
go run ./scripts/gen-runtime-docs
```

or test-only check if generation is too heavy.

Acceptance:

- Adding a runtime facet without docs fails `make test-contracts`.

### Issue 11: `packages/project/internal/scaffold` Has No Direct Tests

Problem:

- Scaffold bugs affect every new user.
- Current coverage mostly comes from CLI init and runtime build tests.

Fix:

1. Add direct scaffold unit tests:

- blank project files.
- matrix/severance/startup examples.
- backend selection.
- runtime neutral `AGENTS.md`.
- no stale `CLAUDE.md` references in provider-neutral source.
- local skill/hook/tool scaffolding.

2. Add golden tests for scaffold output tree.

Acceptance:

- `packages/project/internal/scaffold` has direct tests.
- Scaffold output changes are visible as golden diffs.

### Issue 12: `packages/architect` Has Low Unit Coverage

Problem:

- Architect is high-risk orchestration.
- It currently has many tests but low statement coverage because the real spawn path is mostly covered by E2E, not unit.

Fix:

Do not chase high coverage blindly. Add focused tests for invariants:

- runtime prompt file selection.
- local skills included in runtime faculties.
- runtime routing for hot deploy/talk/NPC.
- spawn cleanup on each failure stage.
- credential sync failure cleanup.
- materialisation failure cleanup.
- destroy sync-back failure behavior.
- image cache decision matrix.
- workspaces mount matrix.

Acceptance:

- Architect unit tests cover failure cleanup paths.
- E2E covers happy lifecycle paths.

### Issue 13: Docker Backend Coverage Is Near Zero

Problem:

- `packages/container/backend` is mostly a thin Docker wrapper.
- Coverage is low because integration with Docker is not unit-tested.

Fix:

1. Keep minimal unit coverage for command construction if any.
2. Add Docker integration tests behind a build tag:

```go
//go:build docker
```

3. Cover:

- create/start/exec/remove.
- labels.
- copy file in/out.
- image exists/list.
- timeout/cancel behavior.

Acceptance:

- Backend has explicit Docker integration coverage, not fake unit tests.

### Issue 14: Compile Root Package Has No Direct Tests

Problem:

- Image builder behavior is critical.
- Dockerfile generator and compile E2E exist, but package root has no direct contract tests.

Fix:

Add tests for:

- build request validation.
- build cache label selection.
- version hash changes when Dockerfile/context/policy/runtime changes.
- `BuildFromBase` tarball layout.
- expected behavior when base image is missing.

Acceptance:

- Cache invalidation has focused tests.
- Policy-only changes are proven to invalidate image cache.

### Issue 15: API Tests Use Some Time Sleeps

Problem:

- `apps/api/server_test.go` has fixed sleeps.

Fix:

- Replace with readiness polling.
- Use local server started with listener and explicit shutdown.
- Add `waitForHTTP` helper.

Acceptance:

- No fixed sleeps in API tests except justified clock behavior.

### Issue 16: Manual QA Debt Is Spread and Not Enforced

Problem:

- `docs/gate-test-debt.md` is useful but not enforced.
- Manual QA items can live forever.

Fix:

1. Move to a structured file:

```yaml
manual_qa:
  gate-sidecar-restart:
    reason: needs real Playwright process crash
    owner: gate
    last_verified: 2026-04-25
    automate_by: 2026-05-31
    replacement_test: packages/gate/sidecar_integration_test.go
```

2. Add contract check:

- Missing `last_verified` fails.
- Expired `automate_by` fails or warns.

Acceptance:

- Manual QA debt is visible, bounded, and reviewed.

### Issue 17: Real Runtime Coverage Is Ad Hoc

Problem:

- OpenAI/Codex real QA was done manually.
- It should be repeatable without being PR-blocking.

Fix:

Add `tests/real-runtime/`:

```text
tests/real-runtime/
  codex-spwn.e2e.test.ts
  claude-spwn.e2e.test.ts
  prompts/
    filesystem-check.md
    skills-check.md
```

Command:

```bash
SPWN_REAL_RUNTIME=1 pnpm -C tests exec vitest run --config vitest.real-runtime.config.ts
```

Tests:

- `spwn auth` sees provider.
- `spwn init --backend codex`.
- `spwn up --backend codex`.
- `spwn agent talk neo` asks:
  - sees `AGENTS.md`,
  - sees runtime config,
  - sees skills,
  - sees expected Faculties.
- second `talk` verifies session continuity.
- cleanup.

Acceptance:

- Real runtime checks are one command.
- They are skipped unless env explicitly enables them.
- They have hard timeout and cleanup.

### Issue 18: CLI Command Coverage Is Not Enforced Per Command

Problem:

- Many CLI areas are covered, but there is no command manifest enforcing help and behavior tests.

Fix:

1. Generate command tree:

```bash
bin/spwn --help
bin/spwn <cmd> --help
```

2. Add `tests/contracts/cli-commands.yaml`.
3. Contract linter checks:

- every command has help snapshot,
- every command has at least one behavior test or exemption,
- every command with `--json` has JSON fixture test.

Acceptance:

- Adding a Cobra command without tests fails.

### Issue 19: CLI Output Assertions Are Mixed

Problem:

- Some specs use exact fixtures.
- Some use substring matching.

Fix:

Define assertion levels:

- `toMatchFixture`: for stable user-facing outputs.
- `json.toMatch`: for machine outputs.
- `toContain`: only for logs/progress/noisy streams.
- Regex: allowed for IDs, timestamps, dynamic values.

Add lint helper or review rule:

- Prefer exact fixture for errors and help.
- Prefer JSON for state.

Acceptance:

- CONTRIBUTING and tests README define output assertion policy.

### Issue 20: Catalog Tool E2E Coverage Is Uneven

Problem:

- Catalog invariant tests exist.
- Real install/probe coverage depends on smoke and compile E2E.
- Gate tools have known manual debt.

Fix:

1. Add `tests/contracts/catalog.yaml`.
2. For each catalog entry:

- manifest parses,
- dependencies resolve,
- Dockerfile installs,
- verify command exists,
- if gate tool, MCP manifest is valid,
- if skill provider, skill files are rendered into runtime output.

3. Add smoke matrix:

- cheap catalog validation on PR.
- full image build on main/nightly.

Acceptance:

- New catalog entry fails without tests or explicit no-op type.

## Implementation Phases

### Phase 0: Stabilize Current Ground

Tasks:

1. Update stale runtime docs:
   - `packages/runtimes/README.md`
   - `packages/runtimes/claudecode/RENDER.md`
   - any docs still saying Codex has no renderer.

2. Add missing tests for the Codex fixes already made:
   - scaffold no Claude references,
   - runtime Faculties includes `skill/focus`,
   - spawn progress says `AGENTS.md`.

3. Run:

```bash
make test
make build
pnpm -C tests exec vitest run cli/build/tree-only/tree-only.e2e.test.ts cli/lifecycle/backend-flag/backend-flag.e2e.test.ts
```

Exit criteria:

- Current branch is green.
- Docs no longer lie about runtime support.

### Phase 1: Create Test Contract Governance

Tasks:

1. Add `tests/contracts`.
2. Add initial YAML registries:
   - runtimes,
   - CLI commands,
   - API routes,
   - web routes,
   - catalog entries.
3. Add `assert-contracts.ts`.
4. Add `make test-contracts`.
5. Wire into `make test-pr`.

Exit criteria:

- Adding a fake runtime/route/command without tests fails.
- Existing repo passes after all current surfaces are registered.

### Phase 2: Fix API Test Architecture

Tasks:

1. Extract router registration into one production method.
2. Refactor tests to use production handler.
3. Add API route contract tests.
4. Add JSON response fixture helpers.
5. Replace API sleeps with readiness polling.

Exit criteria:

- `apps/api` route tests cannot drift from production.
- API contract registry is complete.

### Phase 3: Fix Web Unit and Network Tests

Tasks:

1. Add `apps/web/vitest.config.ts`.
2. Add `pnpm -C apps/web test`.
3. Add MSW.
4. Convert network-heavy tests to MSW.
5. Extract pure stream parsing if needed.
6. Add `make test-web-unit`.

Exit criteria:

- Web unit tests run in CI.
- Network client behavior is tested at HTTP boundary.

### Phase 4: Fix Playwright Web E2E

Tasks:

1. Isolate `SPWN_HOME`.
2. Use `SPWN_TEST_LABEL`.
3. Tighten cleanup to test label.
4. Replace all `waitForTimeout`.
5. Split tests into:
   - UI-only with MSW/local fixture API,
   - full-stack with real API and Docker.
6. Add page objects:
   - `WorldsPage`,
   - `AgentsPage`,
   - `WorldDetailPage`,
   - `CommandPalette`.

Exit criteria:

- `make test-web` is deterministic.
- No writes to real `~/.spwn`.
- No fixed sleeps.

### Phase 5: Gate Node Tests

Tasks:

1. Wire `apps/gate/browser` and `apps/gate/sdk` into workspace test commands.
2. Add SDK tests.
3. Add browser sidecar tests.
4. Add gate integration test with local static HTTP server.
5. Convert relevant `docs/gate-test-debt.md` items to automated tests.

Exit criteria:

- `make test-gate-node` exists and runs in CI.
- Gate manual debt is reduced and structured.

### Phase 6: Runtime Simulator Contracts

Tasks:

1. Document simulator contracts for Claude and Codex.
2. Add tests that execute scripts directly.
3. Verify JSON mode shape.
4. Verify session and resume behavior.
5. Verify prompt file detection.
6. Update `spwn-test:latest` build if scripts change.

Exit criteria:

- Simulator drift fails before CLI E2E.

### Phase 7: Docker and Compile Coverage

Tasks:

1. Add Docker backend integration tests behind `docker` tag.
2. Add compile root tests for cache/hash/request behavior.
3. Add policy cache invalidation test.
4. Add build-from-base tree layout tests.

Exit criteria:

- Docker backend has explicit integration coverage.
- Compile cache behavior has direct tests.

### Phase 8: CI Matrix Upgrade

Tasks:

1. Add `make test-pr`.
2. Add `make test-docker-pr`.
3. Add `make test-release`.
4. Add `make test-nightly`.
5. Update `.github/workflows/validate.yaml`.
6. Add scheduled nightly workflow.
7. Add path filters if runtime is too high.

Exit criteria:

- CI expresses the same pyramid as docs.
- Local and CI commands match.

### Phase 9: Real Runtime Smoke

Tasks:

1. Add opt-in real runtime Vitest config.
2. Add Codex real runtime test.
3. Add Claude real runtime test if credentials can be safely provided.
4. Add budget/timeouts.
5. Store artifacts.
6. Make cleanup unconditional.

Exit criteria:

- One command repeats the manual Codex QA.
- Scheduled/manual only.

### Phase 10: Final Audit and Hardening

Tasks:

1. Run all test tiers.
2. Generate coverage reports.
3. Generate missing-test report.
4. Run docs drift checks.
5. Run stale manual QA report.
6. Verify cleanup leaves no `world-*` containers.
7. Verify no test touches real `~/.spwn`.
8. Verify no fixed sleeps in web/API tests.

Exit criteria:

- Every known issue is closed or has a tracked exemption.
- `make test-release` passes locally.
- CI passes on a clean branch.

## Proposed Makefile Targets

```makefile
test-pr: lint test test-contracts test-web-unit test-ts web-build

test-docker-pr: build-test-image test-e2e test-e2e-compile

test-release: test-pr test-docker-pr test-smoke test-web

test-nightly: test-release test-real-runtime

test-contracts:
	pnpm -C tests exec tsx contracts/assert-contracts.ts

test-web-unit:
	pnpm -C apps/web test

test-gate-node:
	pnpm -C apps/gate test

test-real-runtime:
	SPWN_REAL_RUNTIME=1 pnpm -C tests exec vitest run --config vitest.real-runtime.config.ts
```

If `tsx` is not already in dependencies, either add it to `tests` or implement contract checks in Go.

## CI Target State

### Pull Request Required

- lint
- Go unit tests
- test contracts
- web unit tests
- CLI E2E with mock runtime image
- web build
- Docker E2E path-aware for touched Docker/runtime/world code

### Push to Main

- all PR checks
- real build smoke
- full Docker E2E
- web E2E

### Nightly

- all main checks
- real runtime smoke
- longer gate/browser sidecar integration
- catalog full matrix

## Test Design Language

Use a screaming spec style for E2E.

Bad:

```ts
await spec("test").exec(["up", "agent talk neo hi"]).run();
expect(result.exitCode).toBe(0);
```

Good:

```ts
await using run = await spwnSpec("agent can talk inside a live codex world")
  .givenProject("docker-pilot")
  .givenRuntime("codex")
  .givenBaseImage("spwn-test:latest")
  .whenWorldStarts("neo")
  .whenAgentReceives("neo", "list runtime files")
  .thenAgentRuntimeWasInvoked("neo")
  .thenAgentSeesPromptFile("neo", "AGENTS.md")
  .thenAgentSeesSkill("neo", "focus")
  .run();
```

This does not require replacing the current runner. Add a thin domain wrapper around it.

## Descriptive Helper Backlog

CLI helpers:

- `givenProject(name)`
- `givenEmptyHome()`
- `givenBaseImage(image)`
- `givenRuntime(runtime)`
- `whenCommand(args)`
- `whenWorldStarts(world)`
- `whenWorldStops(world)`
- `whenAgentTalks(agent, message)`
- `thenExitCode(code)`
- `thenStdoutMatchesFixture(name)`
- `thenJsonMatchesFixture(name)`
- `thenContainerExists(world)`
- `thenContainerFileExists(world, path)`
- `thenContainerFileContains(world, path, text)`
- `thenNoWorldContainersRemain()`

Web helpers:

- `app.gotoHome()`
- `app.expectShellReady()`
- `worlds.expectLoaded()`
- `worlds.expectWorldVisible(name)`
- `worlds.openWorld(name)`
- `world.expectAgentVisible(name)`
- `world.expectStatus(status)`
- `world.destroy()`
- `toast.expectMessage(text)`

API helpers:

- `api.expectRoute(method, path).returns(status)`
- `api.createAgent(name)`
- `api.createWorld(config)`
- `api.destroyWorld(id)`
- `api.pollWorld(id).untilStatus(status)`

## Required Regression Specs

Add or confirm these exist:

### Runtime Rendering

- Claude emits `CLAUDE.md`, `.claude/settings.json`, `.claude/skills`.
- Codex emits `AGENTS.md`, `.codex/config.toml`, `.codex/hooks.json`, `.agents/skills`.
- No provider-neutral scaffold mentions provider-specific prompt files.
- Local `skill/...` appears in Faculties for tree-only and real spawn.
- Runtime model pins render correctly.
- Hooks render correctly for Claude and Codex.

### Spawn

- Spawn writes correct prompt filename in progress.
- Spawn cleans up container when probe fails.
- Spawn cleans up container when materialisation fails.
- Spawn syncs provider credentials for selected runtime only.
- Hot deploy uses persisted world runtime.
- Second talk resumes same session.

### Build

- `spwn build --tree-only --runtime codex` emits expected files.
- `spwn build --tree-only --runtime claude-code` emits expected files.
- `spwn build` copies tree into derived image.
- Policy-only change invalidates image cache.
- Base-image override is respected.

### CLI

- Every command has help output.
- Every JSON command has fixture coverage.
- Error messages do not leak raw Go noise.
- Commands work from project subdirectories.
- Commands with state use isolated `SPWN_HOME`.

### API

- Every registered route has at least status/error contract.
- Read-only mode returns 503 for write operations needing architect.
- CORS preflight works.
- Request validation rejects malformed JSON.
- Path parameters with encoded names work.

### Web

- App boots with API healthy.
- Navigation works.
- Worlds list loads.
- Agent detail loads.
- World lifecycle works.
- Errors render recoverably.
- Network failures show actionable UI.

### Gate

- `/healthz` works.
- Browser MCP tools registered.
- Browser session lifecycle works.
- Proxy strips route prefixes correctly.
- Tool subprocess env vars are passed.
- Tool subprocess restart behavior is tested.

## Double-Check Loop

Run this loop after each phase.

### Step A: Fast Local Proof

```bash
make test
make test-contracts
make build
```

If it fails:

1. Fix the first failure.
2. Re-run the exact failing command.
3. Re-run Step A from the top.

### Step B: Targeted E2E Proof

Run only the suite for the touched area:

```bash
pnpm -C tests exec vitest run <specific spec>
go test -v -tags=e2e ./tests/e2e/<specific package>
```

If it fails:

1. Capture stdout/stderr.
2. Inspect containers by test label.
3. Fix.
4. Re-run the failed spec.
5. Re-run Step A.

### Step C: Full Docker Proof

```bash
make build-test-image
make test-e2e
make test-e2e-compile
make test-ts
```

If it fails:

1. Fix.
2. Re-run failing suite.
3. Re-run Step A.
4. Re-run Step C.

### Step D: Web Proof

```bash
make test-web-unit
make test-web
```

If it fails:

1. Open Playwright trace.
2. Replace any sleep-related fragility with state-based waits.
3. Re-run failing spec.
4. Re-run Step D.

### Step E: Release Proof

```bash
make test-release
```

If it fails:

1. Fix.
2. Re-run failed target.
3. Re-run `make test-release`.

### Step F: Cleanup Proof

```bash
docker ps --filter label=spwn.kind=world --format '{{.Names}} {{.Status}}'
git status --short
```

Expected:

- No unexpected `world-*` containers.
- Only intentional source changes.
- No generated junk committed.

## Done Criteria for the Whole Plan

This plan is complete only when:

1. `make test-pr` passes locally.
2. `make test-release` passes locally or documented machine limits explain any skipped suite.
3. CI uses the same Make targets.
4. Test contracts fail on missing coverage.
5. Web tests do not touch real `~/.spwn`.
6. `rg "waitForTimeout" tests/web` returns zero or justified exceptions.
7. API tests use production route registration.
8. `apps/web` unit tests run in CI.
9. Gate Node sidecar/SDK tests run in CI.
10. Runtime docs match code.
11. Manual QA debt is structured and bounded.
12. Real runtime smoke is one opt-in command.
13. New runtime/route/command/catalog/web additions have an obvious test checklist.

## Suggested PR Breakdown

PR 1:

- Fix runtime docs drift.
- Add scaffold tests for provider-neutral AGENTS.md.
- Add missing Codex regression tests.

PR 2:

- Add test contracts directory.
- Add `make test-contracts`.
- Register current runtimes and CLI commands.

PR 3:

- Refactor API router.
- Add API route contracts.

PR 4:

- Wire web Vitest.
- Add MSW.
- Convert `stream-chat` tests.

PR 5:

- Isolate Playwright.
- Remove fixed sleeps.

PR 6:

- Add Gate Node tests.
- Reduce gate debt file.

PR 7:

- Add simulator contract tests.
- Fix any simulator drift.

PR 8:

- Add Docker backend and compile root integration coverage.

PR 9:

- Upgrade CI matrix and Make targets.

PR 10:

- Add real runtime smoke tests.
- Add nightly workflow.

## Final Challenge

Do not let this become a giant refactor branch.

The scalable architecture is the contract layer plus the harness rules. Land that early, then every later PR becomes simpler because the repo starts telling you what proof is missing.

