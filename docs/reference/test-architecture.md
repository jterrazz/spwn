# Test architecture

Where every suite of spwn lives, what each one proves, and what it stands on. How to run them and how to add one is [Test](../guides/test.md).

## The layers

Each layer refuses what the one above it does: a unit test never starts a container, and no Docker suite calls a provider's API.

| Layer             | Where                                                                         | Proves                                                              | Runs in                                     |
| ----------------- | ----------------------------------------------------------------------------- | ------------------------------------------------------------------- | ------------------------------------------- |
| Static            | `.golangci.yml`, the TypeScript toolchain                                     | the layer map, the lint rulebook, the names, the docs               | `make lint`                                 |
| Unit and golden   | `*_test.go` beside every Go file; `__tests__/` of `apps/web`, `apps/gate/sdk` | one module's contract, renderer output byte for byte                | `make test`                                 |
| Local integration | `apps/api/*_test.go`, through the production `Handler()`                      | every API route, on the router the server itself mounts             | `make test`                                 |
| CLI               | `specs/cli/`                                                                  | the compiled binary, a terminal session at a time, worlds in Docker | `make test`                                 |
| Website           | `specs/website/`                                                              | the web UI, rendered in a headless Chromium against `spwn web`      | `make test`                                 |
| Go Docker         | `packages/world/tests/e2e/`, `packages/compile/e2e/`                          | world lifecycles and image builds, through the Go API               | `make test-go-e2e`, `make test-compile-e2e` |
| Smoke             | `specs/cli/up/real-build.spec.ts`, `specs/cli/up/upgrade-cycle.spec.ts`       | a real image build, with no simulator in place                      | `make test-smoke`                           |

## What both specifications share

`specs/__setup__/` holds what the `cli` and the `website` specifications both read: `architect.ts`, the lock a spec holds while it drives the one architect container (`withoutArchitect`). The lock is a folder in the system's temporary directory holding its holder's pid; a holder that died, or never wrote its pid, leaves a lock the next spec takes over.

## The CLI specification

`specs/cli/cli.specification.ts` declares the one runner, `cli`, bound to `.artifacts/go/spwn`. Under it, one folder per domain of the CLI (`agent/`, `check/`, `up/`, `world/` and the rest) holds that domain's documents and chains, and the ground they stand on:

- `__fixtures__/`: the projects a spec spreads into its working directory; the pool at `specs/cli/__fixtures__/` is what several domains share, a domain's own is beside its specs.
- `__expected__/`: a golden too long to write in place.
- `__setup__/`: code a domain's chains share, such as `up/__setup__/world.ts`, which reads a world's labels back.

The runner's defaults isolate every run: `SPWN_HOME` is a folder of the run's working directory, the keychain and the providers' tokens are unset, and live credential validation is skipped. One environment is named for a document's `env:`, `sealed`, which also points `HOME` at an empty folder of the run, for a command that reads the user's home (`~/.codex`, `~/.claude`). Every container a run starts carries the run's label, `sh.spwn.test.run`, and the runner removes them at scope exit.

## The website specification

The UI is the run's, the API is each file's:

- the UI, built once per run by the project's global setup (`__setup__/ui.global-setup.ts`): `next build` in `apps/web` with `NEXT_PUBLIC_API_URL` set to a free loopback port, then `next start`, the production server `spwn web` itself starts;
- the API, `spwn web --no-open`, started by `specs/website/website.specification.ts` for each file on that port, serving a fresh project that `__setup__/project.ts` writes to a temporary directory: the `matrix` and `startup` worlds, their agents copied from `catalog/`, a spwn home past the onboarding wizard. The client reads its API address from the build, so the files take turns on it.

One folder per area of the UI (`worlds/`, `agents/`, `navigation/`), one file per page. A page is read whole through its accessibility tree, the Docker version a `{{semver}}` token. The page calls the API on a port of its own, so cross-origin requests are allowed.

## The simulators

No suite but the smoke calls a real runtime. `docker/spwn-test/` builds `spwn-test:latest`, an image where `claude` and `codex` are protocol-faithful shell scripts:

- `claude/mock.sh` accepts the real CLI's flags, inspects what the world laid down (`/agents/<name>/CLAUDE.md`, `/workspaces`, `/world/knowledge/`), and writes what it saw to `/tmp/claude-mock.json`; `--exit-code` and `--sleep` stage a failure or a timeout.
- `codex/mock.sh` speaks `codex exec --json` and `codex exec resume <session-id>`, and keeps the session ids a resume reads.

A spec pins the image with `SPWN_BASE_IMAGE=spwn-test:latest`. The scripts are contracts with the vendors' CLIs: when a real CLI changes its protocol, its simulator changes in the same commit.

## The Go Docker suites

`packages/world/tests/e2e/setup/` is their harness:

- `NewTestContext(t)` gives each test a spwn home in `t.TempDir()`, a unique label, and a cleanup that destroys every world it spawned.
- `NewSpawnBuilder(t)` spawns a world fluently and returns an assertion chain (`ExpectState`, `ExpectContainer`, `ExpectMock`), one assertion type per observable surface.
- `WaitFor(t, timeout, interval, desc, condition)` polls a condition; no test sleeps.
- `FixturesDir()` is the folder of seed projects beside the suite, `__tests__/__fixtures__/`.

`MockAssertion` reads the simulator's `/tmp/claude-mock.json` back: `SawMind()`, `SawClaudeMD()`, `SawSkill("focus")`.
