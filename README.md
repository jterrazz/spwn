# Universe

> What does it mean to create a reality for something that can think?

Intelligence is pattern recognition—and models like Claude have mastered it. But intelligence without a world is a brain in a jar. Universe builds the missing half: not a better toolchain, but a **reality**—a world with physics, populated with elements, inhabited by a conscious entity that remembers, reflects, and evolves.

```bash
universe spawn --mind my-agent --image node:22 --workspace ./my-project
```

One command. A world is created with its own laws of nature. An agent's Mind—its persistent blueprint—is mounted inside. Claude Code is dropped in with full shell access. When the task ends, the world is destroyed—but the Mind persists. The agent remembers what it learned.

## The World Has Physics

In our world, gravity isn't a rule you follow—it's a fact you can't escape. A Universe works the same way.

The Docker image defines the agent's reality—not what's *permitted*, but what's *possible*:

- **Physics**—the laws of nature. No network interface means HTTP is *physically impossible*, not just forbidden. Memory and CPU are finite. Filesystem structure is imposed.
- **Elements**—manufactured objects. `git`, `curl`, `python3` are screwdrivers and hammers. No `curl` means no one built an HTTP client. No `apt` means no new tools can ever be manufactured—the element set is locked.

You can't prompt-inject a missing network interface. You can't jailbreak a tool that was never installed. Security isn't a rule—it's a fact of the world.

> No chains on the agent. Chains in the physics.

And because agents operate inside a real Unix shell—not a list of pre-defined functions—they get the **combinatorial freedom** of the entire command line. N tools don't give you N capabilities. They give you N!—any command's output can pipe into any other command's input.

## The World Has Life

Physics alone is a dead world. Life requires an agent—a conscious entity backed by a Mind:

| Life | Agent's Mind | What it holds |
| --- | --- | --- |
| Episodic memory | Journal | What happened to me |
| Semantic memory | Knowledge | What I know—distilled from experience |
| Procedural memory | Playbooks | How I do things—earned, not taught |
| Personality | Personas | Who I am |
| Senses | Interactions | How I perceive beyond my world |
| Nervous system | Gate | How signals travel between mind and world |

```
mind/
├── personas/      # WHO — identity, personality, system prompts
├── skills/        # WHAT — invocable capabilities
├── knowledge/     # KNOWING — facts, context, understanding
├── playbooks/     # HOW — step-by-step procedures earned through practice
├── journal/       # HISTORY — auto-generated session logs
└── sessions/      # CONTINUITY — Claude Code resume tokens
```

The agent has unconstrained will. No guardrails, no behavioral chains. Dangerous actions aren't forbidden—they're physically impossible. You don't need to tell a creature "don't fly" in a world without atmosphere.

## Life Evolves

**Reflexion**—after each session, the agent reflects. Strategies that worked get promoted to playbooks. Strategies that failed get discarded. Natural selection for behavior.

**Sleep**—periodically, the agent consolidates its journal into durable knowledge, prunes stale playbooks, resolves contradictions, and updates its self-model. Not rest—reorganization.

**Forking**—clone an agent, run copies in different environments, keep the branch that performs best, merge the winners back. Population-level adaptation.

Model weights never change. What evolves is the agent's Mind.

## How It Works

The Architect creates a Docker container, mounts the Mind and workspace, auto-generates a [`physics.md`](https://github.com/jterrazz/universe-wiki/blob/main/02-architecture/physics.md) manifest describing the world's reality, and spawns [Claude Code](https://docs.anthropic.com/en/docs/claude-code) inside.

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
│  ├── /mind       (agent blueprint)      │
│  ├── /workspace  (project files)        │
│  ├── /gate       (Unix socket + bins)   │
│  └── /universe/physics.md (read-only)   │
│                                         │
└─────────────────────────────────────────┘
```

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
| `--image` | Docker image (defines physics and elements) | `ubuntu:24.04` |
| `--mind` | Mind ID (persistent identity) | — |
| `--workspace` | Host directory to mount at `/workspace` | — |
| `--memory` | Container memory limit | — |
| `--timeout` | Execution timeout | — |
| `--interaction` | MCP interaction bridge (`source:as:cap1,cap2`) | — |

## How It Compares

| Solution | Approach | What Universe adds |
| --- | --- | --- |
| **MCP** | Exposes individual tools | Full shell—N! compositions instead of N tools |
| **LangChain / CrewAI** | Chains tool calls | Combinatorial freedom, not deterministic chains |
| **E2B** | Cloud sandboxes | Self-hosted, persistent identity, self-learning |
| **Claude Code alone** | Operates on your machine | Isolation, persistent Mind, physics-based security |
| **Docker** | Container runtime | Mind, Gate, agent spawning, lifecycle, evolution |

## Project Structure

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
- [ ] **Phase 4** — Multiverse (fork, merge, reflexion, sleep) *current*
- [ ] **Phase 5** — Firecracker backend (microVMs, vsock, virtio-fs)
- [ ] **Phase 6** — Thin SDKs (TypeScript, Python)
- [ ] **Phase 7** — Ecosystem (templates, Mind marketplace, multi-universe orchestration)

See the full [Roadmap](https://github.com/jterrazz/universe-wiki/blob/main/01-overview/roadmap.md) and [Wiki](https://github.com/jterrazz/universe-wiki) for deep dives into architecture, philosophy, and design decisions.

## License

MIT
