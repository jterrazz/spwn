# Macrohard

> A corporate simulation with three agents and local skills.

Macrohard is a multi-agent example that models a small software company.
Nadella sets direction, Gates builds the technical foundation, and Ballmer
sells whatever they ship. Each agent has distinct skills loaded from
`spwn/skills/`, and they collaborate through the spwn inbox system.

This example demonstrates:
- **Project-wide deps** in `spwn.yaml` inherited by all agents
- **Agent-specific deps** (Gates adds `@spwn/python` on top of the shared set)
- **Local skills** referenced by name in `agent.yaml`, defined in `spwn/skills/`
- **Role hierarchy** -- one chief (Nadella) delegates to two workers

## Agents

| Agent | Role | Skills | Extra Deps |
|---|---|---|---|
| **nadella** | chief | product-strategy | -- |
| **gates** | worker | code-review, architecture | @spwn/python |
| **ballmer** | worker | sales-pitch | -- |

## Structure

```
macrohard/
  spwn.yaml              # project deps: @spwn/unix, @spwn/git, @spwn/node
  spwn.lock               # pinned dependency versions
  agents/
    nadella/
      agent.yaml          # role: chief, skills: [product-strategy]
      identity/profile.md
      AGENTS.md
    gates/
      agent.yaml          # role: worker, deps: [@spwn/python], skills: [code-review, architecture]
      identity/profile.md
      AGENTS.md
    ballmer/
      agent.yaml          # role: worker, skills: [sales-pitch]
      identity/profile.md
      AGENTS.md
  spwn/
    skills/
      product-strategy.md
      code-review.md
      architecture.md
      sales-pitch.md
```

## Quick start

```bash
spwn init @spwn/macrohard
spwn up macrohard
```

## How it works

1. **Nadella** (chief) receives the user's goal, applies the `product-strategy`
   skill to decompose it, and messages Gates and Ballmer with assignments.
2. **Gates** (worker) picks up technical tasks, uses `code-review` and
   `architecture` skills, and reports back with working code.
3. **Ballmer** (worker) picks up communication tasks, uses the `sales-pitch`
   skill, and delivers copy, demos, or pitch decks.
4. Nadella synthesizes their outputs and decides the next move.

## Dependency model

- `spwn.yaml` declares `@spwn/unix`, `@spwn/git`, `@spwn/node` -- every
  agent inherits these automatically.
- Gates' `agent.yaml` adds `@spwn/python` -- only he gets Python.
- Nadella and Ballmer have no extra deps -- they rely on the project set.
