# Agent Identity

An agent's identity is a directory of markdown files — human-readable, version-controllable, no database.

## Mind structure

The Mind is organized in 5 layers, each a directory under `~/.spwn/agents/{Name}/`:

```
~/.spwn/agents/Neo/
├── profile.yaml              # team, role, engine, delegation rules
├── core/                     # who the agent is
│   └── persona.md            # role, style, preferences, purpose, behavior
├── skills/                   # what the agent can do — procedures, checklists
├── knowledge/                # what the agent knows — facts about the codebase
├── playbooks/                # how the agent works — step-by-step workflows learned from experience
└── journal/                  # what happened — session logs per world
```

## The 5 layers

| Biology | Layer | What it stores |
|---------|-------|----------------|
| Personality | **Core** | Who I am — role, style, preferences, purpose, values, behavior |
| Skills | **Skills** | What I can do — procedures, checklists |
| Semantic memory | **Knowledge** | What I know — facts about the codebase, the team, the domain |
| Procedural memory | **Playbooks** | How I do things — step-by-step workflows promoted from experience |
| Episodic memory | **Journal** | What happened to me — session logs, one per world per session |

The `profile.yaml` at root holds metadata: team assignment, runtime engine, delegation preferences. The 5 layer directories hold the actual content — all markdown, all human-editable.

## Evolution

Agents evolve through three mechanisms:

- **Dream** (`spwn agent dream <name>`) — Analyzes experience from the journal, discovers patterns, promotes successful strategies to playbooks. Failed ones are discarded. Natural selection for behavior.
- **Sleep** (`spwn agent sleep <name>`) — Graceful shutdown — saves state, archives stale files, prunes old sessions. Raw experience consolidates into durable knowledge.
- **Fork** (`spwn agent fork <src> <dst>`) — Clones an agent's entire Mind. Run copies in different environments, keep the branch that performs best.

> *"Every task leaves a trace. Every trace becomes knowledge. Every knowledge shapes the next decision."*

## Use cases

### Team with a leader

```bash
spwn up --leader morpheus --agent neo --agent trinity -w ./acme-api
spwn msg send neo --from morpheus "Implement Stripe webhooks"
spwn msg send trinity --from morpheus "Write tests for webhooks"
```

### Solo developer

```bash
spwn up --agent neo -w ./my-app
spwn agent talk neo "Refactor the auth module to use sessions"
# neo works on it, remembers the codebase next time
```

### Multi-runtime

```bash
spwn up --agent neo --runtime claude-code -w .   # Anthropic (default)
spwn up --agent smith --runtime codex -w .        # OpenAI
spwn up --agent oracle --runtime aider -w .       # Open source
```
