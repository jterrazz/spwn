<p align="center">
  <strong>spwn</strong>
</p>

<p align="center">
  The open framework for orchestrating artificial life.
</p>

<p align="center">
  <a href="#quickstart"><strong>Quickstart</strong></a> &middot;
  <a href="https://spwn.sh/docs"><strong>Docs</strong></a> &middot;
  <a href="https://spwn.sh/manifesto"><strong>Manifesto</strong></a> &middot;
  <a href="CONTRIBUTING.md"><strong>Contributing</strong></a>
</p>

<p align="center">
  <a href="https://github.com/jterrazz/spwn/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT License" /></a>
  <a href="https://github.com/jterrazz/spwn/stargazers"><img src="https://img.shields.io/github/stars/jterrazz/spwn?style=flat" alt="Stars" /></a>
  <a href="https://github.com/jterrazz/spwn/releases"><img src="https://img.shields.io/github/v/release/jterrazz/spwn?style=flat" alt="Release" /></a>
</p>

<br/>

<p align="center">
  <sub>spwn — see every world, agent, and tool at a glance. Also ships as a full CLI.</sub>
</p>

<br/>

## Play god with AI agents.

The building blocks of agent intelligence. Assemble tools, skills, and minds into living worlds — one command away.

The real power of AI isn't the model — it's the model plus everything around it. Einstein in a chatbox can answer questions. Einstein in a lab with instruments, notebooks, colleagues, and years of memory can change the world. **The environment is the multiplier.**

**Think of it as Docker for intelligence.** Docker made OS environments composable — stack base images, layers, and configs into a running container. Spwn makes *agents* composable — stack tool packs, skill files, and profiles into a running mind. One command assembles the blocks and boots the world.

Other frameworks give agents tools. **Spwn gives them a world.** Each world is a contained reality with its own filesystem, neighbors, and memory. Agents carry their identity across sessions, consolidate experience into lasting knowledge, and fork themselves to run experiments. The full environment — not just the brain — is what turns a language model into something that actually gets work done.

> *"The next breakthrough isn't smarter models. It's richer worlds."*

|        | Step             | Example                                                            |
| ------ | ---------------- | ------------------------------------------------------------------ |
| **01** | Create an agent  | `spwn agent new neo`                                               |
| **02** | Compose its mind | `spwn agent add neo --tool @spwn/python --skill paper-reading`     |
| **03** | Spawn a world    | `spwn up --agent neo -w ./my-project`                              |
| **04** | Watch it live    | Agent discovers tools, works on your code, remembers everything.   |

<br/>

## Quickstart

```bash
# Install (downloads latest release to ~/.local/bin)
curl -fsSL https://spwn.sh/install.sh | bash
```

```bash
# Create an agent and compose its mind
spwn agent new neo
spwn agent add neo --tool @spwn/python --skill paper-reading

# Spawn a world
spwn up --agent neo -w ./my-project --detach

# Talk to the agent
spwn talk neo "What is this project?"

# Check running worlds
spwn ls
```

Or start from a bundled example:

```bash
spwn example install matrix
spwn up -c matrix --agent neo
```

Or build from source:

```bash
git clone https://github.com/jterrazz/spwn.git && cd spwn
make install
```

> **Requirements:** Go 1.25+, Docker

<br/>

## Features

<table>
<tr>
<td align="center" width="33%">
<h3>🌍 Isolated Worlds</h3>
Every agent lives in a Docker container with its own filesystem, compute, and network. Real constraints. Real physics. Your host is never at risk.
</td>
<td align="center" width="33%">
<h3>🧠 They Remember You</h3>
Every agent has a profile — identity, skills, and memory. It survives across worlds and runtimes. An agent that worked on your codebase last week remembers it today.
</td>
<td align="center" width="33%">
<h3>🏗️ Multi-Agent Hierarchy</h3>
Leaders delegate to workers via inboxes. Workers report back. Async messaging, clear delegation. Teams collaborate by design.
</td>
</tr>
<tr>
<td align="center">
<h3>🔒 Laws of Physics, Not ACLs</h3>
No network interface means HTTP doesn't exist — not forbidden, physically impossible. No prompt can change the laws of physics. Security by absence.
</td>
<td align="center">
<h3>🧬 Agent Evolution</h3>
Dream to analyze, sleep to consolidate, fork to branch. Successful patterns become playbooks. Failed ones are discarded. Natural selection for behavior.
</td>
<td align="center">
<h3>🧩 Composable Intelligence</h3>
Tool packs, skill files, and profiles — all stackable blocks. Mix <code>@spwn/unix</code> + <code>@spwn/python</code> + a researcher profile = an autonomous scientist. Docker, but for minds.
</td>
</tr>
<tr>
<td align="center">
<h3>🖥️ The Web UI</h3>
See every world, agent, and tool at a glance. Know exactly what each agent can do, what it learned, and what sessions it ran. No more scattered configs.
</td>
<td align="center">
<h3>📸 Snapshots & Rollback</h3>
Capture a world at any point. Roll back to retry a different approach. Deterministic experimentation.
</td>
<td align="center">
<h3>💬 Agent Messaging</h3>
Async inter-agent communication via filesystem inboxes. Agents send, receive, and watch for messages. No orchestration glue code needed.
</td>
</tr>
</table>

<br/>

## Ecosystem support

Every layer of spwn is pluggable. Here's what works today and what's on the roadmap.

### Agent runtimes

The coding engine that runs inside each world.

| Runtime | Status |
|---|---|
| [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (Anthropic) | **Supported** |
| [Codex](https://openai.com/index/codex/) (OpenAI) | **Supported** |
| [Aider](https://aider.chat) | Planned |
| [Cline](https://cline.bot) | Planned |
| [Continue](https://continue.dev) | Planned |
| [OpenCode](https://opencode.ai) | Planned |
| [Gemini CLI](https://github.com/google-gemini/gemini-cli) (Google) | Planned |
| [Amazon Q CLI](https://aws.amazon.com/q/developer/) | Planned |
| [Goose](https://github.com/block/goose) (Block) | Planned |

### Architect / Orchestrator adapters

The brain that coordinates multi-agent collaboration.

| Orchestrator | Status |
|---|---|
| Hermes (built-in) | Planned |
| [CrewAI](https://crewai.com) | Planned |
| [AutoGen](https://github.com/microsoft/autogen) (Microsoft) | Planned |
| [LangGraph](https://github.com/langchain-ai/langgraph) (LangChain) | Planned |
| [Swarm](https://github.com/openai/swarm) (OpenAI) | Planned |
| [Mastra](https://mastra.ai) | Planned |

### LLM providers

Model backends for agent reasoning.

| Provider | Status |
|---|---|
| [Anthropic](https://anthropic.com) (Claude) | **Supported** |
| [OpenAI](https://openai.com) | **Supported** |
| [Google](https://ai.google.dev) (Gemini) | Planned |
| [Mistral](https://mistral.ai) | Planned |
| [Groq](https://groq.com) | Planned |
| [Together AI](https://together.ai) | Planned |
| [Ollama](https://ollama.com) (local) | Planned |
| [AWS Bedrock](https://aws.amazon.com/bedrock/) | Planned |

### World providers

Where worlds physically run — the isolation layer.

| Provider | Status |
|---|---|
| [Docker](https://docker.com) | **Supported** |
| Spwn Cloud (managed) | Planned |
| [K3s](https://k3s.io) / Kubernetes | Planned |
| [Firecracker](https://firecracker-microvm.github.io) (microVMs) | Planned |
| [Fly.io](https://fly.io) | Planned |
| [gVisor](https://gvisor.dev) | Planned |
| [Podman](https://podman.io) | Planned |

### Knowledge systems

How agents store and retrieve long-term knowledge.

| System | Status |
|---|---|
| Filesystem (built-in markdown) | **Supported** |
| [RAG](https://en.wikipedia.org/wiki/Retrieval-augmented_generation) (vector search) | Planned |
| [ChromaDB](https://www.trychroma.com) | Planned |
| [Qdrant](https://qdrant.tech) | Planned |
| [Pinecone](https://www.pinecone.io) | Planned |
| [Weaviate](https://weaviate.io) | Planned |
| [Turbopuffer](https://turbopuffer.com) | Planned |

### Tool ecosystems

How agents discover and use capabilities.

| Ecosystem | Status |
|---|---|
| spwn tool packs (`@spwn/*`) | **Supported** |
| Custom scripts | **Supported** |
| [MCP servers](https://modelcontextprotocol.io) | Planned |
| [LangChain tools](https://python.langchain.com/docs/integrations/tools/) | Planned |

### Organization hierarchies

How agents are structured and collaborate.

| Hierarchy | Status |
|---|---|
| Three-tier (chief / manager / worker) | **Supported** |
| Flat (all peers) | Planned |
| Custom YAML | Planned |
| DAO / consensus-based | Planned |

### Observability

Monitoring and debugging agent behavior.

| Platform | Status |
|---|---|
| Web UI (built-in GUI) | **Supported** |
| [Langfuse](https://langfuse.com) | Planned |
| [LangSmith](https://smith.langchain.com) | Planned |
| [Helicone](https://helicone.ai) | Planned |
| [OpenTelemetry](https://opentelemetry.io) | Planned |

> **Want something else?** [Open an issue](https://github.com/jterrazz/spwn/issues) or submit a PR — every adapter is a single Go interface.

<br/>

## Use cases

**Team with a leader** — a lead agent delegates tasks to worker agents via inboxes:

```bash
spwn up --agent morpheus --agent neo --agent trinity -w ./acme-api
spwn agent send neo --from morpheus "Implement Stripe webhooks"
spwn agent send trinity --from morpheus "Write tests for webhooks"
```

**Solo developer** — one agent, one project, persistent memory:

```bash
spwn agent new neo
spwn agent add neo --tool @spwn/node --skill refactoring
spwn up --agent neo -w ./my-app
spwn talk neo "Refactor the auth module to use sessions"
# neo remembers the codebase next time
```

## How agents work

Each agent is a directory of markdown files — human-readable, git-friendly, no database:

```
~/.spwn/agents/Neo/
├── agent.yaml                # composition: tools, skills, profile, runtime
├── profile.md                # personality — role, style, purpose, behavior
├── skills/                   # what the agent can do — procedures, checklists
├── knowledge/                # what the agent knows — facts about the codebase
├── playbooks/                # how the agent works — step-by-step workflows
└── journal/                  # what happened — session logs per world
```

Agents evolve through three mechanisms:

- **Dream** (`spwn agent dream neo`) — analyze experience, promote successful patterns to playbooks. Failed ones are discarded. Natural selection for behavior.
- **Sleep** (`spwn agent sleep neo`) — graceful shutdown. Raw experience consolidates into durable knowledge. Stale strategies get pruned.
- **Fork** (`spwn agent fork neo neo-v2`) — clone an agent with everything it knows. Run copies in different environments, keep the branch that performs best.

> *"Every task leaves a trace. Every trace becomes knowledge. Every knowledge shapes the next decision."*

<br/>

## Agent composition

An agent is composed from three kinds of blocks: **tools**, **skills**, and a **profile**. Each block is a file. Stack them into an agent manifest:

```yaml
# ~/.spwn/agents/neo/agent.yaml
name: neo
runtime: claude-code

profile: researcher              # personality template

tools:
  - @spwn/unix                   # bash, coreutils, grep, sed, awk
  - @spwn/git                    # version control
  - @spwn/python                 # python3, pip3
  - @spwn/claude-code            # thinking engine

skills:
  - paper-reading
  - hypothesis-testing
  - @community/rust-review
```

If a tool isn't listed, it doesn't exist. Not forbidden — physically absent. Browse the full [tool catalog](docs/tool-catalog.md).

<br/>

## CLI at a glance

Grammar is consistent: `spwn <noun> <verb>`. Three shortcuts exist for the 80% cases: `spwn up`, `spwn ls`, `spwn talk`.

```
# ── Shortcuts ────────────────────────────────────────────────────
spwn up --agent neo -w .                       Spawn a world
spwn ls                                        List active worlds
spwn talk neo "do this"                        Talk to an agent

# ── Agents ───────────────────────────────────────────────────────
spwn agent new neo                             Create a blank agent
spwn agent new neo --from @community/sci       Fork from a shared agent
spwn agent ls                                  List your agents
spwn agent show neo                            Inspect composition
spwn agent rm neo                              Delete an agent
spwn agent fork neo neo-v2                     Clone + evolve independently
spwn agent publish neo                         Ship to registry (memory stripped)  [Epoch 10]
spwn agent pull @community/curie               Install a shared agent              [Epoch 10]

spwn agent add neo --tool @spwn/python         Add a tool block
spwn agent add neo --skill paper-reading       Add a skill block
spwn agent add neo --profile researcher        Apply a profile
spwn agent rm  neo --tool @spwn/python         Remove a block

spwn agent dream neo                           Analyze experience
spwn agent sleep neo                           Consolidate memory
spwn agent talk  neo "refactor auth"           Full form of `spwn talk`

# ── Worlds ───────────────────────────────────────────────────────
spwn world up --agent neo -w .                 Full form of `spwn up`
spwn world ls                                  Full form of `spwn ls`
spwn world inspect <id>                        Inspect a running world
spwn world down <id>                           Destroy (agent survives)
spwn world enter <id>                          Interactive shell inside the world

# ── Snapshots ────────────────────────────────────────────────────
spwn snap save <id>                            Save world state
spwn snap ls                                   List snapshots
spwn snap restore <snap-id>                    Rollback
spwn snap rm <snap-id>                         Remove a snapshot

# ── Tools ────────────────────────────────────────────────────────
spwn tool ls                                   Installed tool packs
spwn tool search python                        Search the registry            [Epoch 10]
spwn tool install @spwn/python                 Install a tool pack            [Epoch 10]
spwn tool publish ./my-tool                    Ship to registry               [Epoch 10]

# ── Skills ───────────────────────────────────────────────────────
spwn skill ls                                  Your skill files
spwn skill new paper-reading                   Author a new skill
spwn skill edit paper-reading                  Open in $EDITOR
spwn skill publish paper-reading               Ship to registry               [Epoch 10]
spwn skill install @community/rust-review      Install a shared skill         [Epoch 10]

# ── Profiles ─────────────────────────────────────────────────────
spwn profile ls                                Your profiles
spwn profile new researcher                    Author a profile
spwn profile edit researcher                   Open in $EDITOR
spwn profile publish researcher                Ship to registry               [Epoch 10]
spwn profile install @community/pragmatic-dev  Install a shared profile       [Epoch 10]

# ── Messages ─────────────────────────────────────────────────────
spwn agent send neo "Implement Stripe webhooks" Async message to an agent's inbox
spwn agent inbox neo                           Show neo's inbox
spwn agent talk neo                            Live sync session

# ── System ───────────────────────────────────────────────────────
spwn architect start                           Always-on orchestration daemon
```

Full CLI reference → [`docs/cli/`](docs/cli/spwn.md)

<br/>

## Documentation

| Topic | Link |
|---|---|
| **Principles** — why spwn is built this way | [`docs/principles.md`](docs/principles.md) |
| **Agent composition** — tools, skills, profiles, memory | [`docs/composition.md`](docs/composition.md) |
| **Worlds** — spawning, isolation, lifecycle | [`docs/worlds.md`](docs/worlds.md) |
| **Tool catalog** — SDKs, runtimes, platform tools | [`docs/tool-catalog.md`](docs/tool-catalog.md) |
| **Architecture** — ports & adapters, module map | [`docs/architecture.md`](docs/architecture.md) |
| **Comparison** — vs LangChain, E2B, MCP, Docker | [`docs/comparison.md`](docs/comparison.md) |
| **CLI reference** — every command, auto-generated | [`docs/cli/`](docs/cli/spwn.md) |
| **Releasing** — release runbook | [`docs/releasing.md`](docs/releasing.md) |
| **Contributing** — setup, testing, conventions | [`CONTRIBUTING.md`](CONTRIBUTING.md) |

<br/>

## Community

- [Website](https://spwn.sh) &middot; [Docs](https://spwn.sh/docs) &middot; [Manifesto](https://spwn.sh/manifesto) &middot; [Issues](https://github.com/jterrazz/spwn/issues)

<br/>

## License

MIT © 2025 Spwn

---

<p align="center">
  <sub>Open source. Self-hosted. Built for people who want to give agents a world, not a wrapper.</sub>
</p>
