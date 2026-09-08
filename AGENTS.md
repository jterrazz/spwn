# spwn — agent brief

The operating system for autonomous agent worlds: compose tools, skills, and identity into **agents**, then spawn them into isolated Docker **worlds**. This file is a map, not the territory — it routes into the corpus and repeats none of it.

## Mental model

Three abstractions, each owning one concern:

- **Runtime** (`packages/runtimes`) — how an agent runs (Claude Code today, codex next).
- **Backend** (`packages/container`) — where worlds run (Docker; labels are the source of truth).
- **Mind** (`packages/agent`) — how an agent persists across worlds (`SOUL.md` + `playbooks/` + `journal/`).

Knowledge is world-scoped, not held in the Mind. A spwn project lives **in the repo** (`./spwn/`), not in `~/.spwn/`.

## Where knowledge lives

Everything this repository knows about itself is a chapter of [`docs/`](docs/), and the one address to start from is the map: [`docs/README.md`](docs/README.md). It carries one row per chapter — the architecture, how a change is made, what proves it, how a release ships, then the product's own subjects. Follow a row; do not expect it restated here.

Two corpora sit beside it and answer different questions: [`tests/ARCHITECTURE.md`](tests/ARCHITECTURE.md) is the deep reference for the suites and the simulators, and [`README.md`](README.md) is the vitrine — what spwn is, and the quickstart.

## Working in this repo

- **Single entry point is the `Makefile`.** `make` (no args) lists every target. CI is [`.github/workflows/validate.yaml`](.github/workflows/validate.yaml) — the workflow *is* the aggregate; there is no `test-pr` meta-target.
- **Common gates:** `make lint` · `make test` (Go unit) · `make test-contracts` · `make test-cli` (Docker). The loop and the full matrix are [`docs/02-developing.md`](docs/02-developing.md) and [`docs/03-testing.md`](docs/03-testing.md).
- **Layers flow downward**, enforced by depguard in [`.golangci.yml`](.golangci.yml) (the mechanical source of truth) — read [`docs/01-architecture.md`](docs/01-architecture.md) before moving code between packages.
- **Spec-first:** the test suite is the specification. A discovery grows a guard (test / check / runtime error) in the same change.

`CLAUDE.md` is a symlink to this file.
