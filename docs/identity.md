# Agent Identity

An agent's identity is a directory of markdown files — human-readable, version-controllable, no database.

## Profile structure

```
~/.spwn/agents/neo/
├── profile.yaml              # role, engine, identity, requires, delegation
├── identity/                 # who the agent is
│   ├── persona.md            # role, style, preferences
│   ├── purpose.md            # mission and goals
│   └── traits.md             # values and behavioral traits
├── skills/                   # what the agent can do
├── memory/                   # what the agent knows and remembers
│   ├── knowledge/            # facts about the codebase
│   ├── playbooks/            # step-by-step workflows
│   └── journal/              # session logs
├── sessions/                 # active session state
└── bonds.md                  # relationships with other agents
```

## Identity layers

| Biology | Layer | What it stores |
|---------|-------|----------------|
| Personality | **Persona** | Who I am — role, style, preferences |
| Core values | **Traits** | What I believe — values and behavioral traits |
| Mission | **Purpose** | Why I exist — mission and goals |
| Skills | **Skills** | What I can do — procedures, checklists |
| Semantic memory | **Knowledge** | What I know — facts about the codebase |
| Procedural memory | **Playbooks** | How I do things — step-by-step workflows |
| Episodic memory | **Journal** | What happened to me — session logs |
| Working memory | **Sessions** | Active session state |
| Social | **Bonds** | Who I know — relationships with other agents |

## Evolution

Agents evolve through three mechanisms:

- **Dream** (`spwn agent dream <name>`) — Analyzes experience, discovers patterns, promotes successful strategies to playbooks. Failed ones are discarded. Natural selection for behavior.
- **Sleep** (`spwn agent sleep <name>`) — Graceful shutdown — saves state, archives stale files, prunes old sessions. Raw experience consolidates into durable knowledge.
- **Fork** (`spwn agent fork <src> <dst>`) — Clones an agent. Run copies in different environments, keep the branch that performs best.

> *"Every task leaves a trace. Every trace becomes knowledge. Every knowledge shapes the next decision."*

## Use cases

### Team with a leader

```bash
spwn up --leader morpheus --agent neo --agent trinity -w ./acme-api
spwn msg send neo --from morpheus "Implement Stripe webhooks"
spwn msg send trinity --from morpheus "Write tests for webhooks"
```

### Organization-wide governance

```yaml
# org.yaml
governance:
  max-worlds: 10
  max-agents-per-world: 8
  allowed-providers: [anthropic, openai]
  cost-limit: $50/day
```

### Solo developer

```bash
spwn up --agent neo -w ./my-app
spwn agent talk neo "Refactor the auth module to use sessions"
# neo works on it, remembers the codebase next time
```

### Multi-runtime

```bash
spwn up --agent neo --runtime pi -w .           # Pi runtime
spwn up --agent smith --runtime aider -w .       # Aider for code review
spwn up --agent oracle --runtime codex -w .      # OpenAI Codex
```
