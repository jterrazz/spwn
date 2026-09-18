# ADR-016: Turborepo over both toolchains

**Status:** Proposed
**Date:** 2026-09-18

## Context

This repository runs two toolchains and had no memory between runs. `go.work` lists twenty Go modules, `pnpm-workspace.yaml` lists three JavaScript packages, and the `Makefile` walked each list with a shell loop — `golangci-lint` once per module, then `pnpm -r lint`; `go test ./...` once per module. Every loop was sequential, and every run started from nothing the previous run had learned.

CI made the same trip eleven times. Each job of [`validate.yaml`](../../.github/workflows/validate.yaml) checks out, provisions its toolchain and calls one Makefile target, so a pull request that touched only `apps/web` still re-linted and re-tested twenty Go modules.

Nothing in the shape of the repository was wrong; what was missing was a task graph that knows what changed.

## Decision

Turborepo 2.11 sits in front of both toolchains. `turbo.json` at the root declares `build`, `lint` and `test`; every Makefile target that used to loop now calls `turbo run`, and CI restores the cache that run writes.

The Go half arrives through experimental workspace support, so two future flags are load-bearing and named here as the risk they are:

- `experimentalGoWorkspaces` reads `go.work` and makes each module a package, named after the last component of its module path.
- `experimentalTaskCommand` is what lets `lint` be `golangci-lint run --allow-parallel-runners ./...` instead of the `go vet ./...` Turborepo maps by default. Without it the migration would have quietly dropped every rule `.golangci.yml` adds, layering enforcement ([ADR-014](014-lint-enforced-layering.md)) included.

Four things the wiring had to work around are recorded because each is a fact about the tools, not a preference:

### The `gate` name collided

`spwn.sh/apps/gate` and `spwn.sh/packages/gate` both derive the package name `gate`, and Turborepo has no package-name override. It refuses to build the graph at all — `turbo ls` exits 1 with *"Failed to add workspace `gate` from `packages/gate/go.mod`, it already exists at `apps/gate/go.mod`"* — so every `turbo` command was dead until one of them moved.

The host app's module path became `spwn.sh/apps/gatehouse`. It was the cheap side: nothing imports that module, so the change is the one `module` line of [`apps/gate/go.mod`](../../apps/gate/go.mod), the directory stays `apps/gate`, and the binary stays `spwn-gate`. Renaming `packages/gate` instead would have touched three import sites and two `require` directives.

The word is the domain's, not the tool's: the gate is the crossing, and the gatehouse is the host-side building that houses and guards it — which is what `apps/gate` is ([Gate](../08-gate.md)).

The JavaScript package in that same directory is named `spwn-gate`, so `apps/gate` is now two packages of one graph — `gatehouse` from `go.work`, `spwn-gate` from the pnpm workspace. Turborepo accepts that; what it refuses is two packages of one NAME.

### `^build` cannot be used

`spwn.sh/packages/world` and `spwn.sh/packages/transpile` require each other. Go permits a module cycle; Turborepo reads it as a cyclic task dependency and refuses to run — *"Cyclic dependency detected: transpile#build, runtimes#build, world#build"*. No task declares `^build`, therefore. Nothing is lost: Go resolves its own compilation order through `go.work`, and no JavaScript package here depends on another.

The one ordering that does matter is declared explicitly. `build`, `lint` and `test` all depend on `dependency#generate`, the `//go:generate` mirror of `/catalog/` that `//go:embed` compiles into the binary.

### The synthetic `go-workspace` scope takes the command too

Turborepo invents a package, `go-workspace`, that depends on every module and owns no directory. A task with no command override skips it; `lint` has one, so `go-workspace#lint` tried to run `golangci-lint` at the repository root, where there is no module. `make lint` passes `--filter='!go-workspace'`, and `make test` selects the Go half as `--filter='go-workspace...'` — the same scope read the other way, as "every module the workspace holds".

### `.turbo` cannot move

`cacheDir` sends the cache to `.artifacts/turbo`, where this repository keeps every artefact. The per-task log Turborepo drops at `<package>/.turbo/turbo-<task>.log` has no such setting, and an unignored one is its own input: the first run writes it, the second hashes it and misses the cache it just filled — measured at 1 of 21 tasks cached instead of 21.

So the root `.gitignore` names `**/.turbo/`. `@jterrazz/typescript` 10.1.6 rules that `.turbo`'s home is `.artifacts/turbo/` and fails a `.gitignore` that says otherwise; it reads a pattern carrying a slash as anchored to its own file and lets this one through. That is a gap in the gate, not a permission — the toolchain owner should decide whether the rule gains an exception for a path its tool cannot relocate.

## Consequences

Measured on one shared laptop, so the numbers are indicative and the ratios are the point. *Cold* is an empty Turborepo cache with `go clean -testcache` and `golangci-lint cache clean` already run; *warm* is the same command again over an untouched tree. Go's own build cache stayed warm throughout, in both columns.

| Target           | Before, cold | Before, warm | After, cold | After, warm |
| ---------------- | ------------ | ------------ | ----------- | ----------- |
| `make lint`      | 28.84s       | 9.81s        | 16.68s      | 1.71s       |
| `make test`      | 14.72s       | 2.36s        | 7.89s       | 1.40s       |
| `make build`     | 0.64s        | 0.63s        | 1.97s       | 1.41s       |
| `make web-build` | 7.53s        | 7.06s        | 8.32s       | 1.08s       |

Cold runs got faster for a reason unrelated to caching: the loops were sequential and the task graph is not.

`make build` got slower, and stays slower. Turborepo's startup and the tar it unpacks cost about a second, which is more than `go build` spends on a tree it has already compiled. The target is kept on the graph anyway, because CI is where a cold build lives and because `.artifacts/go/spwn` restored from cache is what lets an E2E job skip compiling at all.

- **Two experimental flags are in the critical path.** A Turborepo release that changes either one breaks every gate at once. The version is pinned in `package.json` and bumping it is a change that must be read, not applied.
- **The repository root is a package now, and `tests/` became `specs/`.** The root package is what Turborepo needs to find the package manager, and it also made `tests/` a root-level test directory in the eyes of `@jterrazz/test`'s I2 rule — 63 diagnostics that were always true and never visible. They were not debt to record: I2 names `specs/` as the home of product specifications, so the member was renamed and its tree flattened one level (`tests/specs/cli/` → `specs/cli/`), which is what took the count to zero. The package is `spwn:specs`, its root IS the specs root, and every ground folder of it — `_contracts/`, `_simulators/`, `_fixtures/`, `_catalog/`, `_manual/`, `_support/` — is now read as ground by `c1-domain-structure`, which is why the one spec that sat in `_smoke/` moved to the `cli/smoke/` domain beside its sibling. The same reading catches the two vitest configs at that root: they import `@jterrazz/test/vitest`, the published entry a vitest config is meant to import and the one deep import rule F3 does not exempt, so a scoped `overrides` in [`specs/oxlint.config.ts`](../../specs/oxlint.config.ts) states it — the toolchain owner should decide whether the exemption belongs in the rule instead.
- **Every CI job needs pnpm.** A Makefile target reaches `turbo` whatever toolchain it drives, so the Go-only jobs gained a Node setup they did not have.
- **`make clean` now also empties the cache.** One `rm -rf .artifacts/` is still the whole answer, which is why the cache lives there.
