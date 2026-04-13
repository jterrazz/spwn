# Agent Composition

An agent is a directory of markdown files — human-readable, version-controllable, no database.

## Structure

```
~/.spwn/agents/neo/
├── agent.yaml                # composition: tools, skills, profile, runtime
├── profile.md                # personality — role, style, purpose, behavior
├── skills/                   # procedures, checklists, playbooks (authored or installed)
├── knowledge/                # facts about the codebase, team, domain
├── playbooks/                # step-by-step workflows promoted from experience
└── journal/                  # session logs — one file per world session
```

The `agent.yaml` declares the composition. Everything else is markdown you can read, edit, and git-diff.

## The building blocks

An agent is composed from three kinds of reusable blocks:

| Block | What it is | Where it lives |
|---|---|---|
| **Tool** | A capability pack (`@spwn/unix`, `@spwn/python`). Shell binaries the agent can invoke. | `~/.spwn/tools/` |
| **Skill** | A procedure, playbook, or piece of knowledge authored in markdown. | `~/.spwn/skills/` and per-agent `skills/` |
| **Profile** | A personality template — role, tone, purpose, behavior. | `~/.spwn/profiles/` |

You stack them into an agent:

```yaml
# ~/.spwn/agents/neo/agent.yaml
name: neo
runtime: claude-code

profile: the-one

tools:
  - "@spwn/unix"
  - "@spwn/git"
  - "@spwn/python"

skills:
  - paper-reading
  - refactoring
  - "@community/rust-review"
```

## Memory

Memory is what the agent *has done*, not what it *can do*. It lives on the agent:

| Biology | Layer | What it stores |
|---------|-------|----------------|
| Semantic memory | **Knowledge** | Facts about the codebase, the team, the domain |
| Procedural memory | **Playbooks** | Step-by-step workflows promoted from experience |
| Episodic memory | **Journal** | Session logs — one per world per session |

Memory survives across worlds. An agent that worked on your codebase last week remembers it today.

## Evolution

Agents evolve through three mechanisms:

- **Dream** (`spwn agent dream neo`) — Analyzes experience from the journal, discovers patterns, promotes successful strategies to playbooks. Failed ones are discarded. Natural selection for behavior.
- **Sleep** (`spwn agent sleep neo`) — Graceful shutdown — saves state, archives stale files, prunes old sessions. Raw experience consolidates into durable knowledge.
- **Fork** (`spwn agent fork neo neo-v2`) — Clones an agent with everything it knows. Run copies in different environments, keep the branch that performs best.

> *"Every task leaves a trace. Every trace becomes knowledge. Every knowledge shapes the next decision."*

## Use cases

### Compose once, spawn many worlds

```bash
spwn agent new neo
spwn agent add neo --tool @spwn/node --skill refactoring --profile pragmatic-dev

spwn up --agent neo -w ./project-a
spwn up --agent neo -w ./project-b    # same agent, different world
```

### Team with a chief

```bash
spwn up --agent morpheus --agent neo --agent trinity -w ./acme-api
spwn agent send neo "Implement Stripe webhooks" --from morpheus
spwn agent send trinity "Write tests for webhooks" --from morpheus
```

### Solo developer

```bash
spwn up --agent neo -w ./my-app
spwn talk neo "Refactor the auth module to use sessions"
# neo works on it, remembers the codebase next time
```

### Multi-runtime

Runtimes are declared per-agent in `agent.yaml`, not at spawn time:

```yaml
# ~/.spwn/agents/neo/agent.yaml
runtime: claude-code
```

```bash
spwn up --agent neo -w .       # uses the runtime declared in neo's agent.yaml
spwn up --agent smith -w .     # smith can declare a different runtime
```
