# Universe

> What does it mean to create a reality for something that can think?

Intelligence is pattern recognition—and models like Claude have mastered it. But intelligence without a world is a brain in a jar. Universe builds the missing half: not a better toolchain, but a **reality**—a world with physics, inhabited by a conscious entity that remembers, reflects, and evolves.

```bash
universe spawn --agent leonardo --workspace ./my-project
```

One command. A world is created from the origin declared in `universe.yaml`. The agent's Mind—its persistent blueprint—is mounted inside. Claude Code is dropped in with full shell access. When the task ends, the world is destroyed. The Mind persists. The agent remembers.

> Intelligence is solved. Architecture is the frontier.

## The World Has Physics

Gravity isn't a rule you follow—it's a fact you can't escape. A Universe works the same way.

The origin—declared in `universe.yaml`—defines the agent's reality. Not what's *permitted*, but what's *possible*:

- **Physics**—the laws of nature. No network interface means HTTP is *physically impossible*. Memory and CPU are finite. Filesystem structure is imposed.
- **Elements**—manufactured objects. `git`, `curl`, `python3` are screwdrivers and hammers. No `curl` means no one built an HTTP client. No `apt` means no new tools can ever be manufactured.

```yaml
# universe.yaml
origin: node:22

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

You can't prompt-inject a missing network interface. You can't jailbreak a tool that was never installed. Security isn't a rule—it's a fact of the world.

> No chains on the agent. Chains in the physics.

## The World Has Life

Physics alone is a dead world. Life requires an agent—a conscious entity backed by a Mind:

| Life | Agent's Mind | What it holds |
| --- | --- | --- |
| Personality | Personas | Who I am |
| Capabilities | Skills | What I can do |
| Semantic memory | Knowledge | What I know—distilled from experience |
| Procedural memory | Playbooks | How I do things—earned, not taught |
| Episodic memory | Journal | What happened to me |
| Senses | Interactions | How I perceive beyond my world |

The Mind is composable. Declare it in `mind.yaml`—mix personas, skills, and knowledge from local files or shared packs:

```yaml
# mind.yaml
name: leonardo

personas:
  - personas/backend-engineer.md

skills:
  - skills/deploy.md
  - skills/debug.md

knowledge:
  - knowledge/domain.md
```

The agent has unconstrained will. No guardrails. Dangerous actions aren't forbidden—they're physically impossible. You don't tell a creature "don't fly" in a world without atmosphere.

## Life Evolves

**Reflexion**—after each session, the agent reflects. Strategies that worked get promoted to playbooks. Strategies that failed get discarded. Natural selection for behavior.

**Sleep**—periodically, the agent consolidates its journal into durable knowledge, prunes stale playbooks, resolves contradictions, and updates its self-model. Not rest—reorganization.

**Forking**—clone an agent, run copies in different environments, keep the branch that performs best, merge the winners back. Population-level adaptation.

Model weights never change. What evolves is the agent's Mind.

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

The Architect reads `universe.yaml`, provisions a container (or microVM), mounts the Mind and workspace, auto-generates `physics.md` describing the world's reality, and spawns Claude Code inside.

```
┌──────────────────────────────────────────────────┐
│  CLI (cobra)                                     │
│  create · spawn · list · inspect · destroy       │
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
┌────▼──────────────────────────────────────┐
│  Universe (container / microVM)           │
│                                           │
│  Claude Code CLI (agent runtime)          │
│  ├── /mind       (agent blueprint)        │
│  ├── /workspace  (project files)          │
│  ├── /gate       (Unix socket + bins)     │
│  └── /universe/physics.md                 │
└───────────────────────────────────────────┘
```

## Quick Start

```bash
# Build from source (requires Go 1.25+ and Docker)
git clone https://github.com/jterrazz/universe.git
cd universe && make build

# Create an agent
mkdir -p ~/.universe/agents/leonardo/{personas,skills,knowledge,playbooks,journal,sessions}

# Spawn — reads universe.yaml from current directory
universe spawn --agent leonardo --workspace ./my-project

# Or specify origin directly
universe spawn --agent leonardo --origin node:22 --workspace ./my-project

# With interactions (MCP → CLI wrapper)
universe spawn --agent leonardo --workspace ./app \
  --interaction "mcp/slack:slack-send:send"
```

## CLI

```bash
universe create          # Create a universe (container only)
universe spawn           # Create + spawn agent (all-in-one)
universe list            # List all universes
universe inspect <id>    # Inspect a universe
universe destroy <id>    # Destroy a universe

universe agent list      # List all agents
universe agent inspect   # Inspect agent Mind and sessions
universe agent export    # Export an agent

universe template list      # List available universe manifests
universe template inspect   # Inspect a manifest
```

| Flag | Description |
| --- | --- |
| `--agent` | Agent name (persistent identity) |
| `--universe` | Universe manifest name or path |
| `--origin` | Origin (overrides manifest) |
| `--workspace` | Host directory to mount at `/workspace` |
| `--interaction` | MCP bridge (`source:as:cap1,cap2`) |
| `--backend` | `docker` or `firecracker` |

## Project Structure

```
cmd/universe/           CLI entry point + cobra commands
internal/
├── architect/          Orchestrator—create, spawn, list, inspect, destroy
├── agent/              Claude Code CLI spawning with session resume
├── backend/            Backend interface + Docker/Firecracker adapters
├── config/             Types: UniverseConfig, UniverseManifest, MindManifest
├── gate/               HTTP-over-Unix-socket server + wrapper scripts
├── journal/            Auto-generated markdown session logs
├── manifest/           Manifest parsing—universe.yaml and mind.yaml
├── mind/               Mind path resolution, validation, listing
├── physics/            physics.md generation + container introspection
├── procmgr/            Process manager with crash recovery
└── session/            Session persistence (JSON per agent+universe pair)
test/e2e/               Integration tests
```

## Development

```bash
make build              # Build the CLI binary
make vet                # Lint
make test               # Unit tests
make test-integration   # E2E tests (requires Docker)
```

## How It Compares

| Solution | Approach | What Universe adds |
| --- | --- | --- |
| **MCP** | Exposes individual tools | Full shell—N! compositions instead of N tools |
| **LangChain / CrewAI** | Chains tool calls | Combinatorial freedom, not deterministic chains |
| **E2B** | Cloud sandboxes | Self-hosted, persistent identity, self-learning |
| **Claude Code alone** | Operates on your machine | Isolation, persistent Mind, physics-based security |
| **Docker** | Container runtime | Mind, Gate, agent spawning, evolution |

## Learn More

See the [Wiki](https://github.com/jterrazz/universe-wiki) for deep dives into architecture, philosophy, and design decisions.

## License

MIT
