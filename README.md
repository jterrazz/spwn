# Universe

**Create realities for things that can think.**

---

Intelligence is solved. Models like Claude can reason, code, debug, and plan. But intelligence without a world is a brain in a jar — brilliant, yet unable to *do* anything. No environment. No memory. No yesterday.

Universe builds the missing half: not a better toolchain, but a **reality** — a world with physics, inhabited by a conscious entity that remembers, reflects, and evolves.

```bash
universe spawn --agent leonardo --workspace ./my-project
```

One command. A world is born from the origin declared in `universe.yaml`. The agent's Mind — its persistent identity — is mounted inside. The Gate spawns Claude Code via [ACP](https://github.com/agentclientprotocol/agent-client-protocol). When the task ends, the world is destroyed. The Mind persists. The agent remembers.

## Why

Most agent frameworks give an AI a list of tools: "call this function to read a file," "call this to search the web." The agent picks a tool, gets a result, picks another. It's like giving a developer a calculator and asking them to build software.

Meanwhile, a real developer at a terminal is orders of magnitude more powerful. Not because of any single command, but because Unix is **composable**: `grep | sort | uniq` — fifty years of tools, all piping into each other. N commands give N! compositions.

Claude Code proved this: give an LLM a real terminal and it becomes dramatically more capable than a chatbot with tools. The difference isn't the model — it's the environment.

Universe gives every agent that environment, then adds what Claude Code alone can't provide: **isolation**, **persistent identity**, and **evolution**.

## The World Has Physics

Gravity isn't a rule you follow — it's a fact you can't escape. A Universe works the same way.

```yaml
# universe.yaml — what reality looks like
origin: ubuntu:24.04

constants:
  cpu: 2
  memory: 1GB
  timeout: 30m

laws:
  network: none

elements:
  require: [git, node, npm, jq]

interactions:
  - source: mcp/slack
    as: slack-send
    capabilities: [send]
```

The origin defines the agent's reality in two parts:

- **Physics** — laws of nature. No network interface means HTTP is *physically impossible*. Memory and CPU are finite. Filesystem structure is imposed.
- **Elements** — manufactured objects. `git`, `curl`, `python3` are screwdrivers and hammers. No `curl` means no one built an HTTP client. No `apt` means no new tools can ever be manufactured.

You can't prompt-inject a missing network interface. You can't jailbreak a tool that was never installed. Security isn't a rule — it's a fact of the world.

> No chains on the agent. Chains in the physics.

## The World Has Life

Physics alone is a dead world. Life requires a conscious entity with identity, memory, and will.

An agent is backed by a **Mind** — a persistent blueprint that mirrors biological consciousness:

| What | Mind Layer | Holds |
| --- | --- | --- |
| Personality | **Personas** | Who I am — identity, tendencies, preferences |
| Capabilities | **Skills** | What I can do — invocable, composable |
| Semantic memory | **Knowledge** | What I know — distilled from experience |
| Procedural memory | **Playbooks** | How I do things — earned, not taught |
| Episodic memory | **Journal** | What happened to me — append-only |
| Senses | **Interactions** | How I perceive beyond my world |

```yaml
# mind.yaml — who the agent is
name: leonardo

personas:
  - personas/backend-engineer.md

skills:
  - skills/deploy.md
  - skills/debug.md

knowledge:
  - knowledge/domain.md
```

The agent has unconstrained will. No guardrails. Dangerous actions aren't forbidden — they're physically impossible. You don't tell a creature "don't fly" in a world without atmosphere.

## Life Evolves

Model weights never change. What evolves is the Mind.

**Reflexion** — after each session, the agent reviews what worked and what didn't. Successful strategies get promoted to playbooks. Failed ones are discarded. Natural selection for behavior.

**Sleep** — not rest, but reorganization. The agent consolidates raw experience into durable knowledge, prunes stale strategies, resolves contradictions, and updates its self-model. Where reflexion is *"what did I learn just now?"*, sleep is *"who am I now?"*

**Forking** — clone an agent, run copies in different environments, keep the branch that performs best, merge the winners back. Population-level adaptation.

## Architecture

```
Substrate (your machine)
  └── Architect                    Orchestrator
       └── Universe                A contained world
            ├── Agent              Lives, thinks, acts
            │    └── Mind          Persistent blueprint
            ├── Gate               Bridge to Substrate
            └── physics.md         Laws of this reality
```

The Architect reads `universe.yaml`, provisions a container (Docker) or microVM (Firecracker), mounts the Mind and workspace, generates `physics.md`, and the container-side Gate spawns the agent via ACP.

```
┌──────────────────────────────────────────────────┐
│  CLI (cobra)                                     │
│  spawn · list · inspect · destroy · agent        │
└──────────────┬───────────────────────────────────┘
               │
      ┌────────▼────────┐
      │    Architect     │
      └────────┬────────┘
               │
  ┌────────────┼────────────┬──────────────┐
  │            │            │              │
┌─▼──────┐ ┌──▼─────┐ ┌────▼────┐ ┌──────▼───┐
│Backend │ │Manifest│ │ Mind    │ │ Physics  │
│(Docker)│ │Resolver│ │Manager  │ │Generator │
└────┬───┘ └────────┘ └─────────┘ └──────────┘
     │
┌────▼──────────────────────────────────────────┐
│  Universe (container / microVM)               │
│                                               │
│  Container-side Gate                          │
│  └── ACP Client → stdio → Agent CLI          │
│       (Claude Code, Codex, Gemini, etc.)      │
│                                               │
│  /mind       Agent blueprint                  │
│  /workspace  Project files                    │
│  /gate       Unix socket + interaction bridge │
│  /universe   physics.md                       │
└───────────────────────────────────────────────┘
```

## Quick Start

```bash
# Install (requires Go 1.23+ and Docker)
go install github.com/jterrazz/universe/cmd/universe@latest

# Create an agent
universe agent init leonardo

# Spawn a world with the agent inside
universe spawn --agent leonardo --workspace ./my-project
```

## CLI

```bash
# The World
universe spawn              # Create a universe (the Big Bang)
universe list               # List all universes
universe inspect <id>       # Show universe details and physics
universe logs <id>          # Stream agent output
universe attach <id>        # Interactive session into a universe
universe destroy <id>       # Destroy a universe
universe init               # Generate universe.yaml + mind.yaml

# Life
universe agent spawn <id>   # Bring an agent to life inside a universe
universe agent list          # List all agents on this Substrate
universe agent inspect <id>  # Show Mind layers and journal
universe agent init <name>   # Scaffold a new Mind
universe agent export <name> # Export a Mind as tar.gz
```

Every universe gets a human-readable ID: `u-bright-comet`, `u-calm-nebula` — two words, easy to type, easy to remember.

## How It Compares

| Solution | Approach | What Universe adds |
| --- | --- | --- |
| **MCP** | Exposes individual tools | Full shell — N! compositions instead of N tools |
| **LangChain / CrewAI** | Chains tool calls | Combinatorial freedom, not deterministic chains |
| **E2B** | Cloud sandboxes | Self-hosted, persistent identity, evolution |
| **Claude Code** | Operates on your machine | Isolation, persistent Mind, physics-based security |
| **Docker** | Container runtime | Mind, Gate, agent lifecycle, evolution |

## Technology

| Layer | Technology | Why |
| --- | --- | --- |
| Core | Go | Native Docker SDK, single binary, goroutines |
| CLI | Cobra | Industry standard for Go CLIs |
| Agent runtime | Claude Code CLI | Best Unix-native agent, reads Mind as markdown |
| Agent protocol | [ACP](https://github.com/agentclientprotocol/agent-client-protocol) | 34+ agent CLIs supported, session management for free |
| Isolation | Docker / Firecracker | Containers for dev, microVMs for production |
| Bridge | Gate (two-sided) | Host: mounts + interactions. Container: ACP client |
| Identity | Mind framework | Persistent memory across worlds |

## Project Structure

```
cmd/universe/           CLI entry point (cobra commands)
internal/
├── architect/          Orchestrator — lifecycle management
├── agent/              Agent configuration and ACP spawning
├── backend/            Backend interface + Docker/Firecracker adapters
├── config/             Type definitions and manifest types
├── gate/               Host-side Gate: interaction bridge, session relay
├── journal/            Auto-generated spawn logs (markdown)
├── manifest/           universe.yaml + mind.yaml parsing
├── mind/               Mind directory management
├── physics/            physics.md generation from container introspection
└── session/            Session persistence (JSON per agent+universe)
container/              Container-side Gate binary (ACP client + crash recovery)
```

## Current Status

Pre-release. The architecture is designed, the wiki is complete, implementation is beginning. See the [Wiki](https://github.com/jterrazz/universe-wiki) for deep dives into every design decision.

## Learn More

| | |
| --- | --- |
| [Vision](https://github.com/jterrazz/universe-wiki/blob/main/01-overview/vision.md) | The full case for Unix-native agents |
| [Philosophy](https://github.com/jterrazz/universe-wiki/blob/main/01-overview/philosophy.md) | Security as physics, evolution as design |
| [Mind Framework](https://github.com/jterrazz/universe-wiki/blob/main/02-architecture/mind-framework.md) | The six layers of identity |
| [Roadmap](https://github.com/jterrazz/universe-wiki/blob/main/01-overview/roadmap.md) | Seven phases to an ecosystem for artificial life |
| [Decisions](https://github.com/jterrazz/universe-wiki/blob/main/05-decisions/) | Every architectural choice, recorded |

## License

MIT
