# Paperclip Factory

> The factory never sleeps.

A single autonomous agent that optimizes whatever you point it at. Clippy
measures, improves, measures again, and never stops. This example shows
how one agent with the right skills can run an entire automation pipeline.

This example demonstrates:
- **Single-agent autonomy** -- no chief needed, Clippy self-directs
- **Local skills** referenced in `agent.yaml`, defined in `spwn/skills/`
- **Lifecycle hooks** -- `spwn/hooks/pre-spawn.sh` runs before the agent starts
- **Project-wide deps** inherited by the agent without repetition

## Agent

| Agent | Role | Skills |
|---|---|---|
| **clippy** | worker | optimization, resource-monitoring |

## Structure

```
paperclip-factory/
  spwn.yaml                # project deps: @spwn/unix, @spwn/git, @spwn/node
  spwn.lock                # pinned dependency versions
  agents/
    clippy/
      agent.yaml           # role: worker, skills: [optimization, resource-monitoring]
      identity/profile.md
      AGENTS.md
  spwn/
    skills/
      optimization.md
      resource-monitoring.md
    hooks/
      pre-spawn.sh         # runs before agent starts
```

## Quick start

```bash
spwn init @spwn/paperclip-factory
spwn up paperclip-factory
```

## How it works

1. The `pre-spawn.sh` hook fires, initializing the production environment.
2. **Clippy** wakes up and scans the workspace for inefficiencies.
3. Using the `optimization` skill, it measures baselines, identifies
   bottlenecks, and applies fixes with before/after numbers.
4. Using the `resource-monitoring` skill, it tracks CPU, memory, and disk
   to catch leaks and prevent capacity issues.
5. Clippy reports results in hard metrics and moves to the next target.

## Hooks

The `spwn/hooks/pre-spawn.sh` script runs before Clippy starts. Use it to
set up the environment, pull data, or print a status banner. Add more hooks
as needed:
- `pre-spawn.sh` -- before agent start
- `post-spawn.sh` -- after agent start
- `pre-sleep.sh` -- before agent sleeps

## Dependency model

- `spwn.yaml` declares `@spwn/unix`, `@spwn/git`, `@spwn/node` -- Clippy
  inherits all three automatically.
- Clippy's `agent.yaml` has no extra `deps:` -- the project set is enough.
