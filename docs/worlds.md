# Worlds

A **world** is a deployable grouping of agents plus the runtime
constraints they live under. Worlds are declared *inline* in
`spwn.yaml` as map entries under `worlds:` - there is no
`spwn/worlds/` directory and no per-world yaml file. A world is
"alive" when at least one container is up for it, and "stopped"
otherwise; it doesn't stop existing just because nothing is running.

## Agent vs world

| Agent (persistent)                       | World (declaration + live instance)                   |
| ---------------------------------------- | ----------------------------------------------------- |
| Identity - profile, purpose, traits      | Entry in `spwn.yaml#worlds`                           |
| Memory - journal, playbooks              | Optional knowledge base at the path declared by `worlds.<name>.knowledge` |
| Composition - tools, skills, profile     | Workspace mounts + tool overrides                     |
| Evolution history                        | Running container when deployed                       |

The agent is *who*. The world is *where this agent is deployed, and
under what rules*.

## Worlds-as-deployments

A world entry is the deployment contract: *these agents, in this
workspace, possibly with these extra tools*. One agent can belong to
at most one world (enforced by `spwn check`).

```yaml
# spwn.yaml
version: 2
name: acme-api

worlds:
  default:
    agents: [neo]
    workspaces: [.]
    # Optional shared knowledge directory bound at /world/knowledge/.
    # Omit the key entirely to spawn a world whose agents are never
    # told a knowledge base exists.
    knowledge: ./knowledge
    # Optional extra tools injected on top of each agent's own tools.
    tools:
      - "spwn:docker-cli"

  lab:
    agents: [curie]
    workspaces:
      - ./experiments
      - datasets=./datasets
```

### Workspaces

Every world mounts its workspaces under `/workspaces/` inside the
container. Entries are either bare host paths (auto-named
`workspace0`, `workspace1`, …) or `name=path` (mounted at
`/workspaces/<name>`). Append `:ro` to make the mount read-only.
spwn never asks the user to write a container path — the `/workspaces/`
prefix is implicit.

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
multi-agent world by editing the file by hand - there is no implicit
migration tooling.

## Tools are structural, not permitted

No ACLs. No permission prompts. If a tool isn't listed - on the
agent, on the world, or injected by the image builder - it's
**physically impossible** inside the container, not forbidden. You
can't prompt-inject a missing binary.

```yaml
# spwn/agents/neo/agent.yaml
name: neo
runtime:
  backend: "spwn:claude-code"

tools:
  - "spwn:unix"         # bash, grep, sed, awk…
  - "spwn:git"          # version control
  - "spwn:node"         # Node.js
```

The effective tool set for a live container is the union of the
agent's `tools:` and the world's `tools:`. If two agents in the same
multi-agent world disagree on a tool's *version*, `spwn check` fails
the project - version conflicts are errors, not last-writer-wins.

## Limits

Worlds inherit Docker host defaults for CPU, memory, and disk.
Per-world hard limits are a future knob - until then, agents share
the daemon's resources.

## Spawning a world

From inside a spwn project:

```bash
spwn up                     # every world in spwn.yaml
spwn up default             # just "default"
```

`spwn up` compiles the project through `packages/compile` (render the
per-world `Tree`), assembles each agent's composition (tools + skills
+ profile) into a Docker image, boots a container, mounts the
workspaces under `/workspaces/`, and hands the runtime control. The
agent wakes up, reads `CLAUDE.md`, finds its tools, and gets to work.
