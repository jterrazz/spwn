<p align="center">
  <strong>spwn</strong>
</p>

<p align="center">
  The open framework for orchestrating AI agents.
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
  <img src="docs/assets/hero-v2.gif" alt="spwn — spawning an agent" width="560" />
</p>

<br/>

## Play god with AI agents.

**The building blocks of artificial life.** Assemble tools, skills, and minds into **living worlds** — one command away.

The real power of AI isn't the model — it's the model plus everything around it. Einstein in a chatbox can answer questions. Einstein in a lab with instruments, notebooks, colleagues, and years of memory can change the world. **The environment is the multiplier.**

**Agents as code. Docker for intelligence.** If Terraform is infrastructure as code, spwn is **agents as code** — stack tool packs, skill files, and profiles into a running mind, then commit the whole declaration to git. Review PRs that change an agent's behavior. Reproduce the same mind across three machines. One `spwn.yaml`, one `spwn build`, one **reproducible artifact**.

<br/>

## Quickstart

```bash
curl -fsSL https://spwn.sh/install.sh | bash
```

Four commands. One running world.

|        | Step              | Example                                    |
| ------ | ----------------- | ------------------------------------------ |
| **01** | Initialise        | `spwn init`                                |
| **02** | Compose the mind  | `spwn agent add neo --tool @spwn/python`   |
| **03** | Spawn the world   | `spwn up`                                  |
| **04** | Talk to it        | `spwn talk neo "what is this project?"`    |

Prefer a bundled demo? `spwn example install matrix`.

> **Requirements:** Docker

<br/>

## Features

<table>
<tr>
<td align="center" width="33%">
<h3>🧩 Composable Intelligence</h3>
Stack tool packs, skill files, and a profile into a running mind. Mix <code>@spwn/unix</code> + <code>@spwn/python</code> + a researcher profile and you have an autonomous scientist. <b>Docker, but for minds.</b>
</td>
<td align="center" width="33%">
<h3>🧠 Persistent Minds</h3>
Memory is a folder of markdown files, not a database. Knowledge survives across runs, playbooks accumulate, a mind that worked on your codebase last week <b>remembers it today</b>.
</td>
<td align="center" width="33%">
<h3>🧬 Agents That Evolve</h3>
<i>Dream</i> to analyze experience, <i>sleep</i> to consolidate memory, <i>fork</i> to branch. Successful patterns become playbooks. Failed ones are discarded. <b>Natural selection for behavior.</b>
</td>
</tr>
<tr>
<td align="center">
<h3>🌍 Real Physics</h3>
Every world is a Docker container with hard, kernel-enforced limits on CPU, memory, disk, and time. Break the sandbox? You can't. <b>Real constraints, not suggestions.</b>
</td>
<td align="center">
<h3>🔒 Laws of Physics, Not ACLs</h3>
No network interface means HTTP doesn't exist — not forbidden, physically impossible. No prompt jailbreak can change the laws of physics. <b>Security by absence.</b>
</td>
<td align="center">
<h3>🏗️ Multi-Agent Teams</h3>
Leaders delegate via filesystem inboxes. Workers report back. Async, observable, no orchestration glue code. <b>Collaboration as a physical medium.</b>
</td>
</tr>
<tr>
<td align="center">
<h3>🧾 Agents as Code</h3>
If Terraform is infrastructure as code, spwn is <i>agents</i> as code. Commit your agents alongside your app code. Review behavior changes in PRs. <b>Reproduce the same mind on any machine.</b>
</td>
<td align="center">
<h3>📦 Reproducible Builds</h3>
<code>spwn check</code> validates the project tree. <code>spwn build</code> flattens it into a content-hashed artifact — pinned Docker image digest, flattened agent tree, normalized manifest. <b>Byte-identical agents across environments.</b>
</td>
<td align="center">
<h3>🔌 Swappable Adapters</h3>
Every layer is a Go interface. Claude Code today, Codex tomorrow. Docker today, Firecracker next year. Markdown memory today, vector store later. <b>No runtime lock-in.</b>
</td>
</tr>
</table>

> *"The next breakthrough isn't smarter models. It's richer worlds."*

<br/>

## Projects are per-repository

**A spwn project lives in the repo, not in your home directory.**
`spwn init` turns any directory into a project — the same way `git init`
turns any directory into a git repo, or `docker init` drops a Dockerfile
and compose file. Commit your agents, your worlds, your tool composition
alongside your code.

```
my-project/
├── spwn.yaml               # manifest — the "package.json" of spwn
├── spwn/                   # committed project assets
│   ├── agents/             #   your agents — committed, travel with the repo
│   ├── worlds/             #   custom world configs
│   ├── tools/              #   `spwn tool get @community/foo` → spwn/tools/foo/
│   └── skills/             #   `spwn skill get @community/review` → spwn/skills/review/
└── .spwn/                  # gitignored local state (live world IDs, cache)
```

`spwn.yaml` is tiny — it declares which world and which agents this repo
runs. Everyone who clones the repo gets the same agents, the same world
physics, the same tool composition. **Reproducibility by construction.**

```yaml
# spwn.yaml
version: 1
name: acme-api
workspace: .

world: default
agents:
  - neo
```

**`~/.spwn/` is for your user identity only** — credentials, daemon
state, activity log. Agents and worlds don't live there. If you want to
share an agent across projects, publish it to a registry (`spwn agent
publish`) and `spwn agent get` it in the next project.

<br/>

## Inside an agent

Each agent is a directory of markdown files — **human-readable, git-friendly, no database**:

```
spwn/agents/neo/
├── agent.yaml                # composition: tools, skills, profile, runtime
├── CLAUDE.md                 # entry point the runtime reads on startup
├── core/                     # identity — profile.md, purpose.md, traits.md
├── skills/                   # procedures and checklists
├── knowledge/                # facts about the codebase
├── playbooks/                # workflows promoted from experience
└── journal/                  # session history — one file per run
```

**Three kinds of blocks: tools, skills, and a profile.** Each block is a file. Stack them into `agent.yaml`:

```yaml
# spwn/agents/neo/agent.yaml
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

**If a tool isn't listed, it doesn't exist.** Not forbidden — physically absent. Browse the full [tool catalog](docs/tool-catalog.md).

**Agents evolve through three mechanisms:**

- **Dream** (`spwn agent dream neo`) — analyze experience, promote successful patterns to playbooks. Failed ones are discarded. Natural selection for behavior.
- **Sleep** (`spwn agent sleep neo`) — graceful shutdown. Raw experience consolidates into durable knowledge. Stale strategies get pruned.
- **Fork** (`spwn agent fork neo neo-v2`) — clone an agent with everything it knows. Run copies in different environments, keep the branch that performs best.

> *"Every task leaves a trace. Every trace becomes knowledge. Every knowledge shapes the next decision."*

<br/>

## Use cases

**Solo developer** — one agent, one project, persistent memory:

```bash
spwn init
spwn agent add neo --tool @spwn/node
spwn up
spwn talk neo "Refactor the auth module to use sessions"
# neo remembers the codebase next time
```

**Team with a leader** — a lead agent delegates tasks to worker agents via inboxes:

```bash
spwn agent new morpheus
spwn agent new trinity
spwn up --agent morpheus --agent neo --agent trinity
spwn agent send neo    "Implement Stripe webhooks" --from morpheus
spwn agent send trinity "Write tests for webhooks"  --from morpheus
```

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

## Ecosystem

Every layer is a swappable Go interface. What ships today, and what's on the roadmap:

| Layer | Shipping today | On the roadmap |
|---|---|---|
| **Agent runtime** | Claude Code, Codex | Aider, Cline, Continue, OpenCode, Gemini CLI, Amazon Q, Goose |
| **LLM provider** | Anthropic, OpenAI | Google, Mistral, Groq, Together, Ollama, AWS Bedrock |
| **World runtime** | Docker | Spwn Cloud, K3s, Firecracker, Fly.io, gVisor, Podman |
| **Memory** | Markdown filesystem | RAG over Chroma, Qdrant, Pinecone, Weaviate, Turbopuffer |
| **Tool ecosystem** | `@spwn/*` packs, custom scripts | MCP servers, LangChain tools |
| **Orchestrator** | Built-in chief / worker hierarchy | Hermes, CrewAI, AutoGen, LangGraph, Swarm, Mastra |
| **Observability** | Web UI | Langfuse, LangSmith, Helicone, OpenTelemetry |

Want something else? [Open an issue](https://github.com/jterrazz/spwn/issues) — every adapter is a single Go file.

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

---

<p align="center">
  <sub>Open source. Self-hosted. Built for people who want to give agents a world, not a wrapper.</sub>
</p>
