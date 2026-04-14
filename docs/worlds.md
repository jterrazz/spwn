# Worlds

A world is a runtime instance - the ephemeral Docker container where an agent actually runs. An agent can live in many worlds over time; the agent persists, the world doesn't.

## What lives on the agent vs the world

| Agent (persistent) | World (ephemeral) |
| --- | --- |
| Identity - purpose, traits, profile | Running container and processes |
| Memory - journal, knowledge, sessions | Mounted workspace directory |
| Composition - tools, skills, profile | Active tool bridges |
| Evolution history | Live state + logs |

The agent is "who". The world is "where, right now".

## Tools are structural

No ACLs. No permission prompts. If a tool isn't listed in the agent's composition, it's **physically impossible** inside its world - not "forbidden," absent. You can't prompt-inject a missing binary.

```yaml
# ~/.spwn/agents/neo/agent.yaml
name: neo
runtime: claude-code

profile: the-one

tools:
  - "@spwn/unix"       # bash, grep, sed, awk…
  - "@spwn/git"        # version control
  - "@spwn/node"       # Node.js
  - "@spwn/claude-code" # thinking engine

skills:
  - paper-reading
  - refactoring
```

If `curl` is not listed, HTTP doesn't exist in Neo's world. Tools are composable, dependency-aware, and verified at world creation. Each tool ships its own skills (Vercel SKILL.md convention). The image is built on-demand from your exact tool selection - no bloated base images.

## Spawning a world

```bash
spwn up --agent neo -w ./my-project
```

This assembles Neo's composition (tools + skills + profile) into a Docker image, boots a container, mounts your workspace, and hands Neo the shell. The agent wakes up, finds its tools, and gets to work.

## Why Spwn is different

| | |
|---|---|
| **Composable intelligence.** | Tools, skills, and profiles are stackable blocks. Mix them into agents like Docker layers into images. |
| **Worlds over wrappers.** | Not another API layer. A full environment with filesystem, compute, and neighbors. |
| **Identity over instances.** | Agents have persistent purpose, skills, and memory. They're individuals, not stateless functions. |
| **Agency over tools.** | MCP gives agents a Swiss Army knife. Spwn gives them a workshop. They discover, compose, and create. |
| **Structure over permissions.** | No ACLs. No allowlists. If a tool isn't listed, it doesn't exist. Security is absence. |
| **Evolution over configuration.** | Agents learn from tasks via dream, consolidate knowledge during sleep, and branch via forking. |

## What was here before

An earlier version of spwn described a `physics:` block with CPU/memory/timeout constants. That configuration still works at the world runtime layer, but it's no longer part of the user-facing mental model. Think in terms of **agents** and **worlds** - the runtime constraints fade into infrastructure.
