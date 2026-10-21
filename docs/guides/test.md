# Test

What proves a change, and how to run each proof. The test suite is the specification of what spwn does: a test that fails says the code is wrong, never the test. Where every suite lives and how each is built is [Test architecture](../reference/test-architecture.md); the conventions every TypeScript test follows are `@jterrazz/test`'s, whose manual the toolchain carries into a session.

## Running the suites

The `Makefile` is the one entry point, and CI calls the same gestures ([`.github/workflows/validate.yaml`](../../.github/workflows/validate.yaml)):

```bash
make lint               # golangci-lint over go.work, then every gate of the TypeScript toolchain
make test               # builds the binary and spwn-test:latest, then every member's tests
make mutation           # the mutation verdict over the change
make check              # lint, test and mutation: everything CI runs
```

`make test` is one Turborepo graph over both toolchains: the Go unit tests of every module in `go.work`, the module tests of `apps/web` and `apps/gate/sdk`, and the two specifications under `specs/`: `cli`, the compiled binary in Docker, and `website`, the web UI in a headless Chromium.

Three suites stay outside it, because each costs minutes and a Docker daemon of its own:

```bash
make test-go-e2e        # the Go world suite, //go:build e2e
make test-compile-e2e   # the Go image-build suite
make test-smoke         # real image builds: spwn init, up, a tool probe (~10 minutes cold)
make test-pkg PKG=agent # one Go module, verbose
```

## Prerequisites

- **Docker**, for every suite that starts a world, the `cli` specification included.
- **The test image**, `spwn-test:latest`, which `make test` builds from `docker/spwn-test/`: the base image with the mock Claude and Codex runtimes in place of the real ones.
- **The binary**, `.artifacts/go/spwn`, which `make test` builds first.
- **Chromium's headless shell**, for the `website` specification: `pnpm -C specs exec playwright install chromium-headless-shell`. No run opens a window.

## Adding a test

### A Go test

A `*_test.go` beside the code it proves, named `TestSubject_Scenario`, table-driven where a rule has several cases. Every filesystem state lives in `t.TempDir()`, with `t.Setenv("SPWN_HOME", …)`: no test reads the operator's `~/.spwn`. A Docker test goes under `packages/world/tests/e2e/` behind `//go:build e2e`, spawning through `setup.NewSpawnBuilder(t)`.

A golden (the runtime renderers' output under `packages/runtimes/__tests__/__fixtures__/`, the catalog bundles under `packages/catalog/`) is rewritten with `UPDATE_GOLDEN=1 go test ./...` in its module, and read before it is committed.

### A CLI specification

A `specs/cli/<domain>/<case>.spec.yaml` document by default: the ground it stands on, the commands of one terminal session, each exit code and both streams byte for byte, and what the run left on disk.

1. State the ground (`fixture:`, `env:`) and the `runs:`, each with a `command:` and a guessed `exit:`.
2. `TEST_UPDATE=1 pnpm -C specs exec vitest --run --project cli <path>` writes the exit codes and the streams.
3. Read what it wrote, and turn into a token whatever stayed literal and varies: a temporary path, a random name, a clock.
4. Run it again without the flag; it passes twice in a row.

A chain, `<aspect>.spec.ts`, is for what a document cannot state: a container read back, a session that starts one (the chain scopes it with `await using`), JSON judged by its structure, a command run inside a world, two runs compared, a long-running process. It opens, below its imports, with that reason, written as a `// hazard: a document cannot …` comment: what a document would lose. Everything else, an exit code, both streams, a file on disk whole, is a document.

A spec that starts the architect, in `cli/` or in `website/`, runs its chain inside `withoutArchitect` (`specs/__setup__/architect.ts`): the product names that container `spwn-architect` whatever the home, and the spec files run in parallel workers, so without the lock they remove each other's container.

### A website specification

A `specs/website/<domain>/<page>.spec.ts` visits one page of the UI, drives it through the visitor's verbs, and compares the outline (the accessibility tree) whole. The run builds the UI once and serves it with `next start`, as `spwn web` does, and each file starts its own `spwn web` on a fresh project, on the one API address that build was given, so the files run one at a time.

### A module test

A `__tests__/<module>.test.ts` beside a TypeScript module of `apps/web` or `apps/gate/sdk`: the module called directly, its network stated at the HTTP boundary, never a stubbed `fetch`.

## The manual passes

Three scenario catalogs a person walks against a live runtime sit outside every gate: [Run a manual pass](run-a-manual-pass.md). What a pass finds lands as a fix and the automated guard that would have caught it.

## Related

- [Develop](develop.md): the loop these gates run in, and what a change owes.
- [Test architecture](../reference/test-architecture.md): the layers, the runners, the simulators.
