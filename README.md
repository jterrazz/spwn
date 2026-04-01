# spwn

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev)
[![Tests](https://img.shields.io/badge/Tests-291-green.svg)]()

**The control plane for AI agents.** Isolated worlds, persistent identity, physics-based security, multi-agent collaboration. One CLI to manage them all.

Everyone is building with AI agents — Claude Code, Codex, Aider, Pi. But there is no structure. Agents forget everything between sessions. Configs are scattered across machines. You have no visibility into what tools are exposed, what an agent learned, or whether your setup is reproducible. spwn brings order to this chaos.

---

## The Problem

- **Your agent forgets everything between sessions.** Context window resets. Knowledge gone. Every conversation starts from scratch.
- **You can't see what tools and skills are exposed.** What can the agent do? What MCP servers are connected? Nobody knows.
- **Your setup isn't reproducible or shareable.** It works on your machine. Good luck onboarding a teammate.
- **You have zero governance over what agents can do.** No cost limits. No resource constraints. No audit trail.

---

## The Solution

spwn creates isolated Docker worlds for AI agents. Each world has physics (what is physically possible), and each agent has a Profile (persistent identity that survives across worlds).

```
Organization (org.yaml)          governance, defaults, shared skills
  └── World                      isolated workspace with physics
       ├── Governor              leads, delegates to citizens
       ├── Citizens              persistent workers with memory
       └── NPCs                  fire-and-forget tasks
```

The agent's identity persists. When the world is destroyed, the agent survives. Next time it runs, it remembers everything.

---

## Install

```bash
# One-liner (downloads latest release to ~/.local/bin)
curl -fsSL https://spwn.sh/install.sh | bash

# Or build from source
git clone https://github.com/jterrazz/spwn.git && cd spwn
make install
```

Both methods install to `~/.local/bin` and auto-add it to your PATH if needed. Override the install directory with `INSTALL_DIR`:

```bash
make install INSTALL_DIR=/usr/local/bin    # system-wide (needs sudo)
make install INSTALL_DIR=~/.bin            # custom location
```

Uninstall with `make uninstall` or `rm ~/.local/bin/spwn`.

**Requirements:** Go 1.25+, Docker.

---

## Quick Start

```bash
# Create an agent
spwn agent new neo

# Spawn a world with the agent inside
spwn up --agent neo -w ./my-project --detach
# → w-default-84721

# Talk to the agent
spwn agent talk neo "What is this project?"
# → neo analyzes the workspace and responds

# Check the environment
spwn ls
```

A Docker container is created. The agent's persistent profile is mounted inside. The runtime (Claude Code by default) is spawned with full shell access. The agent reads its briefing, understands its role, and starts working.

---

## Key Features

**Persistent Identity** — Agents have a Profile: persona, traits, purpose, bonds, skills, knowledge, playbooks, journal, sessions. It survives across worlds and runtimes. An agent that worked on your codebase last week remembers it today.

**Physics-Based Security** — No network interface? HTTP doesn't exist. Not "forbidden" — physically impossible. You can't prompt-inject a missing binary. You can't social-engineer a network stack that was never installed.

**Pluggable Everything** — Claude Code runtime + Hermes architect adapter today. Pi, Codex, OpenCode, Gemini, Aider runtimes planned. Swap any piece. The core never changes.

**Agent Collaboration** — Governors delegate to citizens via an inbox. Agents message each other, check inboxes, report back. Multi-agent workflows with clear hierarchy.

**Declarative Configuration** — `org.yaml` -> `world.yaml` -> `profile.yaml`. Cascading overrides. Version-controllable. Reproducible across machines.

**Full Visibility** — `spwn ls` shows every world, agent, and their state. `spwn profile <name>` shows the full character sheet. `spwn inspect <id>` shows physics and resource usage.

---

## Use Cases

### Solo developer

```bash
spwn up --agent neo -w ./my-app
spwn agent talk neo "Refactor the auth module to use sessions"
# neo works on it, remembers the codebase next time
```

### Team with a governor

```bash
spwn up --governor morpheus --agent neo --agent trinity -w ./acme-api
spwn msg send neo --from morpheus "Implement Stripe webhooks"
spwn msg send trinity --from morpheus "Write tests for webhooks"
```

### Multi-runtime

```bash
spwn up --agent neo --runtime pi -w .           # Pi runtime
spwn up --agent smith --runtime aider -w .       # Aider for code review
spwn up --agent oracle --runtime codex -w .      # OpenAI Codex
```

### Organization-wide governance

```yaml
# org.yaml
governance:
  max-worlds: 10
  max-citizens-per-world: 8
  allowed-providers: [anthropic, openai]
  cost-limit: $50/day
```

---

## Concepts

| Concept | What | Persists |
|---------|------|----------|
| **Organization** | Governance, defaults, shared skills | `org.yaml` |
| **World** | Isolated workspace with physics | Docker container |
| **Governor** | Leader agent, delegates to citizens | Profile persists |
| **Citizen** | Persistent worker agent | Profile persists |
| **NPC** | One-shot task, no memory | Nothing |
| **Profile** | Full character sheet (persona, traits, purpose, bonds, skills, memory) | Survives world destruction |
| **Physics** | Constants + laws + elements | Per world config |
| **Gate** | Bridge between host and world | Capability-enforced |

### Agent directory

The agent's identity is a directory of markdown files — human-readable, version-controllable, no database.

```
~/.spwn/agents/neo/
├── profile.yaml              # tier, engine, identity, requires, delegation
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

### Evolution

Agents evolve through three mechanisms:

- **Reflexion** (`spwn agent reflect <name>`) — Reviews journal entries. Strategies that worked get promoted to playbooks. Failed ones are discarded. Natural selection for behavior.
- **Sleep** (`spwn agent sleep <name>`) — Archives stale files, prunes old sessions. Raw experience consolidates into durable knowledge.
- **Forking** (`spwn agent fork`) — Clones an agent. Run copies in different environments, keep the branch that performs best.

### Physics

The world manifest defines what is physically possible:

```yaml
physics:
  constants:
    cpu: 2
    memory: 1GB
    timeout: 30m
  laws:
    network: none
  elements:
    - @unix        # bash, coreutils, grep, sed, awk, find
    - @git
    - @node
    - jq
    gate:
      - source: mcp/slack
        as: slack-send
        capabilities: [send]
```

If `curl` is not in the element list, it does not exist. Elements are verified at world creation and exposed in the agent's `/world/faculties.md`.

---

## CLI Reference

### World Operations (top-level)

```
spwn up --agent neo -w .              Spawn a world with an agent
spwn ls                               List active worlds
spwn inspect <id>                     Show world details and physics
spwn down <id>                        Destroy a world (agent survives)
spwn logs <id>                        Stream agent output
spwn attach <id>                      Interactive shell
```

### Agent Management

> Profile is the passport. Agent is the person.

```
spwn agent new <name>                 Create a new agent
spwn agent ls                         List all agents
spwn agent rm <name>                  Remove an agent
spwn agent talk <name> [message]      Talk to a running agent
spwn agent reflect <name>             Promote journal patterns to playbooks
spwn agent sleep <name>               Consolidate and prune memory
spwn agent fork <src> <dst>           Clone an agent
spwn agent export <name>              Export agent as tar.gz
spwn agent import <file>              Import agent from tar.gz
```

### Profile (character sheet — the passport, not the person)

```
spwn profile <name>                   Show full character sheet
spwn profile <name> purpose           Show/edit purpose
spwn profile <name> traits            Show/edit traits
spwn profile <name> persona           Show/edit persona
spwn profile <name> bonds             Show/edit bonds
spwn profile <name> skills            List skills
spwn profile <name> playbooks         List playbooks
spwn profile <name> knowledge         List knowledge
spwn profile <name> journal           Session history
spwn profile <name> sessions          Active sessions
spwn profile <name> edit              Edit profile.yaml
spwn profile <name> tier              Show/set agent tier
spwn profile <name> engine            Show/set runtime engine
```

### Messaging

```
spwn msg send <agent> --from <sender> "msg"   Send message to agent
spwn msg inbox <agent>                         Show agent inbox
spwn msg watch <agent>                         Watch for new messages
```

### Snapshots

```
spwn snap save <id>                   Save world state
spwn snap ls                          List snapshots
spwn snap restore <snap>              Restore from snapshot
spwn snap rm <snap>                   Remove a snapshot
```

### Platform & System

```
spwn architect start|stop|status|connect  Your always-on world builder
spwn dash start|open                 Visual dashboard
spwn get install|ls|search|rm        Install from the marketplace
spwn auth login|logout|token          Authentication
```

Use `spwn <command> --help` for full details on any command.

---

## Runtime Adapters

spwn treats agent runtimes as swappable adapters. The container-side Gate speaks [ACP](https://github.com/agentclientprotocol/agent-client-protocol), so adding a new runtime is a container image change.

```
RUNTIME       BASE IMAGE            TYPE
claude-code   node:20               runtime
pi            node:20               runtime
codex         node:20               runtime
opencode      debian:bookworm-slim  runtime
gemini        node:20               runtime
aider         python:3.12-slim      runtime
zeroclaw      debian:bookworm-slim  architect Planned
hermes        debian:bookworm       architect Available
openclaw      node:20               architect Planned
```

---

## Architecture

Multi-module Go monorepo with Ports and Adapters architecture. 8 port interfaces, each with swappable adapters:

| Port | What it abstracts | Default adapter |
|------|-------------------|-----------------|
| Runtime | How agents think | Claude Code (ACP) |
| Provider | Which LLM | Anthropic |
| Backend | Where worlds run | Docker |
| Channel | External communication | CLI |
| Memory | How profiles persist | Filesystem (markdown) |
| Store | How state is tracked | JSON file |
| Tool | What agents can do | Built-in + MCP |
| Skill | Reusable capabilities | Local files |

### Project layout

```
spwn/
├── core/                       Domain libraries
│   ├── universe/                 World management, ports & adapters
│   ├── agent/                    Agent lifecycle, profile, evolution
│   ├── gate/                     Host-container bridge
│   ├── messenger/                Inter-agent messaging
│   └── foundation/               Primitives (paths, IDs, constants)
│
├── apps/                       Consumers
│   ├── cli/                      The spwn binary
│   └── dash/                     Visual dashboard
│
└── platform/                   Build infrastructure
    ├── images/                   Docker images (base, test)
    ├── gate-runtime/             Container-side Rust gate
    └── fixtures/                 Test fixtures
```

### Dependency graph

```
apps/cli ──> core/universe, core/agent, core/gate, core/messenger, core/foundation
core/universe ──> core/agent, core/gate, core/foundation
core/agent ──> core/foundation
core/gate ──> core/foundation
core/messenger ──> core/foundation
```

---

## Comparison

| | Approach | What spwn adds |
|---|---------|----------------|
| **MCP** | Exposes tools one at a time | Full shell — N! compositions, not N tools |
| **LangChain / CrewAI** | Chains function calls | Emergent behavior, not deterministic chains |
| **E2B** | Cloud sandboxes | Self-hosted, persistent identity, evolution |
| **Claude Code** | Runs on your machine | Isolation, physics-based security, persistent profile |
| **Docker** | Container runtime | Agent lifecycle, identity, Gate, evolution |

spwn is not a competitor to Claude Code — it is the complement. Claude Code is the intelligence. spwn is the world to be intelligent in.

---

## Links

- **Website:** [spwn.sh](https://spwn.sh)
- **Docs:** [spwn.sh/docs](https://spwn.sh/docs)
- **Blueprint:** [github.com/jterrazz/spwn-wiki](https://github.com/jterrazz/spwn-wiki)
- **Contributing:** [CONTRIBUTING.md](CONTRIBUTING.md)

## License

MIT
