# The Matrix

> There is no spoon.

The simplest possible spwn world: one agent, one sandbox, no project. Designed to be the first thing a new user spawns - talk to Neo, watch it explore, understand the model.

Perfect for first-time users: the fastest path from "I installed spwn" to "I see an agent running in a Docker container on my machine."

## What's inside

| Component | Details |
|---|---|
| **World** | `matrix` - 2 CPU, 2 GB RAM, 4 GB disk, 1h timeout |
| **Tools** | Unix, Git, Node.js 20, Python 3 |
| **Agent: neo** | A curious, low-ego explorer. Explains what it's doing as it does it. Asks clarifying questions rather than guessing. |

## Prerequisites

- spwn installed (`curl -fsSL https://spwn.sh/install.sh | bash`)
- Docker running
- An Anthropic API key (set via `claude setup-token` or `ANTHROPIC_API_KEY`)

## Install

```bash
spwn example install matrix
```

## Spawn

```bash
# Basic sandbox (no project, just exploration)
spwn up -c matrix --agent neo

# Or mount a project for Neo to explore
spwn up -c matrix --agent neo -w ./my-project
```

## Explore

```bash
# Ask Neo to give you a tour of the world
spwn agent talk neo "Show me what you can see. Explore the world."

# Neo will walk you through:
#   /world/        - the world manifest, physics, faculties
#   /mind/         - its own persistent identity
#   /workspace/    - mounted project (if any)

# Check what's happening
spwn ls
spwn logs <world-id>

# Drop into the container yourself
spwn world enter <world-id>
```

## What to try next

```bash
# Give Neo a real task
spwn agent talk neo "Read this codebase and explain the architecture"

# Let Neo learn from the session
spwn agent dream neo

# Move on to a multi-agent example
spwn down <world-id>
spwn example install startup
```

## Cleanup

```bash
spwn down <world-id>
rm ~/.spwn/worlds/matrix.yaml
rm -rf ~/.spwn/agents/neo
```
