# Concepts

spwn is the **operating system for autonomous agent worlds**. You compose tools, skills, and identity into agents, then spawn them into isolated worlds where they wake up, find their tools, and get to work.

## The three abstractions

The domain has three abstractions, each owning exactly one concern:

| Abstraction | What it owns | Implementation |
| ----------- | ------------ | -------------- |
| **Runtime** | How an agent actually runs (CLI invocation, session capture, credential plumbing) | `packages/runtimes` — Claude Code today, codex next; others plug in as a ~50 LOC adapter |
| **Backend** | Where worlds run | `packages/container` — Docker; container labels are the source of truth for world state |
| **Mind** | How an agent persists across worlds | `packages/agent` — two markdown layers (`playbooks/`, `journal/`) on the host filesystem plus a single `SOUL.md` at the agent root |

Knowledge is **world-scoped**, not held in the Mind: declare a host path via `worlds.<name>.knowledge` in `spwn.yaml` (default `./spwn/knowledge`) and it is bind-mounted into `/world/knowledge/`. Omit the key and the world's agents are never told a knowledge base exists.

## Vocabulary

### Entities

- **Agent** — a persistent mind, composed from tools, skills, and an identity. Has memory and evolution history. The main thing you create and ship.
- **World** — a runtime instance. Ephemeral. Where an agent actually runs: a Docker container with filesystem, tools, and lifecycle. Dies when stopped.
- **Architect** — the always-on orchestration daemon, connected to all channels. Creates and destroys worlds; self-manages via spwn.

### Building blocks (composable, reusable)

- **Dependency** — the distribution unit. A `spwn.yaml` manifest (catalog or GitHub repo) shipping any combination of tools, skills, hooks, and agents. Installed via `spwn install`, pinned in `spwn.lock`.
- **Skill (bare form)** — a `spwn/skills/<name>.md` file. The simplest authoring path for "write a paragraph of instructions."
- **Automation** — a `worlds.<name>.automations.<id>` entry that wakes one agent on a trigger (cron expression or filesystem watch). Receipts land at `<root>/.spwn/runs.jsonl`. Engine in `packages/automation`; user guide in [Automations](automations.md).

### Agent internals

- **Soul** — who the agent is (purpose, voice, values) in a single file at `spwn/agents/<name>/SOUL.md`. Persists across world restarts.
- **Memory** — journal and sessions. Persists across worlds, grows with experience. (Knowledge is world-scoped, not agent-scoped.)
- **Composition** — an agent's active dependencies (tools, skills, hooks), declared as one `dependencies:` list in `agent.yaml` using the `spwn:` / `skill:` / `tool:` / `hook:` schemes.

### Hierarchy (inside a world)

- **Chief** — lead agent inside a world. Decomposes tasks, delegates to workers, aggregates results.
- **Worker** — persistent worker agent with its own identity and memory.
- **NPC** — ephemeral agent, no persistent memory. Single task, fire-and-forget.

### Evolution

- **Dream** — analyze experience, discover patterns, promote successes to playbooks. `spwn agent dream <name>`
- **Sleep** — graceful shutdown: save state, consolidate, prune. `spwn agent sleep <name>`
- **Fork** — clone an agent with everything it knows. `spwn agent fork <src> <dst>`

## IDs

- World: `world-{planet}-{5digits}` (e.g. `world-rhea-84721`)
- Agent: `agent-{name}-{5digits}` (e.g. `agent-neo-52103`)

Generated with `crypto/rand`.

## Development methodology: spec-first

spwn is built spec-first — the test suite is the living specification. See [Testing](07-testing.md) for the workflow (specify → encode → implement → verify) and the layer pyramid.

## Related

- [Getting started](01-getting-started.md) — install and first agent.
- [Primitives](04-primitives.md) — the on-disk shape of each block.
- [Architecture](05-architecture.md) — how the packages that own these abstractions are layered.
