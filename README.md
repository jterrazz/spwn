# Universe

> We're not building an agent framework. We're building the infrastructure for artificial life.

## Intelligence Is Solved. Architecture Is the Frontier.

Models like Claude can reason, write code, and solve problems. The intelligence is there. But drop that intelligence into a blank API call and it has no tools, no filesystem, no memory of yesterday. It's a brain in a jar.

Universe builds the missing half. Not a better toolchain—a **reality**. A world with physics, evolved technologies, inhabited by a conscious entity that remembers, reflects, and evolves.

## Life Emerges From the Architecture

Most agent frameworks ask: *"How do we give an AI access to tools?"* Universe asks: *"What does it mean to create a reality for something that can think?"*

The answer comes from simulation theory. You define physics—laws that govern what's possible. You give the world technologies—capabilities it has evolved. You place a conscious entity inside. You let it act, learn, sleep, evolve.

This isn't a metaphor. The physics define what's possible. The Mind defines who the agent is. We are building simulated realities for digital minds.

## Agency Is What Makes Intelligence Alive

Weights give you pattern recognition. But thinking alone doesn't scale. What makes life adaptable isn't the neural architecture—it's the freedom to act within a real environment.

Tool-call agents are a brain connected to buttons—they can only press what someone pre-wired. That's an animal in a zoo. Universe gives agents genuine agency—an animal in the wild:

- **Discover**—`man`, `--help`, `apt search`—self-directed exploration no one pre-programmed
- **Compose**—pipe anything into anything, combine tools in ways no designer anticipated
- **Create**—write scripts, build new tools, invent solutions that didn't exist before
- **Adapt**—install packages, modify configs, restructure its own approach on the fly

Tool-call frameworks scale linearly (more tools = more schemas). Agency scales combinatorially—you don't pre-engineer every behavior, you create the conditions for behavior to emerge.

> MCP gives agents a Swiss Army knife. Universe gives them a workshop.

## Agents That Evolve

Model weights never change. What evolves is the Mind.

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
go install github.com/jterrazz/universe/cmd/universe@latest

# Create an agent identity
universe agent init leonardo

# Spawn a world with the agent inside
universe spawn --agent leonardo --workspace ./my-project
```

A contained Linux environment is created. The agent's persistent identity is mounted inside. Claude Code is spawned with full shell access. When the task ends, the world is destroyed—but the agent's Mind survives. Next time it runs, it remembers what it learned.

---

## How It Works

### Physics & Technologies

The universe manifest defines the agent's reality:

```yaml
physics:
  origin: ubuntu:24.04        # The starting point — what this world is made of

  constants:                   # Finite resources, like physical constants
    cpu: 2
    memory: 1GB
    timeout: 30m

  laws:                        # Structural constraints — cannot be broken
    network: none

  elements:                    # Raw matter — files, data, mounts
    - workspace: ./my-project

technologies:                  # What this world has evolved (Civ tech tree)
  - @unix                      # Pack: bash, coreutils, grep, sed, awk, find, etc.
  - @git
  - @node
  - jq
  gate:                        # Technologies bridged from Substrate
    - source: mcp/slack
      as: slack-send
      capabilities: [send]
```

**Physics** are the laws of nature—things you can't opt out of. No network interface means the outside world doesn't exist. CPU and memory are finite. Elements are the raw matter—files and mounts. These are gravity.

**Technologies** are what the world has evolved—like a Civilization tech tree. `@unix`, `@git`, `@node` are capability packs. `jq` is an individual tool. If `curl` isn't in the tech tree, that technology was never developed. If there's no `apt`, no new technologies can ever be evolved—the tech tree is frozen forever. Technologies are verified at creation time and exposed in the agent's `/universe/faculties.md`.

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

```
Substrate (your machine)
  └── Architect                    Creates and destroys worlds
       └── Universe                A contained reality
            ├── Agent              The living entity inside
            │    └── Mind          Its persistent identity
            ├── Gate               Bridge between worlds
            ├── physics.md         The constraints of this reality
            └── faculties.md       What the agent can do
```

When you run `universe spawn`:

1. The **Architect** reads `universe.yaml` and provisions a Docker container
2. The agent's **Mind** is mounted at `/mind`, the project at `/workspace`
3. The Architect generates **`physics.md`** (constraints) and **`faculties.md`** (verified technologies + gate bridges)
4. The container-side **Gate** spawns the agent CLI via [ACP](https://github.com/agentclientprotocol/agent-client-protocol)
5. The agent reads its Mind, reads the physics and faculties, and starts working

The Gate is a two-sided bridge. The host side handles file mounts and technology bridging (MCP servers → shell commands). The container side wraps an ACP client. Because it speaks ACP, swapping agent runtimes is a container image change—Claude Code today, Codex CLI or Gemini CLI tomorrow.

---

## Commands

```bash
# World
universe spawn              # Create a universe
universe list               # List all universes
universe inspect <id>       # Show details, physics, agent status
universe logs <id>          # Stream agent output
universe attach <id>        # Interactive shell into a running universe
universe destroy <id>       # Destroy a universe (Mind survives)
universe init               # Scaffold universe.yaml + agent.yaml

# Life
universe agent spawn <id>   # Bring an agent to life in an existing universe
universe agent list          # List all agents
universe agent inspect <id>  # Show Mind layers, journal, sessions
universe agent init <name>   # Create a new Mind
universe agent export <name> # Export a Mind as tar.gz
```

### A typical session

```
$ universe spawn --agent leonardo --workspace ./acme-api

  Spawning universe...

  ✓ Provisioned container from origin ubuntu:24.04
  ✓ Mounted workspace ./acme-api → /workspace
  ✓ Generated faculties.md (14 technologies verified)
  ✓ Mounted Mind "leonardo" → /mind
  ✓ Spawned Claude Code (session a1b2c3d4)

  Universe is alive.

  ID:       u-bright-comet-84721
  Agent:    leonardo
  Origin:   ubuntu:24.04
  Status:   running

$ universe destroy u-bright-comet-84721

  ✓ Stopped agent
  ✓ Removed container
  ✓ Mind persisted at ~/.universe/agents/leonardo

  Universe destroyed. Mind survives.
```

---

## Comparison

| | Approach | What Universe adds |
| --- | --- | --- |
| **MCP** | Exposes tools one at a time | Full shell — N! compositions, not N tools |
| **LangChain / CrewAI** | Chains function calls | Emergent behavior, not deterministic chains |
| **E2B** | Cloud sandboxes | Self-hosted, persistent identity, evolution |
| **Claude Code** | Runs on your machine | Isolation, physics-based security, persistent Mind |
| **Docker** | Container runtime | Agent lifecycle, identity, Gate, evolution |

Universe isn't another link in the tool chain—it replaces the chain. And it's not a competitor to Claude Code—it's the complement. Claude Code is the intelligence. Universe is the world to be intelligent in.

---

## Stack

| | Technology | Why |
| --- | --- | --- |
| **Core** | Go | Docker SDK is first-party Go. Single binary. Goroutines. |
| **CLI** | Cobra | The standard for Go CLIs (kubectl, docker, gh) |
| **Agent** | Claude Code CLI | Best Unix-native agent. Reads markdown natively. |
| **Protocol** | [ACP](https://github.com/agentclientprotocol/agent-client-protocol) | Standard protocol, 34+ agent CLIs. Session management for free. |
| **Isolation** | Docker | Each agent gets its own container |
| **Bridge** | Gate | Two-sided. Host: mounts + technologies. Container: ACP client. |

### Project layout

```
cmd/universe/           CLI entry point (cobra)
internal/
├── architect/          Orchestrator — create, spawn, destroy
├── agent/              Agent selection and ACP spawning
├── backend/            Backend interface + Docker adapter
├── config/             Types and manifest definitions
├── gate/               Host-side Gate: technology bridge, session relay
├── journal/            Automatic spawn logs (markdown)
├── manifest/           universe.yaml + agent.yaml parsing
├── mind/               Mind directory management and validation
├── physics/            physics.md + faculties.md generation via container introspection
└── session/            Session persistence (JSON per agent+universe)
container/              Container-side Gate (ACP client + crash recovery)
```

---

## Status

Early development. Architecture designed, [wiki](https://github.com/jterrazz/universe-wiki) complete, implementation underway.

**Learn more:**
[Vision](https://github.com/jterrazz/universe-wiki/blob/main/blueprint/vision.md) ·
[Philosophy](https://github.com/jterrazz/universe-wiki/blob/main/blueprint/philosophy.md) ·
[Mind Framework](https://github.com/jterrazz/universe-wiki/blob/main/domains/systems/mind-framework.md) ·
[Epochs](https://github.com/jterrazz/universe-wiki/blob/main/epochs/) ·
[Architecture Decisions](https://github.com/jterrazz/universe-wiki/blob/main/domains/)

## License

MIT
