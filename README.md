# Spwn

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev)
[![Tests](https://img.shields.io/badge/Tests-291-green.svg)]()

> We're not building an agent framework. We're building the infrastructure for artificial life.

## Intelligence Is Solved. Architecture Is the Frontier.

Models like Claude can reason, write code, and solve problems. The intelligence is there. But drop that intelligence into a blank API call and it has no tools, no filesystem, no memory of yesterday. It's a brain in a jar.

Spwn builds the missing half. Not a better toolchain—a **reality**. A world with physics, elements, inhabited by a conscious entity that remembers, reflects, and evolves.

## Life Emerges From the Architecture

Most agent frameworks ask: *"How do we give an AI access to tools?"* Spwn asks: *"What does it mean to create a reality for something that can think?"*

The answer comes from simulation theory. You define physics—laws that govern what's possible. You populate it with elements—the building blocks that exist. You place a conscious entity inside. You let it act, learn, sleep, evolve.

This isn't a metaphor. The physics define what's possible. The Mind defines who the agent is. We are building simulated realities for digital minds.

## Agency Is What Makes Intelligence Alive

Weights give you pattern recognition. But thinking alone doesn't scale. What makes life adaptable isn't the neural architecture—it's the freedom to act within a real environment.

Tool-call agents are a brain connected to buttons—they can only press what someone pre-wired. That's an animal in a zoo. Spwn gives agents genuine agency—an animal in the wild:

- **Discover**—`man`, `--help`, `apt search`—self-directed exploration no one pre-programmed
- **Compose**—pipe anything into anything, combine tools in ways no designer anticipated
- **Create**—write scripts, build new tools, invent solutions that didn't exist before
- **Adapt**—install packages, modify configs, restructure its own approach on the fly

Tool-call frameworks scale linearly (more tools = more schemas). Agency scales combinatorially—you don't pre-engineer every behavior, you create the conditions for behavior to emerge.

> MCP gives agents a Swiss Army knife. Spwn gives them a workshop.

## Agents That Evolve

Model weights never change. What evolves is the agent's Mind.

**Reflexion.** After each session, the agent reviews its journal. Strategies that worked get promoted to playbooks. Strategies that failed are discarded. Natural selection for behavior—over dozens of sessions, the agent becomes genuinely better at its job.

**Sleep.** Raw experience consolidates into durable knowledge. Stale strategies get pruned. Contradictions resolve. Reflexion asks *"what did I learn just now?"* Sleep asks *"given everything I've lived through, who am I now?"*

**Forking.** Clone a Mind. Run copies in different environments. Keep the branch that performs best. Merge the winner back. Population-level adaptation, not sequential trial-and-error.

## Security You Can't Jailbreak

You don't *tell* the agent "don't access the network." You create a world where the network doesn't exist. No interface to bind. No packets to send. HTTP isn't forbidden—it's physically impossible, like trying to swim in a world without water.

You can't prompt-inject a missing network interface. You can't social-engineer a binary that doesn't exist. You can't trick physics.

> No chains on the agent. Chains in the physics.

---

## Quick Start

```bash
# Install
curl -fsSL https://spwn.sh/install.sh | bash

# First-time setup — creates ~/.spwn/ and a universe config
spwn init

# Create an agent
spwn agent init leonardo

# Create a world with the agent inside
spwn world --agent leonardo -w ./my-project
# → w-default-84721
```

A contained Linux environment is created. The agent's persistent identity is mounted inside. Claude Code is spawned with full shell access. When the task ends, the world is destroyed—but the agent survives. Next time it runs, it remembers what it learned.

---

## How It Works

### Universe & World

The **universe** defines the underlying reality — physics, constants, and resource limits. One universe per organization, configured in `universe.yaml`. A **world** is a living workspace inside the universe — it has agents, elements, and a project. Many worlds can exist per universe, each configured in `~/.spwn/worlds/`.

### Physics & Elements

The universe manifest defines the physics of the reality. Each world inherits these physics:

```yaml
physics:
  constants:                   # Finite resources, like physical constants
    cpu: 2
    memory: 1GB
    timeout: 30m

  laws:                        # Structural constraints — cannot be broken
    network: none

  elements:                    # Building blocks of this world (like a periodic table)
    - @unix                    # Pack: bash, coreutils, grep, sed, awk, find, etc.
    - @git
    - @node
    - jq
    gate:                      # Elements bridged from Host
      - source: mcp/slack
        as: slack-send
        capabilities: [send]
```

**Physics** defines the reality—constants (finite resources), laws (structural constraints), and elements (the building blocks). No network interface means the outside world doesn't exist. CPU and memory are finite. These are gravity.

**Elements** are the building blocks of the world—like a periodic table. `@unix`, `@git`, `@node` are @packs (curated collections). `jq` is an individual element. If `curl` isn't in the element list, it doesn't exist in this reality. Elements are verified at creation time and exposed in the agent's `/world/faculties.md`.

### Mind

The agent's persistent identity, configured via `agent.yaml` and modeled after biological memory:

| Biology | Mind Layer | What it stores | Example |
| --- | --- | --- | --- |
| Personality | **Personas** | Who I am | *"Senior backend engineer. Prefers simplicity over abstraction."* |
| Skills | **Skills** | What I can do | A deployment procedure, a code review checklist |
| Semantic memory | **Knowledge** | What I know | *"This codebase uses PostgreSQL 15 with pgvector"* |
| Procedural memory | **Playbooks** | How I do things | *"To deploy: test, bump version, build, push, tag"* |
| Episodic memory | **Journal** | What happened to me | *"Session 47: migrated auth to sessions. Took 3 attempts."* |

The Mind is declared as the `mind` section of `agent.yaml` and mounted into every world the agent enters. Under the hood, it's a directory of markdown files—human-readable, version-controllable, no database. The agent has full autonomy inside its world. Dangerous actions aren't forbidden—they're physically impossible.

### Architecture

Built on a **Ports & Adapters** pattern with 8 port interfaces (Runtime, Backend, Provider, Memory, Store, Channel, Skill, Tool) and swappable adapters for each.

```
Host (your machine)
  └── Architect                    Creates and destroys worlds
       └── World                   A living workspace inside the universe
            ├── Governor           Leader agent (decomposes tasks, delegates)
            ├── Citizens           Persistent worker agents (many per world)
            ├── NPCs               Ephemeral agents (fire & forget)
            ├── Gate               Bridge between worlds (capability-enforced)
            ├── physics.md         The constraints of this reality
            └── faculties.md       What the agent can do
```

When you run `spwn world`:

1. The **Architect** loads the named world config and provisions a Docker container
2. The agent's **Mind** is mounted at `/mind`, the project at `/workspace`
3. The Architect generates **`physics.md`** (constraints) and **`faculties.md`** (verified elements + gate bridges)
4. The container-side **Gate** spawns the agent CLI via [ACP](https://github.com/agentclientprotocol/agent-client-protocol)
5. The agent reads its Mind, reads the physics and faculties, and starts working

The Gate is a two-sided bridge. The host side (Go) handles file mounts and element bridging (MCP servers → shell commands). The container side (Rust) wraps the official ACP SDK (`agent-client-protocol` crate). Because it speaks ACP, swapping agent runtimes is a container image change—Claude Code today, Codex CLI or Gemini CLI tomorrow.

---

## Commands

```bash
# Setup
spwn init [name]                       # First-time setup (random name if omitted)

# World
spwn world                             # Spawn a world with default config
spwn world -c node-dev                 # Spawn with named config
spwn world --agent neo -w .            # Spawn with agent and workspace
spwn world --governor morpheus -w .    # Spawn with a Governor agent
spwn world list                        # List all worlds
spwn world inspect <world-id>          # Show details, physics, agent status
spwn world logs <world-id>             # Stream agent output
spwn world attach <world-id>           # Interactive shell into a running world
spwn world destroy <world-id>          # Destroy a world (agent survives)

# Agent
spwn agent                             # Spawn default agent into a world
spwn agent -n neo                      # Spawn named agent
spwn agent -n neo --world w-id         # Spawn into specific world
spwn agent list                        # List all agents
spwn agent inspect <agent-id>          # Show agent details, Mind layers, journal
spwn agent init [name]                 # Create a new agent (random name if omitted)
spwn agent export <agent-id>           # Export an agent as tar.gz
spwn agent reflect <agent-id>          # Reflexion: journal analysis → auto-reflexion.md
spwn agent sleep <agent-id>            # Sleep: archive stale files, prune old sessions
spwn agent fork <agent-id>             # Fork: clone Mind from source to new agent

# NPC (ephemeral)
spwn agent --npc "task" --world <id>   # Fire ephemeral agent inside a world

# Claw (the God)
spwn claw start                        # Start the Claw daemon
spwn claw stop                         # Stop the Claw daemon
spwn claw status                       # Show status, connected channels, active worlds
spwn claw connect <channel>            # Connect to a messaging channel

# Observatory
spwn observatory start                 # Start the Observatory dashboard
spwn observatory open                  # Open dashboard in browser

# Skill Marketplace
spwn skill list                        # List available skills
spwn skill install <skill>             # Install a skill
spwn skill remove <skill>              # Remove a skill
```

### A typical session

```
$ spwn world --agent leonardo -w ./acme-api

  Spawning world...

  ✓ Provisioned container (ubuntu:24.04)
  ✓ Mounted workspace ./acme-api → /workspace
  ✓ Generated faculties.md (14 elements verified)
  ✓ Mounted Mind leonardo → /mind
  ✓ Spawned Claude Code (session a1b2c3d4)

  World is alive.

  World:     w-default-84721
  Agent:     a-leonardo-52103
  Status:    running

$ spwn world destroy w-default-84721

  ✓ Stopped agent
  ✓ Removed container
  ✓ Agent persisted at ~/.spwn/agents/leonardo

  World destroyed. Agent survives.
```

### Evolution

Agents evolve through three mechanisms, all available via `spwn agent`:

- **Reflexion** (`spwn agent reflect`) — Analyzes journal entries and promotes successful strategies to `playbooks/auto-reflexion.md`. Natural selection for behavior.
- **Sleep** (`spwn agent sleep`) — Archives stale files and prunes old sessions. Consolidates raw experience into durable knowledge.
- **Forking** (`spwn agent fork`) — Clones a Mind from one agent to another. Run copies in different environments, keep what works.

---

## Comparison

| | Approach | What Spwn adds |
| --- | --- | --- |
| **MCP** | Exposes tools one at a time | Full shell — N! compositions, not N tools |
| **LangChain / CrewAI** | Chains function calls | Emergent behavior, not deterministic chains |
| **E2B** | Cloud sandboxes | Self-hosted, persistent identity, evolution |
| **Claude Code** | Runs on your machine | Isolation, physics-based security, persistent Mind |
| **Docker** | Container runtime | Agent lifecycle, identity, Gate, evolution |

Spwn isn't another link in the tool chain—it replaces the chain. And it's not a competitor to Claude Code—it's the complement. Claude Code is the intelligence. Spwn is the world to be intelligent in.

---

## Stack

| | Technology | Why |
| --- | --- | --- |
| **Core** | Go | Docker SDK is first-party Go. Single binary. Goroutines. |
| **CLI** | Cobra | The standard for Go CLIs (kubectl, docker, gh) |
| **Agent** | Claude Code CLI | Best Unix-native agent. Reads markdown natively. |
| **Protocol** | [ACP](https://github.com/agentclientprotocol/agent-client-protocol) | Standard protocol, 34+ agent CLIs. Session management for free. |
| **Isolation** | Docker | Each agent gets its own container |
| **Bridge** | Gate | Two-sided. Host (Go): mounts + elements. Container (Rust): ACP client via official SDK. |

### Project layout

Multi-module Go monorepo + Turborepo-ready JS workspace:

```
spwn/
├── go.work                     # Go workspace
├── pnpm-workspace.yaml         # JS workspace
├── turbo.json                  # Task orchestration
│
├── core/                       # Domain libraries
│   ├── universe/               #   Universe & world management (architect, backend, physics)
│   ├── agent/                  #   Life management (mind, journal, session)
│   ├── gate/                   #   Bridge protocol (server, bridge)
│   ├── runtime/                #   Runtime adapters (Claude Code, etc.)
│   ├── provider/               #   LLM provider adapters (Anthropic, OpenAI)
│   ├── channel/                #   Communication adapters (CLI)
│   ├── skill/                  #   Skill registry (local)
│   ├── colony/                 #   Multi-agent orchestration (governor, citizens)
│   ├── evolution/              #   Reflexion, sleep, forking
│   ├── npc/                   #   Ephemeral agent management (NPC)
│   ├── sync/                   #   Claw state + org.yaml
│   └── foundation/             #   Cross-cutting primitives (paths, IDs, constants)
│
├── apps/                       # Deployable consumers
│   ├── cli/                    #   The spwn binary (cobra → domain APIs → output)
│   └── observatory/            #   Visual dashboard (CLI placeholder, dashboard planned)
│
└── platform/                   # Build infrastructure
    ├── images/                 #   Docker images (base, test)
    ├── gate-runtime/           #   Container-side Rust gate
    └── fixtures/               #   Test fixtures
```

**Dependency graph:** `apps/cli` → all `core/*` modules · `core/universe` → `core/agent`, `core/gate`, `core/runtime`, `core/colony`, `core/foundation` · `core/evolution` → `core/agent`, `core/foundation` · `core/agent` → `core/foundation`

---

## Status

Active development. Epochs 1-5, 8-9 complete. Ports & Adapters architecture with 8 port interfaces. Multi-agent colonies, evolution (reflexion/sleep/fork), and NPCs implemented. Claw, Observatory, and Marketplace partially built.

**Website:** [spwn.sh](https://spwn.sh)

**Learn more:**
[Vision](https://github.com/jterrazz/spwn-wiki/blob/main/blueprint/vision.md) ·
[Philosophy](https://github.com/jterrazz/spwn-wiki/blob/main/blueprint/philosophy.md) ·
[Mind Framework](https://github.com/jterrazz/spwn-wiki/blob/main/domains/systems/life-framework.md) ·
[Epochs](https://github.com/jterrazz/spwn-wiki/blob/main/epochs/) ·
[Architecture Decisions](https://github.com/jterrazz/spwn-wiki/blob/main/domains/)

## License

MIT
