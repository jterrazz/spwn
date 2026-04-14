# Worlds

A world is a runtime instance - the ephemeral Docker container where an agent actually runs. An agent can live in many worlds over time; the agent persists, the world doesn't.

## Agent vs world

| Agent (persistent)                       | World (ephemeral)              |
| ---------------------------------------- | ------------------------------ |
| Identity - profile, purpose, traits      | Running container              |
| Memory - journal, knowledge, playbooks   | Mounted workspace              |
| Composition - tools, skills, profile     | Live tool bridges              |
| Evolution history                        | Process state + logs           |

The agent is *who*. The world is *where, right now*.

## Tools are structural, not permitted

No ACLs. No permission prompts. If a tool isn't listed in the agent's composition, it's **physically impossible** inside its world - not forbidden, absent. You can't prompt-inject a missing binary.

```yaml
# spwn/agents/neo/agent.yaml
name: neo
runtime: claude-code

tools:
  - "@spwn/unix"         # bash, grep, sed, awk…
  - "@spwn/git"          # version control
  - "@spwn/node"         # Node.js
  - "@spwn/claude-code"  # thinking engine
```

If `curl` isn't listed, HTTP doesn't exist in neo's world. Tools are composable, dependency-aware, and verified at world creation. The image is built on-demand from the exact selection - no bloated base image.

Tools are **world-scoped**: when a world is destroyed, its tool bridges go with it. Installing a tool via `spwn agent add` updates the agent's composition; the next `spwn up` rebuilds the world with the new tool available.

## Physics

Worlds also carry hard limits declared in `spwn/worlds/<name>.yaml`:

```yaml
physics:
  constants:
    cpu: 2          # CPU cores
    memory: 1g      # RAM
    disk: 4g        # rootfs cap
    timeout: 30m    # wall clock per session
```

These are kernel-enforced. An agent can't burn your battery or fill your disk by accident.

## Spawning a world

From inside a spwn project:

```bash
spwn up                     # use the world + agents declared in spwn.yaml
spwn up --agent neo         # override which agent spawns
spwn up --build             # rebuild the artifact first, then spawn
```

`spwn up` assembles the agent's composition (tools + skills + profile) into a Docker image, boots a container, mounts the workspace at `/work/`, and hands the runtime control. The agent wakes up, reads `CLAUDE.md`, finds its tools, and gets to work.
