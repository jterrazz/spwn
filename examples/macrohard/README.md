# Macrohard

> Your three-agent software company in a box.

A tiny software company with a chief and two developers. Ballmer assigns work, Gates and Nadella build it. The three agents live in the same world and communicate through their per-world inboxes.

This example showcases **multi-agent hierarchy with messaging** - a leader agent decomposes work and delegates to worker agents, who report back through the inbox system.

## What's inside

| Component | Details |
|---|---|
| **World** | `macrohard` - Docker-isolated workspace |
| **Tools** | Unix, Git, Node.js 20, Python 3 |
| **Agent: ballmer** | Chief role. The product manager. Decomposes tasks, assigns work, reviews results. Energetic, decisive, results-obsessed. |
| **Agent: gates** | Worker role. The architect. Careful, systematic, thinks before coding. Writes clean, documented code. |
| **Agent: nadella** | Worker role. The builder. Fast, pragmatic, ships first and iterates. Prefers working code over perfect design. |

## Prerequisites

- spwn installed (`curl -fsSL https://spwn.sh/install.sh | bash`)
- Docker running
- An Anthropic API key (set via `claude setup-token` or `ANTHROPIC_API_KEY`)

## Install

```bash
spwn example install macrohard
```

## Spawn

```bash
# Start the company - Ballmer leads, Gates and Nadella build
spwn up -c macrohard --agent ballmer --agent gates --agent nadella

# Mount a real project for them to work on
spwn up -c macrohard --agent ballmer --agent gates --agent nadella -w ./my-project
```

## Explore

```bash
# Give Ballmer a task - he'll delegate to the workers
spwn agent talk ballmer "We need a REST API for user management. Plan it and assign the work."

# Ballmer will:
#   1. Break the task into subtasks
#   2. Message Gates and Nadella via their inboxes
#   3. Track progress by reading their journals

# Check the message flow
spwn msg inbox gates
spwn msg inbox nadella

# Watch all three agents work
spwn logs <world-id>

# Send a direct message to a worker
spwn msg send gates --from ballmer "Priority change - auth endpoints first"
```

## How the hierarchy works

1. **Ballmer** (chief) reads the task, breaks it down, and messages workers via `/world/inbox/{name}/`.
2. **Gates** and **Nadella** (workers) check their inboxes, do the work, and write results to their journals.
3. **Ballmer** periodically reads `/agents/gates/` and `/agents/nadella/` to see progress.
4. The cycle repeats - decompose, delegate, review, decide.

The two developers have different personalities on purpose: Gates writes careful, well-documented code; Nadella ships fast and iterates. The same task, assigned to both, produces different results - and Ballmer decides which approach wins.

## What to try next

```bash
# Let each agent learn from the session
spwn agent dream ballmer
spwn agent dream gates
spwn agent dream nadella

# Compare what they learned
spwn agent mind gates
spwn agent mind nadella

# Fork Gates into a specialist
spwn agent fork gates gates-frontend
```

## Cleanup

```bash
spwn down <world-id>
rm ~/.spwn/worlds/macrohard.yaml
rm -rf ~/.spwn/agents/ballmer ~/.spwn/agents/gates ~/.spwn/agents/nadella
```
