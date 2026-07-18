# spwn — agent brief

The operating system for autonomous agent worlds. Compose tools, skills, and identity into **agents**, then spawn them into isolated Docker **worlds**. This file is a map, not the territory — it routes into the [`docs/`](docs/) corpus, where each piece of knowledge lives exactly once. When you need the *what* or the *why*, follow the link; don't expect it restated here.

## Mental model

Three abstractions, each owning one concern:

- **Runtime** (`packages/runtimes`) — how an agent runs (Claude Code today, codex next).
- **Backend** (`packages/container`) — where worlds run (Docker; labels are the source of truth).
- **Mind** (`packages/agent`) — how an agent persists across worlds (`SOUL.md` + `playbooks/` + `journal/`).

Knowledge is world-scoped, not held in the Mind. A spwn project lives **in the repo** (`./spwn/`), not in `~/.spwn/`. Full model in [Concepts](docs/02-concepts.md).

## Where knowledge lives

| Task | Chapter |
| ---- | ------- |
| Install, first agent, project + config layout | [`docs/01-getting-started.md`](docs/01-getting-started.md) |
| Domain model, vocabulary, IDs, evolution | [`docs/02-concepts.md`](docs/02-concepts.md) |
| CLI grammar + command map (generated pages in `docs/cli/`) | [`docs/03-cli.md`](docs/03-cli.md) |
| `spwn.yaml`, agents, tools, skills, hooks, commands, dep grammar | [`docs/04-primitives.md`](docs/04-primitives.md) |
| Monorepo layout, layered dependency graph, DooD, code style | [`docs/05-architecture.md`](docs/05-architecture.md) |
| Host-side gate: cookies, MCP routing, browser sidecar | [`docs/06-gate.md`](docs/06-gate.md) |
| Testing strategy, layer pyramid, running the suites | [`docs/07-testing.md`](docs/07-testing.md) |
| Automations (cron + fs triggers) | [`docs/automations.md`](docs/automations.md) |
| Worked examples | [`docs/recipes.md`](docs/recipes.md) |
| Built-in `spwn:*` catalog | [`docs/dependency-catalog.md`](docs/dependency-catalog.md) |
| Release runbook, self-update system | [`docs/contributing/`](docs/contributing/) |
| Deep test-suite reference + simulators | [`tests/ARCHITECTURE.md`](tests/ARCHITECTURE.md) |

## Working in this repo

- **Single entry point is the `Makefile`.** `make` (no args) lists every target. CI is [`.github/workflows/validate.yaml`](.github/workflows/validate.yaml) — the workflow *is* the aggregate; there is no `test-pr` meta-target.
- **Common gates:** `make lint` · `make test` (Go unit) · `make test-contracts` · `make test-cli` (Docker). Full matrix in [Testing](docs/07-testing.md).
- **Layers flow downward**, enforced by depguard in [`.golangci.yml`](.golangci.yml) (the mechanical source of truth) — see [Architecture](docs/05-architecture.md) before moving code between packages.
- **Spec-first:** the test suite is the specification. A discovery grows a guard (test / check / runtime error) in the same change.
- Contributor setup: [`CONTRIBUTING.md`](CONTRIBUTING.md).

`CLAUDE.md` is a symlink to this file.
