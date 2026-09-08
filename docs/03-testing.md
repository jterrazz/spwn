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

## Deeper reference

The test suite has its own detailed reference, co-located with the tests:

- [`../tests/ARCHITECTURE.md`](../tests/ARCHITECTURE.md) — the full layer breakdown, the `spec` harness cookbook, contracts/governance, simulators, and fixtures.
- [`../tests/README.md`](../tests/README.md) — how to run each layer and its conventions.
- [`03-testing.md`](03-testing.md) — original design rationale and open issues.
- [`qa/`](qa/) — manual QA passes that complement the automated suite.

## Related

- [Architecture](01-architecture.md) — the layers the pyramid covers.
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — contributor setup.
