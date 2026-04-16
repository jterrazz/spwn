# The Matrix

> There is no spoon.

The simplest spwn example: one agent, one world. Designed to be the first
thing a new user spawns. Talk to Neo, watch it explore, understand the model.

## What this example demonstrates

- **Project deps vs agent deps.** `spwn.yaml` declares `unix` and `git` as
  project-wide deps inherited by all agents. Neo's `agent.yaml` adds only
  `node` — it does not repeat the project deps.
- **Local skills.** The `spwn/skills/` directory contains markdown files that
  teach agents specific behaviors. Neo references `world-exploration` and
  `self-reflection` in its `agent.yaml` under `skills:`.
- **Identity profiles.** Each agent has an `identity/profile.md` that defines
  its persona, voice, and traits.
- **AGENTS.md prompt.** A provider-neutral prompt file that any LLM backend
  can consume, describing how the agent should behave.
- **Lock file.** `spwn.lock` pins every resolved dependency as one line per
  entry in the format `ref version source`.

## Directory structure

```
matrix/
  spwn.yaml                          # project manifest (deps + worlds)
  spwn.lock                          # resolved dependency versions
  agents/
    neo/
      agent.yaml                     # agent config (runtime, agent deps, skills)
      AGENTS.md                      # provider-neutral agent prompt
      identity/
        profile.md                   # persona and voice
  spwn/
    skills/
      world-exploration.md           # skill: how to explore a spwn world
      self-reflection.md             # skill: how to journal observations
```

## Quick start

```bash
# Initialize from the catalog
spwn init @spwn/matrix

# Bring the world up
spwn up matrix

# Talk to Neo
spwn agent talk neo "Show me what you can see."

# Tear it down
spwn down matrix
```

## What Neo does

Neo wakes up with no prior context. Using its exploration skill, it walks
the filesystem in order — `/world/`, `~/identity/`, `/workspaces/` — and
narrates everything it finds. Using its self-reflection skill, it journals
observations so future sessions have context to build on.

The goal is not to complete a task. The goal is to make you feel like you
understand how spwn works after ten minutes of conversation.
