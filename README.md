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

spwn creates isolated Docker worlds for AI agents. Each world has physics (what is physically possible), and each agent has a Mind (persistent identity that survives across worlds).

```
Organization (org.yaml)          governance, defaults, shared skills
  └── World                      isolated workspace with physics
       ├── Governor              leads, delegates to citizens
       ├── Citizens              persistent workers with memory
       └── NPCs                  fire-and-forget tasks
```

The agent's Mind persists. When the world is destroyed, the agent survives. Next time it runs, it remembers everything.

---

## Quick Start

```bash
# Install
curl -fsSL https://spwn.sh/install.sh | bash

# First-time setup
spwn init

# Create an agent
spwn agent init neo

# Spawn a world with the agent inside
spwn world --agent neo -w ./my-project --detach
# → w-default-84721

# Talk to the agent
spwn agent talk neo "What is this project?"
# → neo analyzes the workspace and responds

# Check the environment
spwn status
```

A Docker container is created. The agent's persistent Mind is mounted inside. The runtime (Claude Code by default) is spawned with full shell access. The agent reads its briefing, understands its role, and starts working.

---

## Key Features

**Persistent Identity** — Agents have a Mind: personas, skills, knowledge, playbooks, journal, sessions. It survives across worlds and runtimes. An agent that worked on your codebase last week remembers it today.

**Physics-Based Security** — No network interface? HTTP doesn't exist. Not "forbidden" — physically impossible. You can't prompt-inject a missing binary. You can't social-engineer a network stack that was never installed.

**Pluggable Everything** — 6 runtime adapters (Claude Code, Pi, Codex, OpenCode, Gemini, Aider) + 3 claw adapters. Swap any piece. The core never changes.

**Agent Collaboration** — Governors delegate to citizens via an inbox. Agents message each other, check inboxes, report back. Multi-agent workflows with clear hierarchy.

**Declarative Configuration** — `org.yaml` -> `world.yaml` -> `life.yaml`. Cascading overrides. Version-controllable. Reproducible across machines.

**Full Visibility** — `spwn status` shows every world, agent, and their state. `spwn agent mind` shows what an agent knows. `spwn world inspect` shows physics and resource usage.

---

## Use Cases

### Solo developer

```bash
spwn world --agent neo -w ./my-app
spwn agent talk neo "Refactor the auth module to use sessions"
# neo works on it, remembers the codebase next time
```

### Team with a governor

```bash
spwn world --governor morpheus --agent neo --agent trinity -w ./acme-api
spwn world send w-acme-12345 --from morpheus --to neo "Implement Stripe webhooks"
spwn world send w-acme-12345 --from morpheus --to trinity "Write tests for webhooks"
```

### Multi-runtime

```bash
spwn world --agent neo --runtime pi -w .           # Pi runtime
spwn world --agent smith --runtime aider -w .       # Aider for code review
spwn world --agent oracle --runtime codex -w .      # OpenAI Codex
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
| **Governor** | Leader agent, delegates to citizens | Mind persists |
| **Citizen** | Persistent worker agent | Mind persists |
| **NPC** | One-shot task, no memory | Nothing |
| **Mind** | 6-layer identity (personas, skills, knowledge, playbooks, journal, sessions) | Survives world destruction |
| **Physics** | Constants + laws + elements | Per world config |
| **Gate** | Bridge between host and world | Capability-enforced |

### Mind layers

The Mind is modeled after biological memory. It's a directory of markdown files — human-readable, version-controllable, no database.

| Biology | Mind Layer | What it stores |
|---------|-----------|----------------|
| Personality | **Personas** | Who I am — role, style, preferences |
| Skills | **Skills** | What I can do — procedures, checklists |
| Semantic memory | **Knowledge** | What I know — facts about the codebase |
| Procedural memory | **Playbooks** | How I do things — step-by-step workflows |
| Episodic memory | **Journal** | What happened to me — session logs |
| Working memory | **Sessions** | Active session state |

### Evolution

Agents evolve through three mechanisms:

- **Reflexion** (`spwn agent reflect`) — Reviews journal entries. Strategies that worked get promoted to playbooks. Failed ones are discarded. Natural selection for behavior.
- **Sleep** (`spwn agent sleep`) — Archives stale files, prunes old sessions. Raw experience consolidates into durable knowledge.
- **Forking** (`spwn agent fork`) — Clones a Mind. Run copies in different environments, keep the branch that performs best.

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

### World

```
spwn world                             Spawn a world with default config
spwn world -c node-dev                 Spawn with named config
spwn world --agent neo -w .            Spawn with agent and workspace
spwn world --governor morpheus -w .    Spawn with a governor agent
spwn world list                        List active worlds
spwn world inspect <id>                Show world details, physics, agents
spwn world logs <id>                   Stream agent output
spwn world attach <id>                 Interactive shell into a running world
spwn world destroy <id>                Destroy a world (agent survives)
spwn world snapshot <id>               Save world state
spwn world snapshots                   List snapshots
spwn world restore <snap>              Restore from snapshot
spwn world send <id>                   Send message between agents
spwn world inbox <id>                  Show messages
spwn world watch <id>                  Watch for new messages
```

### Agent

```
spwn agent                             Spawn an agent into a world
spwn agent -n neo                      Spawn named agent
spwn agent -n neo --world w-id         Spawn into specific world
spwn agent --npc "task" --world <id>   Fire ephemeral NPC
spwn agent init [name]                 Create a new agent
spwn agent list                        List all agents
spwn agent inspect <name>              Show agent details and Mind layers
spwn agent talk <name> [message]       Talk to a running agent
spwn agent delete <name>               Remove an agent
spwn agent export <name>               Export as tar.gz
spwn agent fork <src> <dst>            Clone an agent's Mind
spwn agent journal <name>              View session history
spwn agent sessions <name>             View saved sessions
spwn agent mind <name>                 Show Mind directory tree
spwn agent stats <name>                Show agent statistics
spwn agent reflect <name>              Analyze journal, promote playbooks
spwn agent sleep <name>                Archive stale knowledge
```

### System

```
spwn init [name]                       First-time setup
spwn status                            Full environment overview
spwn doctor                            Diagnose environment issues
spwn upgrade                           Upgrade to latest version
```

### Claw (orchestration daemon)

```
spwn claw start                        Start the Claw daemon
spwn claw stop                         Stop the Claw daemon
spwn claw status                       Show status, channels, active worlds
spwn claw connect <channel>            Connect to a messaging channel
```

### Observatory and Skills

```
spwn observatory start                 Start the visual dashboard
spwn observatory open                  Open dashboard in browser
spwn skill list                        List available skills
spwn skill install <skill>             Install a skill
spwn skill remove <skill>              Remove a skill
```

### Global flags

```
--json                                 Output as JSON
-q, --quiet                            Suppress non-essential output
-v, --verbose                          Show debug information
--version                              Show version
```

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
zeroclaw      debian:bookworm-slim  claw
hermes        debian:bookworm       claw
openclaw      node:20               claw
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
| Memory | How Minds persist | Filesystem (markdown) |
| Store | How state is tracked | JSON file |
| Tool | What agents can do | Built-in + MCP |
| Skill | Reusable capabilities | Local files |

### Project layout

```
spwn/
├── core/                       Domain libraries
│   ├── universe/                 World management, ports & adapters
│   ├── agent/                    Agent lifecycle, mind, evolution
│   ├── gate/                     Host-container bridge
│   ├── messenger/                Inter-agent messaging
│   └── foundation/               Primitives (paths, IDs, constants)
│
├── apps/                       Consumers
│   ├── cli/                      The spwn binary
│   └── observatory/              Visual dashboard
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
| **Claude Code** | Runs on your machine | Isolation, physics-based security, persistent Mind |
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
