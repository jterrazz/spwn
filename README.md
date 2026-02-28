# Universe

> We're not giving agents tools. We're creating realities where they can think, act, and evolve.

## Start With Nothing

An empty Docker image. No tools, no network, no data. A vacuum. This is a world with physics but no matter — like a universe after the Big Bang, before atoms formed.

The physics are already interesting. Filesystem is gravity — structure is imposed, not chosen. Compute is energy — finite, consumed by every action, subject to entropy. And absence is the strongest force — what doesn't exist can't be jailbroken, injected, or bypassed. You can't prompt-inject the absence of `curl`.

## Add Elements

Install tools. Each one is an element in your periodic table. With `grep`, `sort`, and `uniq`, chemistry becomes possible — they combine in ways no one designed. N tools don't give you N capabilities. They give you N! — a combinatorial explosion of possible reactions.

Open a network port: you've just increased the speed of light. The universe can now perceive things beyond itself.

The Docker image doesn't configure a container. It defines the laws of nature for a world.

```bash
universe spawn --mind my-agent --image ubuntu:24.04 --workspace ./project
```

## Add Life

Place a Mind inside the world. Not a prompt — an identity. It has memory, personality, and senses that connect it to the outside world through the Gate.

| Life | Mind | What it holds |
| --- | --- | --- |
| Episodic memory | Journal | What happened to me |
| Semantic memory | Knowledge | What I know — distilled from experience |
| Procedural memory | Playbooks | What I know how to do — earned, not taught |
| Personality | Personas | Who I am |
| Senses | Interactions | How I perceive beyond my body |
| Nervous system | Gate | How signals travel between mind and world |

```
mind/
├── personas/      # WHO — identity, roles, system prompts
├── skills/        # WHAT — invocable capabilities
├── knowledge/     # KNOWING — facts, context, understanding
├── playbooks/     # HOW — step-by-step procedures
├── journal/       # WHAT HAPPENED — auto-generated session logs
└── sessions/      # CONTINUITY — Claude Code resume tokens
```

The Mind has free will. No guardrails, no behavioral chains. Why? Because dangerous actions aren't forbidden — they're physically impossible. You don't need to tell a creature "don't fly" in a world without atmosphere.

> *No chains on the mind. Chains in the physics.*

## Let It Evolve

After each session, the mind reflects. What strategies worked? Those survive. What failed? Discarded. This is natural selection for behavior — not training, not fine-tuning, but evolution.

Periodically, the mind sleeps. Not resting — reorganizing. Raw experience consolidates into durable knowledge. Stale strategies get pruned. Contradictions resolve. The mind dreams, cross-referencing disparate experiences, surfacing patterns no single session would catch.

Fork the mind. Two copies, two environments, two evolutionary paths. Speciation. Merge the successful branches back. Population-level adaptation.

Model weights never change. What evolves is the Mind.

## How It Works

The Architect creates a Docker container, mounts a Mind (persistent identity) and a workspace (project files), auto-generates a `physics.md` manifest describing the universe's reality, and spawns [Claude Code](https://docs.anthropic.com/en/docs/claude-code) inside. When the task ends, the universe is destroyed — but the Mind persists.

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

## The Concepts

```
Substrate            The base reality — your machine
  └── Architect         The builder of worlds
       └── Universe        A contained reality — its own physics, tools, and laws
            ├── Mind           A living identity — learns, sleeps, evolves
            └── Gate           The wormhole — connects Universe to Substrate
```

## How It Compares

| Solution | Approach | Universe difference |
| --- | --- | --- |
| **MCP** | Exposes individual tools | Universe gives the full shell — N! compositions |
| **LangChain / CrewAI** | Chains tool calls | Universe gives combinatorial freedom, not chains |
| **E2B** | Cloud sandboxes | Universe is self-hosted with persistent Mind and self-learning |
| **Claude Code alone** | Operates on your machine | Universe adds isolation, persistent identity, physics |
| **Docker** | Container runtime | Docker is the backend; Universe adds Mind, Gate, and agent spawning |

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
