# Worlds

A **world** is a deployable grouping of agents plus the runtime
constraints they live under. Worlds are declared *inline* in
`spwn.yaml` as map entries under `worlds:` — there is no
`spwn/worlds/` directory and no per-world yaml file. A world is
"alive" when at least one container is up for it, and "stopped"
otherwise; it doesn't stop existing just because nothing is running.

## Agent vs world

| Agent (persistent)                       | World (declaration + live instance) |
| ---------------------------------------- | ----------------------------------- |
| Identity - profile, purpose, traits      | Entry in `spwn.yaml#worlds`         |
| Memory - journal, knowledge, playbooks   | Mounted workspace                   |
| Composition - tools, skills, profile     | Physics caps + tool overrides       |
| Evolution history                        | Running container when deployed     |

The agent is *who*. The world is *where this agent is deployed, and
under what rules*.

## Worlds-as-deployments

A world entry is the deployment contract: *these agents, in this
workspace, with this physics, possibly with these extra tools*. One
agent can belong to at most one world (enforced by `spwn check`).

```yaml
# spwn.yaml
version: 2
name: acme-api

worlds:
  default:
    agents: [neo]
    workspaces: [.]
    physics:
      cpu: 2
      memory: 2g
    # Optional extra tools injected on top of each agent's own tools.
    tools:
      - "@spwn/docker-cli"

  lab:
    agents: [curie]
    workspaces:
      - ./experiments
      - ./datasets:/workspace/datasets
    physics:
      cpu: 4
      memory: 4g
```

### Workspaces

Every world mounts at least one workspace under `/workspace` inside
the container. The *first* entry may be a bare host path (mounted at
`/workspace`). Any additional entries must use the explicit
`host:/workspace/<name>` form so the target directory is unambiguous.

### Agents in worlds

`agents:` names directories under `spwn/agents/`. `spwn check`
rejects a world that references a missing directory, and it also
rejects an agent listed in more than one world (one-agent-one-world).

### Bringing worlds up and down

```bash
spwn up                     # start every world in spwn.yaml (compose-style)
spwn up default             # start one world by name
spwn agent neo              # start the world that contains neo
spwn down                   # stop every world
spwn world stop lab         # stop one world
```

Creating an agent with `spwn agent new <name>` automatically inserts
a single-agent world into `spwn.yaml` so you never end up with an
agent that has nowhere to run. You can later merge it into a
multi-agent world by editing the file by hand — there is no implicit
migration tooling.

## Tools are structural, not permitted

No ACLs. No permission prompts. If a tool isn't listed — on the
agent, on the world, or injected by the image builder — it's
**physically impossible** inside the container, not forbidden. You
can't prompt-inject a missing binary.

```yaml
# spwn/agents/neo/agent.yaml
name: neo
runtime:
  backend: "@spwn/claude-code"

tools:
  - "@spwn/unix"         # bash, grep, sed, awk…
  - "@spwn/git"          # version control
  - "@spwn/node"         # Node.js
```

The effective tool set for a live container is the union of the
agent's `tools:` and the world's `tools:`. If two agents in the same
multi-agent world disagree on a tool's *version*, `spwn check` fails
the project — version conflicts are errors, not last-writer-wins.

## Physics

Worlds carry hard limits declared in their `spwn.yaml` entry:

```yaml
worlds:
  default:
    agents: [neo]
    workspaces: [.]
    physics:
      cpu: 2          # CPU cores  (Docker --cpus)
      memory: 2g      # RAM limit  (Docker -m)
```

These are the Docker-enforceable knobs: CPU (`--cpus`) and memory
(`-m`). Network mode (bridge, outbound enabled) and the ephemeral,
read-only-by-default filesystem are part of the same contract and are
also enforced by the container runtime.

Disk quotas and wall-clock timeouts are **not yet enforced** — they
would require Docker `storage-opt` (devicemapper-only, not portable)
or external supervision machinery, so they are out of scope for now.
When `physics:` is omitted, host defaults apply.

## Spawning a world

From inside a spwn project:

```bash
spwn up                     # every world in spwn.yaml
spwn up default             # just "default"
```

`spwn up` first flattens the project into `.spwn/build/` (validate +
content-hash + pin), then assembles each agent's composition (tools +
skills + profile) into a Docker image, boots a container, mounts the
workspaces under `/workspace`, and hands the runtime control. The
agent wakes up, reads `CLAUDE.md`, finds its tools, and gets to work.
The build step is a no-op when the cache hash matches.
