# Worlds

A world is a Docker container started from the project image, and everything an agent keeps has to cross its boundary through a declared mount. This chapter is how a world runs, the port that runs it, and what moves in and out of it. The YAML that declares any of it is [Primitives](07-primitives.md).

## How a world runs

- **One image per project.** `spwn build` compiles the declared reality — agents, tools, skills, rendered context — into a project-specific image. Worlds are containers started from that image; nothing is installed at world-start time.
- **Container labels are the world state.** The running backend is queried directly; there is no shadow registry to drift.
- **Lifecycle**: create → start → run agents → stop or destroy. Destroying a world removes the container; everything worth keeping lives in a mount or in the agent's Mind.
- **Session continuity survives the container.** An agent's session state persists with its Mind on the host, so re-entering a world — or being respawned after a crash — resumes the conversation instead of starting from zero.

## The Backend port

The machinery programs against an interface — provision, exec, mount, destroy, ensure-image — never against Docker itself ([`packages/container`](../packages/container) is the adapter, and [ADR-011](decisions/011-ports-and-adapters.md) the rule). Nothing above the port names Docker, which keeps the door open for stronger isolation backends without changing agent-facing behaviour.

Docker is the default and, today, the only adapter. Designed, not shipped: edge deployment — ephemeral agents on lightweight infrastructure — as a second adapter behind the same port.

## What crosses the boundary

| Mechanism        | Direction       | Use case                                  | Persistence            |
| ---------------- | --------------- | ----------------------------------------- | ---------------------- |
| Workspace mount  | Bidirectional   | Task input and output (code, data, files) | Persists on the host   |
| Knowledge mount  | Read into world | World-scoped reference material           | Persists on the host   |
| Mind persistence | Bidirectional   | Agent memory (playbooks, journal)         | Persists across worlds |
| Baked blocks     | Into the image  | Tools, skills, rendered context           | Rebuilt with the image |

- **Workspaces** are host paths a world declares, mounted at `/workspace` — the way an agent is handed a project and its output collected. When the world is destroyed, the results stay on the host.
- **Knowledge** is world-scoped, not agent-scoped: the path a world names is mounted at `/world/knowledge/`. Omit the key and no mount happens — the agents are never told a knowledge base exists.
- **The Mind** — `SOUL.md`, playbooks, journal — lives on the host filesystem and travels with the agent across worlds; the journal is appended as sessions run ([The Mind](11-mind.md)).
- **Everything composed** — tools, skills, the rendered operating manual — is baked into the image at build time and delivered to the paths each runtime expects. No bind-mount indirection and no symlinks, so a world's capabilities are immutable while it runs.

## Invariants

- **Mounts are explicit.** Only declared directories cross the boundary, so what an agent can touch on the host is enumerable from the manifests.
- **Worlds are disposable; their mounts are not.** Everything an agent should keep lands in a mount; everything else dies with the world.
- **Changes through a mount are immediately visible on the host** — these are bind mounts, not copies. Sharing one Mind between two live worlds is therefore possible and demands care; fork the agent instead.
- **Secrets are not files in a world.** Credentialed capabilities reach agents through the [gate](08-gate.md), which keeps credentials host-side.

## Related

- [Physics](10-physics.md) — what exists inside the world once it is running.
- [Architecture](01-architecture.md) — where the container adapter sits in the layer graph.
- [Primitives](07-primitives.md) — `workspaces:`, `knowledge:`, and the blocks that get baked in.
