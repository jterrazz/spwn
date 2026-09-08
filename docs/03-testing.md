# Testing

spwn is built **spec-first**: the test suite is the living specification of what the system should do. Each behavioral test describes a user-visible behavior; if a spec fails, the implementation is wrong, not the spec.

```
1. Specify   — define behavior in the knowledge (what the system SHOULD do)
2. Encode    — write tests that encode those specs (they fail initially)
3. Implement — write code that makes the tests pass
4. Verify    — the test suite IS the living specification
```

## The layer pyramid

| Layer | Location | Speed | Infra |
| ----- | -------- | ----- | ----- |
| **Unit** | `*_test.go` next to source files | ~1s | none |
| **E2E (Go)** | `packages/world/tests/e2e/`, `packages/compile/e2e/` | ~30s | Docker |
| **E2E (TS)** | `tests/specs/cli/`, `tests/_smoke/` | ~2–5min | built binary |
| **Web E2E** | `tests/web/` | varies | Playwright + real Next.js + Go API |

Each domain tests only its own contract. Cross-domain flows (spawn world + agent → verify journal) are the CLI's responsibility, exercised by the TypeScript E2E suite against the compiled `.artifacts/go/spwn`. The TS E2E suite uses [`@jterrazz/test`](https://github.com/jterrazz/package-test); runtime simulators in `tests/_simulators/` (mock Claude/Codex CLIs, baked into `spwn-test:latest`) stand in for the real runtimes.

### Two forms, one runner

Most CLI E2E specs are **documents**: a `<case>.spec.yaml` beside its siblings, stating one terminal session — the fixture it stands on, each command, each exit code, the streams byte-for-byte, and what the run left on disk. A document is the default because a spwn command IS a terminal session, and the format asserts every run of one instead of stopping at the first failure.

The rest are **chains**, `<aspect>.test.ts`, for what the format cannot state: a container read back with `.container(name)`, JSON judged by shape, an ABSENCE (`files:` says what a file contains, never what it does not), a count, two runs compared to each other, a host shell-out, output that varies with the operator's machine, a long-running process. Every chain file opens with the reason it is one; when only a single assertion needs code, the session still lives in a document and `cli.run('<case>.spec.yaml')` asserts it whole.

Both forms bind to the same runner, `tests/specs/cli/cli.specification.ts`. The full grammar and the `TEST_UPDATE=1` workflow are in [`../tests/README.md`](../tests/README.md#typescript-e2e-setup-testsspecscli).

## Running the suites

All gates run through the `Makefile` (single entry point; CI mirrors it in `.github/workflows/validate.yaml`):

```bash
make lint                # go vet across go.work + pnpm -r lint (oxlint + oxfmt + knip)
make test                # Go unit tests across the workspace (~5s)
make test-pkg PKG=agent  # verbose go test for one package
make test-contracts      # static governance: every surface declared its tests
make test-web-unit       # apps/web vitest (MSW-mocked network)
make test-gate-node      # apps/gate vitest (sidecar + SDK)

# Docker required:
make test-image          # build spwn-test:latest (runtime simulators)
make test-go-e2e         # Go world E2E (Architect/world/container)
make test-compile-e2e    # Go image-build E2E (compile + Dockerfile rendering)
make test-cli            # TypeScript CLI E2E against the compiled binary
make test-smoke          # real-build smoke: spwn init → up → tool probe
make test-web            # Playwright web E2E (real Next.js + Go API + Chromium)
```

Run `make` with no arguments for the full annotated target list.

## What a test may assume

Ten rules govern every layer of the pyramid. They are what the suite is built to hold, not aspirations, and a review that lets one go is the review that lets the pyramid rot from the bottom.

1. **Tests follow architecture.** Every package and every layer has a default proof type; a new surface inherits it rather than negotiating one.
2. **No hidden manual gates.** Anything that stays manual is listed with an owner, a reason, and the path to automating it.
3. **Use real boundaries.** Real parsers, real filesystems in temp dirs, real HTTP servers, real Docker, real Playwright, wherever that is practical.
4. **Mock external vendors, not your own system.** Anthropic, OpenAI, the OS keychain — never a spwn package standing in for another spwn package.
5. **Prefer network-level doubles.** `httptest.Server` in Go, MSW in Node and the web; stubbing `fetch` directly is for a pure parser test and nothing else.
6. **Every E2E reads like a spec.** Helpers expose `givenProject`, `whenWorldStarts`, `thenAgentSeesSkill` — never raw process plumbing.
7. **No fixed sleeps in browser tests.** `waitForTimeout` is replaced by a locator expectation, an API poll, an event probe, or a readiness marker.
8. **Local equals CI.** Make targets are the source of truth; CI calls them and never a hand-written variant.
9. **Coverage is a signal, not a goal.** A threshold is added where it makes architectural sense, and nowhere else.
10. **Regression surfaces get golden or contract tests.** Runtime render output, CLI output, generated docs, API schemas and catalog manifests are machine-compared.

Rule 1 is the one with a gate behind it. `make test-contracts` reads the registry in `tests/_contracts/` and refuses a surface that declared no proof: every runtime needs its renderer/tool/spawn tests, every API route its route contract, every CLI command at least help coverage plus one behaviour spec or a declared exemption, every catalog entry its manifest validation, every web route its component or Playwright cover. Without it a contributor can add a command, a route or a tool and nothing anywhere notices that it is unproven.

## Deeper reference

The test suite has its own detailed reference, co-located with the tests:

- [`../tests/ARCHITECTURE.md`](../tests/ARCHITECTURE.md) — the full layer breakdown, the `spec` harness cookbook, contracts/governance, simulators, and fixtures.
- [`../tests/README.md`](../tests/README.md) — how to run each layer and its conventions.

## Related

- [Architecture](01-architecture.md) — the layers the pyramid covers.
- [Developing](02-developing.md) — the loop these gates run in, and what a change owes.
