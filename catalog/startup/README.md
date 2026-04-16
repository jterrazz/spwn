# Startup

> Three agents, one world, one company.

A complete AI startup in a box. The CEO decides what ships, the DevOps engineer keeps the pipeline flowing, and the Analyst runs experiments. All three live in a single world and communicate through their inboxes.

This is the best example for understanding **multi-agent collaboration** and the **dependency inheritance model** - how agents share project-wide deps while adding their own, how local skills are shared across agents, and how hooks run at spawn time.

## Architecture

```
startup/
  spwn.yaml              # project deps: unix, git, node (inherited by all)
  spwn.lock              # resolved dependency manifest
  spwn/
    skills/              # local skills, shared across agents
      sprint-planning.md
      code-review.md
      deployment.md
    hooks/               # lifecycle hooks
      pre-spawn.sh       # runs before world initialization
  agents/
    ceo/                 # chief role, no extra deps
      agent.yaml         # skills: sprint-planning
      AGENTS.md
      identity/profile.md
    devops/              # worker role, adds @spwn/docker-cli
      agent.yaml         # skills: deployment, code-review
      AGENTS.md
      identity/profile.md
    analyst/             # worker role, adds @spwn/python
      agent.yaml         # skills: sprint-planning, code-review
      AGENTS.md
      identity/profile.md
```

## Dependency model

| Layer | Deps | Defined in |
|---|---|---|
| **Project-wide** | @spwn/unix, @spwn/git, @spwn/node | `spwn.yaml` |
| **CEO** | (inherits project deps only) | `agents/ceo/agent.yaml` |
| **DevOps** | + @spwn/docker-cli | `agents/devops/agent.yaml` |
| **Analyst** | + @spwn/python | `agents/analyst/agent.yaml` |

Agents inherit all project-wide deps automatically. Their `agent.yaml` only lists **additions** - never repeat what's already in `spwn.yaml`.

## What's inside

| Component | Details |
|---|---|
| **World** | `startup` - all three agents, workspace mounted at `.` |
| **Shared deps** | Unix, Git, Node.js (every agent gets these) |
| **Agent: ceo** | Chief role. No extra deps. Uses sprint-planning skill. |
| **Agent: devops** | Worker role. Adds Docker CLI. Uses deployment + code-review skills. |
| **Agent: analyst** | Worker role. Adds Python. Uses sprint-planning + code-review skills. |
| **Skills** | sprint-planning, code-review, deployment (shared local skills) |
| **Hook** | pre-spawn.sh logs world initialization |

## Prerequisites

- spwn installed (`curl -fsSL https://spwn.sh/install.sh | bash`)
- Docker running
- An Anthropic API key (set via `claude setup-token` or `ANTHROPIC_API_KEY`)

## Install

```bash
spwn init @spwn/startup
```

This copies the world config and all three agent profiles into `~/.spwn/`. Nothing is overwritten if you already have agents with the same names.

## Spawn

```bash
# Start the world with all three agents
spwn up -c startup --agent ceo --agent devops --agent analyst

# Or mount your own project into the world
spwn up -c startup --agent ceo --agent devops --agent analyst -w ./my-project
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

# Interactive shell inside the world
spwn world enter <world-id>
```

## How the agents collaborate

1. **The CEO** reads `/agents/devops/` and `/agents/analyst/` memory and journal to understand what's been happening.
2. **The CEO** makes a decision and messages the relevant agent via inbox.
3. **DevOps** and **Analyst** check their inboxes, do the work, and write results to their own knowledge and journal.
4. Next cycle, the CEO reads the updated state and decides again.

No orchestration code. The collaboration emerges from the architecture - persistent memory + inboxes + clear roles.

## Key concepts demonstrated

- **Dep inheritance**: project deps in `spwn.yaml` flow to all agents; agent-specific deps are additions only.
- **Local skills**: markdown files in `spwn/skills/` injected into agent context at boot. Multiple agents can reference the same skill.
- **Local hooks**: shell scripts in `spwn/hooks/` that run at lifecycle events (pre-spawn, post-spawn, etc.).
- **Role hierarchy**: the `chief` role (CEO) can read other agents' state; `worker` roles (devops, analyst) focus on their domain.

## Next steps

```bash
# Dream - let the CEO analyze past decisions
spwn agent dream ceo

# Fork - clone the analyst to try a different approach
spwn agent fork analyst analyst-v2

# Snapshot - save the world state before a risky experiment
spwn snap save <world-id>
```

## Cleanup

```bash
spwn down <world-id>
rm ~/.spwn/worlds/startup.yaml
rm -rf ~/.spwn/agents/ceo ~/.spwn/agents/devops ~/.spwn/agents/analyst
```
