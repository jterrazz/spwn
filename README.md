# Universe

> Creating realities where agents can think, act, and evolve.

AI agents today can reason brilliantly but can't *do* anything real. They call predefined functions, one at a time, in sequences someone had to design in advance.

But a developer at a terminal doesn't use 20 predefined tools. They have `bash` — and with it, the entire Unix ecosystem. Fifty years of composable commands, piped and chained in ways no one could anticipate. That's not more tools. That's a categorically different kind of power.

Claude Code proved it — give an LLM a real terminal and it becomes 100x more capable than a chatbot. Universe builds on this: Claude Code provides the raw reasoning and autonomy — reading files, writing code, composing commands, adapting its approach. Universe provides the cognitive architecture (Mind — persistent memory, skills, identity) and the physical reality (isolated environments with their own laws, tools, and resources). Together, they form a complete agent — reasoning within a structured, evolving identity, operating inside a contained world.

```bash
universe spawn --mind my-agent --image ubuntu:24.04 --workspace ./project
```

## The Deeper Idea

Universe isn't just "Docker for AI." Each universe is a **contained reality** with its own physics. The Docker image doesn't configure a container — it defines the laws of nature for that world. If `curl` isn't installed, HTTP doesn't exist — not as a rule, but as a fact of physics. Security isn't about telling agents what not to do. It's about creating worlds where dangerous actions are physically impossible.

Inside that reality, a **Mind** operates with genuine agency — free to think, decide, combine any tool with any other, and adapt its own approach. No behavioral chains. No guardrails. Full autonomy within the physics of its world.

And Minds aren't static. They're **living identities** that grow over time — accumulating experience in journals, refining strategies through Reflexion, sleeping to consolidate what they know, and forking into parallel branches that evolve independently. They don't just complete tasks. They get better at completing tasks.

> *No chains on the mind. Chains in the physics.*

## The Concepts

```
Substrate            The base reality — your machine
  └── Architect         The builder of worlds
       └── Universe        A contained reality — its own physics, tools, and laws
            ├── Mind           A living identity — learns, sleeps, evolves
            └── Gate           The wormhole — connects Universe to Substrate
```

| Concept | What it is |
| --- | --- |
| **Universe** | A contained reality. The Docker image defines the physics — what tools exist, what network is possible, what resources are available. |
| **Mind** | A living identity. Personas, skills, knowledge, playbooks, and a journal of lived experience. Grows over time. |
| **Gate** | The wormhole between realities. Bridges external services (MCP servers) into the universe as native CLI commands. |
| **Architect** | The builder of worlds. Creates, manages, and destroys universes on demand. |
| **Substrate** | The base reality — the machine from which all universes are spawned. |

## How It Works

The Architect creates a Docker container, mounts a Mind (persistent identity) and a workspace (project files), auto-generates a `physics.md` manifest describing the universe's reality, and spawns [Claude Code](https://docs.anthropic.com/en/docs/claude-code) inside. When the task ends, the universe is destroyed — but the Mind persists.

### The Mind

A persistent directory structure mounted into every universe. Not just memory — a complete identity:

```
mind/
├── personas/      # WHO — identity, roles, system prompts
├── skills/        # WHAT — invocable capabilities
├── knowledge/     # KNOWING — facts, context, understanding
├── playbooks/     # HOW — step-by-step procedures
├── journal/       # WHAT HAPPENED — auto-generated session logs
└── sessions/      # CONTINUITY — Claude Code resume tokens
```

Without a Mind, every universe starts from zero. With one, agents accumulate expertise, refine their approach, and learn from their own history.

### The Physics

Each universe gets an auto-generated `/universe/physics.md` describing its reality — constants (resource limits), laws (invariant rules), elements (installed binaries), interactions (bridged MCP servers), and topology (filesystem layout). The agent reads the physics to understand its world. Everything else doesn't exist.

### The Gate

An HTTP-over-Unix-socket server that bridges external services into the universe as CLI commands. The agent calls `slack-send` and the Gate routes it through the socket to the Substrate's MCP server. From the agent's perspective, the command just *exists* — part of the physics.

## Why This Matters

| | Tool-call agents | Universe agents |
| --- | --- | --- |
| **Execution** | Call predefined functions one at a time | Full shell — any command, any pipe, any composition |
| **Capabilities** | N tools = N capabilities | N tools = N! capabilities (combinatorial explosion) |
| **Learning** | Static — same behavior every time | Learn, sleep, evolve — get better over time |
| **Security** | Guardrails ("don't do X") | Physics ("X doesn't exist") |
| **Agency** | Tools to be controlled | Autonomous entities with genuine agency |

> MCP gives agents a Swiss Army knife. Universe gives them a workshop.

## Quick Start

```bash
# Build from source (requires Go 1.25+ and Docker)
git clone https://github.com/jterrazz/universe.git
cd universe && make build

# Create a Mind (persistent agent identity)
mkdir -p ~/.universe/minds/my-agent/{personas,skills,knowledge,playbooks,journal,sessions}

# Spawn an agent
universe spawn --mind my-agent --workspace ./my-project

# With a custom image and memory limit
universe spawn --mind my-agent --image node:22 --workspace ./app --memory 2g

# With external service bridging (MCP → CLI wrapper)
universe spawn --mind my-agent --workspace ./app \
  --interaction "mcp/slack:slack-send:chat.postMessage,channels.list"
```

## CLI

```bash
universe create          # Create a universe (container only)
universe spawn           # Create + start + spawn agent (all-in-one)
universe list            # List all universes
universe inspect <id>    # Inspect a universe
universe destroy <id>    # Stop and remove a universe

universe mind list       # List all minds
universe mind inspect    # Inspect mind structure and sessions
universe mind export     # Export a mind as tar.gz
```

| Flag | Description | Default |
| --- | --- | --- |
| `--image` | Docker image (defines the physics) | `ubuntu:24.04` |
| `--mind` | Mind ID (persistent identity) | — |
| `--workspace` | Host directory to mount at `/workspace` | — |
| `--memory` | Container memory limit | — |
| `--timeout` | Execution timeout | — |
| `--interaction` | MCP interaction bridge (`source:as:cap1,cap2`) | — |

## Architecture

```
┌──────────────────────────────────────────────┐
│  CLI (cobra)                                 │
│  create · spawn · list · inspect · destroy   │
└──────────────┬───────────────────────────────┘
               │
      ┌────────▼────────┐
      │    Architect     │  Orchestrator
      └────────┬────────┘
               │
  ┌────────────┼────────────┬─────────────┐
  │            │            │             │
┌─▼──────┐ ┌──▼────┐ ┌─────▼─────┐ ┌────▼─────┐
│Backend │ │Session│ │ Journal   │ │ Physics  │
│(Docker)│ │Store  │ │Generator  │ │Generator │
└────┬───┘ └───────┘ └───────────┘ └──────────┘
     │
┌────▼────────────────────────────────────┐
│  Docker Container                       │
│                                         │
│  Claude Code CLI (agent runtime)        │
│  ├── /mind       (persistent identity)  │
│  ├── /workspace  (project files)        │
│  ├── /gate       (Unix socket + bins)   │
│  └── /universe/physics.md (read-only)   │
│                                         │
└─────────────────────────────────────────┘
```

### Project Structure

```
cmd/universe/          CLI entry point + cobra commands
internal/
├── architect/         Orchestrator — create, spawn, list, inspect, destroy
├── agent/             Claude Code CLI spawning with session resume
├── backend/           Backend interface + Docker implementation
├── config/            Data structures (UniverseConfig, Interaction types)
├── gate/              HTTP-over-Unix-socket server + wrapper scripts
├── journal/           Auto-generated markdown session logs
├── mind/              Mind path resolution, validation, listing
├── physics/           physics.md generation + container introspection
├── procmgr/           Process manager with crash recovery
└── session/           Session persistence (JSON per mind+universe pair)
test/e2e/              End-to-end integration tests
```

## How It Compares

| Solution | Approach | Universe difference |
| --- | --- | --- |
| **MCP** | Exposes individual tools | Universe gives the full shell — N! compositions |
| **LangChain / CrewAI** | Chains tool calls | Universe gives combinatorial freedom, not chains |
| **E2B** | Cloud sandboxes | Universe is self-hosted with persistent Mind and self-learning |
| **Claude Code alone** | Operates on your machine | Universe adds isolation, persistent identity, physics |
| **Docker** | Container runtime | Docker is the backend; Universe adds Mind, Gate, and agent spawning |

## Development

```bash
make build              # Build the CLI binary
make vet                # Lint
make test               # Unit tests
make test-integration   # E2E tests (requires Docker)
make test-clean         # Clean up orphan test containers
```

## Roadmap

- [x] **Phase 1** — Go core + CLI (lifecycle, Docker backend, Claude Code spawning)
- [x] **Phase 2** — Full Mind + session persistence (6-layer Mind, journal, resume)
- [x] **Phase 3** — Interactions (Gate server, MCP bridging, process manager)
- [ ] **Phase 4** — Multiverse (fork, merge, reflexion, sleep) ← *current*
- [ ] **Phase 5** — Firecracker backend (microVMs, vsock, virtio-fs)
- [ ] **Phase 6** — Thin SDKs (TypeScript, Python)
- [ ] **Phase 7** — Ecosystem (templates, Mind marketplace, multi-universe orchestration)

See the full [Roadmap](https://github.com/jterrazz/universe-wiki/blob/main/01-overview/roadmap.md) and [Wiki](https://github.com/jterrazz/universe-wiki) for deep dives into architecture, philosophy, and design decisions.

## License

MIT
