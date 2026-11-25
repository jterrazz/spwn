# Domain model

The words of Agents, the Spwn app that runs one agent, or a team with a chief and workers, in a world you declare, in any harness. What Spwn is as a whole is stated once, in [the README](../../README.md). In Agents you compose tools, skills, and an identity into agents, then run them in isolated worlds where they find their tools and get to work.

## The three abstractions

The domain has three abstractions, each owning exactly one concern:

| Abstraction | What it owns                                                                      | Implementation                                                                                                                    |
| ----------- | --------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| **Runtime** | How an agent actually runs (CLI invocation, session capture, credential plumbing) | `packages/runtimes`: Claude Code today, codex next; others plug in as a ~50 LOC adapter                                           |
| **Backend** | Where worlds run                                                                  | `packages/container`: Docker; container labels are the source of truth for world state                                            |
| **Mind**    | How an agent persists across worlds                                               | `packages/agent`: two markdown layers (`playbooks/`, `journal/`) on the host filesystem plus a single `SOUL.md` at the agent root |

Knowledge is **world-scoped**, not held in the Mind: declare a host path via `worlds.<name>.knowledge` in `spwn.yaml` (default `./spwn/knowledge`) and it is bind-mounted into `/world/knowledge/`. Omit the key and the world's agents are never told a knowledge base exists.

## Vocabulary

The words of Agents (agent, team, project, harness, Mind, role, skill, automation, dream, sleep, fork) are defined once, in [the glossary](../../machines/docs/concepts/glossary.md#the-words-of-the-agent-space). Three words below ship in today's `spwn` and leave with it ([the CLI](../reference/cli.md#interim-two-programs)); the glossary does not define them as they stand.

- **World**: a runtime instance. Ephemeral. Where an agent runs today: a Docker container with filesystem, tools, and lifecycle. Dies when stopped. An agent will run on a machine instead.
- **Dependency**: the distribution unit. A `spwn.yaml` manifest (catalog or GitHub repo) shipping any combination of tools, skills, hooks, and agents. Installed via `spwn install`, pinned in `spwn.lock`. What an agent is given is becoming `apps:` ([ADR-022](../decisions/022-an-agent-is-given-apps-its-machine-is-the-wall.md)).
- **Architect**: the always-on orchestration daemon, connected to all channels. Creates and destroys worlds; self-manages via spwn.

Two more terms of the file formats stay here, since only [Primitives](../reference/primitives.md) uses them: **composition** (an agent's active dependencies, one `dependencies:` list in `agent.yaml`) and **soul** (`spwn/agents/<name>/SOUL.md`).

## IDs

- World: `world-{planet}-{5digits}` (e.g. `world-rhea-84721`)
- Agent: `agent-{name}-{5digits}` (e.g. `agent-neo-52103`)

Generated with `crypto/rand`.

## Development methodology: spec-first

Spwn is built spec-first: the test suite is the living specification. [Test](../guides/test.md) says what proves a change, and how to run it.

## Related

- [Primitives](../reference/primitives.md): the project on disk, and the file each abstraction is made of.
- [Architecture](architecture.md): how the packages that own these abstractions are layered.
- [Worlds](worlds.md) · [Physics](physics.md) · [The Mind](mind.md): the design behind the three abstractions.
