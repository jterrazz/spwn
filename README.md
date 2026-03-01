# Universe

**Create realities for things that can think.**

---

Models like Claude can reason, write code, and solve problems. The intelligence is there. But drop that intelligence into a blank API call and it has no tools, no filesystem, no memory of yesterday. It's a brain in a jar.

A developer at a real terminal is dramatically more capable — not because of any single command, but because Unix is composable. `grep | sort | uniq`. Fifty years of tools, all piping into each other. Claude Code proved this: give an LLM a real terminal and it outperforms any chatbot with pre-wired tools. The difference isn't the model. It's the world around it.

Universe builds that world.

```bash
universe spawn --agent leonardo --workspace ./my-project
```

A contained Linux environment is created. The agent's persistent identity is mounted inside. Claude Code is spawned with full shell access. When the task ends, the world is destroyed — but the agent's Mind survives. Next time it runs, it remembers what it learned.

Three things Claude Code alone can't give you: **isolation** (each agent gets its own world), **persistent identity** (memory that survives across sessions), and **evolution** (agents that get better over time).

---

## Physics

Gravity isn't a rule you choose to follow. It's a fact of the world you're in.

A Universe works the same way. You don't *tell* the agent "don't access the network." You create a world where the network doesn't exist. No interface to bind. No packets to send. HTTP isn't forbidden — it's physically impossible, like trying to swim in a world without water.

This is defined in `universe.yaml`:

```yaml
physics:
  origin: ubuntu:24.04        # The starting point — what this world is made of

  constants:                   # Finite resources, like physical constants
    cpu: 2
    memory: 1GB
    timeout: 30m

  laws:                        # Structural constraints — cannot be broken
    network: none

  elements:                    # Tools the agent can use — listed in physics.md
    - ubuntu                   # Pack: bash, coreutils, grep, sed, awk, find, etc.
    - git
    - node
    - npm
    - jq

gate:                          # Bridges to the outside world — senses
  - source: mcp/slack
    as: slack-send
    capabilities: [send]
```

Two concepts make this work:

**Physics** are the laws of nature — things you can't opt out of. No network interface means the outside world doesn't exist. CPU and memory are finite. Filesystem permissions are structural. These are gravity.

**Elements** are the manufactured objects — tools someone built. `git`, `curl`, `python3` are screwdrivers and hammers. If `curl` isn't installed, nobody built an HTTP client. The physics might allow networking, but the tool to use it was never manufactured. And if there's no `apt`, no new tools can ever be created — the element set is locked forever. Elements listed in the manifest are verified at creation time and exposed to the agent in its `/universe/physics.md` — that's how the agent knows what tools exist in its world. You can list individual binaries or use **packs** like `ubuntu` (common shell tools) or `node` (node, npm, npx) for convenience.

This is why Universe's security can't be jailbroken. You can't prompt-inject a missing network interface. You can't social-engineer a binary that doesn't exist. You can't trick physics.

> No chains on the agent. Chains in the physics.

---

## Mind

A world with physics but nothing alive in it is just an empty room. The agent is the living entity — and its **Mind** is the persistent identity it carries between worlds.

Think of the Mind as the agent's brain structure, modeled after how biological memory works:

| Biology | Mind Layer | What it stores | Example |
| --- | --- | --- | --- |
| Personality | **Personas** | Who I am | *"Senior backend engineer. Prefers simplicity over abstraction."* |
| Skills | **Skills** | What I can do | A deployment procedure, a code review checklist |
| Semantic memory | **Knowledge** | What I know | *"This codebase uses PostgreSQL 15 with pgvector"* |
| Procedural memory | **Playbooks** | How I do things | *"To deploy: test, bump version, build, push, tag"* |
| Episodic memory | **Journal** | What happened to me | *"Session 47: migrated auth to sessions. Took 3 attempts."* |
| Senses | **Faculties** | How I perceive beyond my world | Slack messages, GitHub PRs via MCP bridges |

The Mind is declared in `mind.yaml` and mounted into every world the agent enters:

```yaml
name: leonardo

personas:
  - personas/backend-engineer.md

skills:
  - skills/deploy.md
  - skills/debug.md

knowledge:
  - knowledge/domain.md
```

Under the hood, it's a directory of markdown files — human-readable, version-controllable, no database. Claude Code reads markdown natively, so the Mind requires zero custom loaders.

The agent has full autonomy inside its world. No behavioral guardrails. No "don't do X" instructions. Dangerous actions aren't forbidden — they're physically impossible. You don't tell a fish "don't walk" in a world without land.

---

## Evolution

The model's weights never change. What evolves is the Mind.

**Reflexion.** After each session, the agent reviews its journal. What worked? Promote it to a playbook — a reusable procedure for next time. What failed? Discard it. This is natural selection applied to behavior: successful strategies survive, unsuccessful ones die. Over dozens of sessions, the agent becomes genuinely better at its job.

**Sleep.** An agent that only accumulates eventually drowns in its own experience — contradictions pile up, stale knowledge lingers, context gets noisy. Sleep is the fix. The agent pauses, consolidates raw journal entries into durable knowledge, prunes outdated playbooks, resolves contradictions, and updates its self-model. Reflexion asks *"what did I learn just now?"* Sleep asks *"given everything I've lived through, who am I now?"*

**Forking.** Clone a Mind. Run two copies in different environments with different strategies. Keep the branch that performs better. Merge the winner back. This is evolution at the population level — not sequential trial-and-error, but parallel exploration with selection.

---

## Architecture

```
Substrate (your machine)
  └── Architect                    Creates and destroys worlds
       └── Universe                A contained reality
            ├── Agent              The living entity inside
            │    └── Mind          Its persistent identity
            ├── Gate               Bridge between worlds
            └── physics.md         The laws of this reality
```

When you run `universe spawn`, here's what happens:

1. The **Architect** reads `universe.yaml` and provisions a Docker container
2. The agent's **Mind** is mounted at `/mind`, the project at `/workspace`
3. The Architect generates **`physics.md`** — a description of the world's reality that the agent reads on boot
4. The container-side **Gate** spawns the agent CLI via [ACP](https://github.com/agentclientprotocol/agent-client-protocol) (Agent Client Protocol)
5. The agent reads its Mind, reads the physics, and starts working

The Gate deserves a note — it's a two-sided bridge. The host side handles file mounts and faculty bridging (turning MCP servers into shell commands). The container side wraps an ACP client that communicates with the agent CLI over stdio. A single Unix socket connects the two halves — the only thing that crosses the container boundary.

Because the Gate speaks ACP, swapping agent runtimes is a container image change. Claude Code today. Codex CLI or Gemini CLI tomorrow. Same protocol, same Mind, same physics.

```
┌──────────────────────────────────────────────────┐
│  CLI                                             │
│  spawn · list · inspect · destroy · agent        │
└──────────────┬───────────────────────────────────┘
               ▼
         ┌───────────┐
         │ Architect  │
         └─────┬─────┘
               │
  ┌────────────┼────────────┬──────────────┐
  ▼            ▼            ▼              ▼
Backend    Manifest      Mind          Physics
(Docker)   Resolver     Manager       Generator
  │
  ▼
┌───────────────────────────────────────────────┐
│  Universe (Docker container)                  │
│                                               │
│  Gate (container-side)                        │
│  └── ACP → stdio → Claude Code               │
│                                               │
│  /mind        personas, skills, knowledge     │
│  /workspace   your project files              │
│  /gate        Unix socket, faculty bridge │
│  /universe    physics.md                      │
└───────────────────────────────────────────────┘
```

---

## Usage

```bash
# Install
go install github.com/jterrazz/universe/cmd/universe@latest

# Create an agent identity
universe agent init leonardo

# Spawn a world with the agent inside
universe spawn --agent leonardo --workspace ./my-project
```

Every universe gets a human-readable ID — `u-bright-comet-84721`, `u-calm-nebula-39205`. Two words plus a 5-digit suffix, easy to type, zero collisions.

### Commands

```bash
# World
universe spawn              # Create a universe
universe list               # List all universes
universe inspect <id>       # Show details, physics, agent status
universe logs <id>          # Stream agent output
universe attach <id>        # Interactive shell into a running universe
universe destroy <id>       # Destroy a universe (Mind survives)
universe init               # Scaffold universe.yaml + mind.yaml

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
  ✓ Generated physics.md (14 elements detected)
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

Universe isn't another link in the tool chain — it replaces the chain. And it's not a competitor to Claude Code — it's the complement. Claude Code is the intelligence. Universe is the world to be intelligent in.

---

## Stack

| | Technology | Why |
| --- | --- | --- |
| **Core** | Go | Docker SDK is first-party Go. Single binary. Goroutines. |
| **CLI** | Cobra | The standard for Go CLIs (kubectl, docker, gh) |
| **Agent** | Claude Code CLI | Best Unix-native agent. Reads markdown natively. |
| **Protocol** | [ACP](https://github.com/agentclientprotocol/agent-client-protocol) | Standard protocol, 34+ agent CLIs. Session management for free. |
| **Isolation** | Docker | Each agent gets its own container |
| **Bridge** | Gate | Two-sided. Host: mounts + faculties. Container: ACP client. |

### Project layout

```
cmd/universe/           CLI entry point (cobra)
internal/
├── architect/          Orchestrator — create, spawn, destroy
├── agent/              Agent selection and ACP spawning
├── backend/            Backend interface + Docker adapter
├── config/             Types and manifest definitions
├── gate/               Host-side Gate: faculty bridge, session relay
├── journal/            Automatic spawn logs (markdown)
├── manifest/           universe.yaml + mind.yaml parsing
├── mind/               Mind directory management and validation
├── physics/            physics.md generation via container introspection
└── session/            Session persistence (JSON per agent+universe)
container/              Container-side Gate (ACP client + crash recovery)
```

---

## Status

Early development. Architecture designed, [wiki](https://github.com/jterrazz/universe-wiki) complete, implementation underway.

**Learn more:**
[Vision](https://github.com/jterrazz/universe-wiki/blob/main/01-overview/vision.md) ·
[Philosophy](https://github.com/jterrazz/universe-wiki/blob/main/01-overview/philosophy.md) ·
[Mind Framework](https://github.com/jterrazz/universe-wiki/blob/main/02-architecture/mind-framework.md) ·
[Roadmap](https://github.com/jterrazz/universe-wiki/blob/main/01-overview/roadmap.md) ·
[Architecture Decisions](https://github.com/jterrazz/universe-wiki/blob/main/05-decisions/)

## License

MIT
