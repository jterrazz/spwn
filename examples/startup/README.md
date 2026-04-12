# Startup

> Three agents, one world, one company.

A complete AI startup in a box. The CEO decides what ships, the DevOps engineer keeps the pipeline flowing, and the Analyst runs experiments. All three live in a single world and communicate through their inboxes.

This is the best example for understanding **multi-agent collaboration** — how agents with different roles coordinate without you orchestrating them manually.

## What's inside

| Component | Details |
|---|---|
| **World** | `startup` — 4 CPU, 4 GB RAM, 8 GB disk, 4h timeout |
| **Tools** | Unix, Git, Node.js 20, Python 3 |
| **Agent: ceo** | Chief role. Reads what the other agents have learned, decides what ships. Decisive, informed, accountable. |
| **Agent: devops** | Worker role. Keeps the pipeline flowing. Deploys, monitors, fixes. Pragmatic, reliable, calm under pressure. |
| **Agent: analyst** | Worker role. Runs experiments, analyzes data, reports findings. Careful, methodical, hypothesis-driven. |

## Prerequisites

- spwn installed (`curl -fsSL https://spwn.sh/install.sh | bash`)
- Docker running
- An Anthropic API key (set via `claude setup-token` or `ANTHROPIC_API_KEY`)

## Install

```bash
spwn example install startup
```

This copies the world config and all three agent profiles into `~/.spwn/`. Nothing is overwritten if you already have agents with the same names.

## Spawn

```bash
# Start the world with all three agents
spwn up -c startup --leader ceo --agent devops --agent analyst

# Or mount your own project into the world
spwn up -c startup --leader ceo --agent devops --agent analyst -w ./my-project
```

## Explore

```bash
# Talk to the CEO
spwn agent talk ceo "What should we ship this week?"

# Send a message from the CEO to the analyst
spwn msg send analyst --from ceo "Run a benchmark on the auth module"

# Check the analyst's inbox
spwn msg inbox analyst

# Watch the world
spwn logs <world-id>

# Interactive shell into the world
spwn attach <world-id>
```

## How the agents collaborate

1. **The CEO** reads `/agents/devops/` and `/agents/analyst/` memory and journal to understand what's been happening.
2. **The CEO** makes a decision and messages the relevant agent via inbox.
3. **DevOps** and **Analyst** check their inboxes, do the work, and write results to their own knowledge and journal.
4. Next cycle, the CEO reads the updated state and decides again.

No orchestration code. The collaboration emerges from the architecture — persistent memory + inboxes + clear roles.

## Next steps

```bash
# Dream — let the CEO analyze past decisions
spwn agent dream ceo

# Fork — clone the analyst to try a different approach
spwn agent fork analyst analyst-v2

# Snapshot — save the world state before a risky experiment
spwn snap save <world-id>
```

## Cleanup

```bash
spwn down <world-id>
rm ~/.spwn/worlds/startup.yaml
rm -rf ~/.spwn/agents/ceo ~/.spwn/agents/devops ~/.spwn/agents/analyst
```
