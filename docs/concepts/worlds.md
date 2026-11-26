# Worlds

A world is a Docker container started from the world image, and everything an agent keeps has to cross its boundary through a declared mount. This chapter is how a world runs, the port that runs it, and what moves in and out of it. The YAML that declares any of it is [Primitives](../reference/primitives.md).

## How a world runs

- **One image per tool set.** `spwn up` builds the world image from the tools the agents declare, tagged by their hash ([ADR-020](../decisions/020-a-world-image-is-tagged-by-its-tool-sets-hash.md)), and renders the declared reality (agents, skills, rendered context) into the container it starts; nothing is installed at world-start time.
- **Container labels are the world state.** The running backend is queried directly; there is no shadow registry to drift.
- **Lifecycle**: create, start, run agents, then stop or destroy. Destroying a world removes the container; everything worth keeping lives in a mount or in the agent's Mind.
- **Session continuity survives the container.** An agent's session state persists with its Mind on the host, so re-entering a world (or being respawned after a crash) resumes the conversation instead of starting from zero.

## The Backend port

The machinery programs against an interface (provision, exec, mount, destroy, ensure-image), never against Docker itself ([`packages/container`](../../packages/container) is the adapter, and [ADR-010](../decisions/010-ports-and-adapters.md) the rule). Nothing above the port names Docker, which keeps the door open for stronger isolation backends without changing agent-facing behaviour.

Docker is the default and, today, the only adapter. Designed, not shipped: edge deployment (ephemeral agents on lightweight infrastructure) as a second adapter behind the same port.

## What crosses the boundary

| Mechanism        | Direction       | Use case                                  | Persistence            |
| ---------------- | --------------- | ----------------------------------------- | ---------------------- |
| Workspace mount  | Bidirectional   | Task input and output (code, data, files) | Persists on the host   |
| Knowledge mount  | Read into world | World-scoped reference material           | Persists on the host   |
| Mind persistence | Bidirectional   | Agent memory (playbooks, journal)         | Persists across worlds |
| Baked blocks     | Into the image  | Tools, skills, rendered context           | Rebuilt with the image |

- **Workspaces** are host paths a world declares, each mounted at `/workspaces/<name>`, its name a slug (`^[a-z][a-z0-9-]*$`) whichever way it is given: the way an agent is handed a project and its output collected. When the world is destroyed, the results stay on the host.
- **The world's user follows the host's on Linux.** When a world starts on a Linux host whose user is not root, the world's `spwn` user takes that user's uid and gid, so the agent owns the workspace as the host user does. The world image needs a `spwn` user and passwordless sudo for it. On macOS, Docker Desktop and OrbStack translate the ownership of a mount. The host is read where the CLI runs, not where the daemon does, so these setups are not covered: a remote daemon that is not on Linux, a macOS CLI driving a remote Linux daemon (the remap is skipped there), a rootless daemon, a daemon with user-namespace remapping, where the container's uids are not the host's, and the architect container, which gets no remap.
- **Knowledge** is world-scoped, not agent-scoped: the path a world names is mounted at `/world/knowledge/`. Omit the key and no mount happens; the agents are never told a knowledge base exists.
- **A host path outside the project is asked for once.** `spwn up` resolves each workspace and the knowledge folder against the project's root, by name and through the links the project itself holds, and mounts that resolved path, with the access it was asked for. One that leaves the project (absolute elsewhere, `..`, or a project link pointing outside) is shown once, quoted and absolute as far as the project's links tell (nothing outside is read before the yes), with what it grants, and asked on a terminal (stdin and stderr both, and anything typed before the ask discarded): once per run, for every world `spwn up` starts, before any of them starts and before anything of the project is printed. A typed yes is kept, once its world started, in `~/.spwn/grants.yaml`, never in the project; off a terminal with no kept yes, `spwn up` refuses, `outside.not-granted`, naming the path. `spwn up --yes` allows the paths for that run only, prints each on stderr (`allowed for this run: "<path>" (<what it grants>)`), and keeps nothing. `spwn automation daemon` asks the same of an fs watch path, whose files' names and times are recorded in the receipts and passed to the project's agent. A path given by `-w` is yours and is not asked. Whatever the CLI prints of the project (a file or world name, a link's text, a path, a value of spwn.yaml) has its control, bidi and invisible characters escaped; what a world's agents print to their own terminal (`spwn agent talk`, an attached session, a followed log) is theirs and is not ([ADR-023](../decisions/023-a-host-path-outside-the-project-is-asked-for-once.md)).
- **A path is resolved again where it is used, since a world of the project may rewrite it.** A running world can mount the project read-write and turn a folder into a link while another world starts, so each workspace and knowledge path is resolved again just before its world's container is made and after it started, and each watch path before and after the watches start: one that resolves elsewhere now refuses that world, `outside.changed-while-starting`, naming the path, with nothing kept. Docker binds by path when the container starts, so a link swapped between the two checks and swapped back is still not seen.
- **A kept yes belongs to the project's folder, not its content.** It is keyed by the project folder's resolved path, the path and the access: a different repository later cloned or moved into the same folder inherits it. `~/.spwn/grants.yaml` lists every kept yes; to revoke one, edit the file and delete its entry.
- **spwn.yaml's names and host paths are checked before anything is shown.** A world, agent or workspace name, a workspace, knowledge or watch path that holds a control character (an escape, a newline, a tab, a C1 code) is refused, `spwn-yaml.control-character`, and a workspace name must be a slug, so the project cannot redraw the ask. A host path that starts with `~` is refused, `spwn-yaml.tilde`: spwn.yaml never expands it, so write the absolute path (one outside the project is then asked for once). A host path holding a `:` beyond a workspace's `name=path:ro`, as written or as the project's links resolve it, is refused, `spwn-yaml.colon`: Docker reads a mount as `source:target:mode`, so the colon would bind a prefix of the path that was checked.
- **Known limit: running a cloned project's agents trusts that project.** Every world receives your credentials read-only at `/credentials` and the gh cache read-write at `/credentials/gh`, and its agents run the prompts the project gives them, so the CLI keeping its own reads and writes inside the project does not make an untrusted project safe to run.
- **The Mind** (`SOUL.md`, playbooks, journal) lives on the host filesystem and travels with the agent across worlds; the journal is appended as sessions run ([The Mind](mind.md)).
- **Everything composed** (tools, skills, the rendered operating manual) is baked into the image at build time and delivered to the paths each runtime expects. No bind-mount indirection and no symlinks, so a world's capabilities are immutable while it runs.

## Invariants

- **Mounts are explicit.** Only declared directories cross the boundary, so what an agent can touch on the host is enumerable from the manifests.
- **Worlds are disposable; their mounts are not.** Everything an agent should keep lands in a mount; everything else dies with the world.
- **Changes through a mount are immediately visible on the host**: these are bind mounts, not copies. Sharing one Mind between two live worlds is therefore possible and demands care; fork the agent instead.
- **Secrets are not files in a world.** Credentialed capabilities reach agents through the [gate](gate.md), which keeps credentials host-side.

## Related

- [Physics](physics.md): what exists inside the world once it is running.
- [Architecture](architecture.md): where the container adapter sits in the layer graph.
- [Primitives](../reference/primitives.md): `workspaces:`, `knowledge:`, and the blocks that get baked in.
