# spwn docs

The written corpus for spwn — knowledge lives here exactly once. The root [`AGENTS.md`](../AGENTS.md) (and its `CLAUDE.md` symlink) and the [`README.md`](../README.md) vitrine route here; they never restate a chapter.

## Chapters

1. [Getting started](01-getting-started.md) — install, first agent, project + config layout.
2. [Concepts](02-concepts.md) — domain model, vocabulary, IDs, evolution.
3. [CLI](03-cli.md) — grammar + command map (per-command pages generated in [`cli/`](cli/)).
4. [Primitives](04-primitives.md) — `spwn.yaml`, agents, tools, skills, hooks, commands, the dep grammar.
5. [Architecture](05-architecture.md) — monorepo layout, layered dependency graph, DooD, code style.
6. [Gate](06-gate.md) — host-side broker: cookies, MCP routing, browser sidecar (experimental).
7. [Testing](07-testing.md) — strategy, layer pyramid, running the suites.

## Topic references

- [Automations](automations.md) · [Recipes](recipes.md) · [Dependency catalog](dependency-catalog.md)
- [`cli/`](cli/) — generated Cobra man pages (regenerate with `make docs`).
- [`contributing/`](contributing/) — release runbook, self-update system.
- [`notes/`](notes/) — design rationale and audits. · [`qa/`](qa/) — manual QA passes.

This is an **application** repo: it adopts the corpus + routing doctrine but generates no `docs/reference/` (no public API surface to project).
