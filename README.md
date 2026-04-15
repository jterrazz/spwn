<p align="center">
  <strong>spwn</strong>
</p>

<p align="center">
  Compose AI agents as code. In your repo.
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
  <img src="docs/assets/hero-v2.gif" alt="spwn - spawning an agent" width="560" />
</p>

<br/>

## Play god with AI agents.

**The building blocks of artificial life.** Assemble tools, skills, and minds into **living worlds** - one command away.

The real power of AI isn't the model. It's the model *plus everything around it*. Oppenheimer in a chatbox can answer questions. Oppenheimer in a lab - with instruments, notebooks, colleagues, and years of memory - can change the world. **The environment is the multiplier.**

That's what spwn gives you, declaratively. If Terraform is infrastructure as code, spwn is **agents as code**: stack tool packs, skill files, and profiles into a running mind, then commit the whole declaration to git. Review the PR that changes an agent's behavior. Reproduce the same mind across three machines. One `spwn.yaml`, one `spwn build`, one **reproducible artifact**. **Docker for intelligence.**

<br/>

## Quickstart

```bash
curl -fsSL https://spwn.sh/install.sh | bash
```

Three commands. One running agent.

|        | Step          | Example                                            |
| ------ | ------------- | -------------------------------------------------- |
| **01** | Scaffold      | `spwn init`                                        |
| **02** | Bring it up   | `spwn agent neo`                                   |
| **03** | Talk to it    | `spwn agent talk neo "what is this project?"`      |

**neo** is the starter agent `spwn init` creates.

Prefer a bundled demo? `spwn init @spwn/matrix` drops a ready-made multi-agent world into the current directory (swap `matrix` for any slug under `catalog/templates/`).

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
<h3>🔒 Laws of Physics, Not ACLs</h3>
No network interface means HTTP doesn't exist - not forbidden, physically impossible. Hard, kernel-enforced limits on CPU, memory, disk, time. No prompt jailbreak can change the laws of physics. <b>Security by absence.</b>
</td>
<td align="center">
<h3>🧾 Agents as Code</h3>
If Terraform is infrastructure as code, spwn is <i>agents</i> as code. Commit your agents alongside your app code. Review behavior changes in PRs. <b>Reproduce the same mind on any machine.</b>
</td>
<td align="center">
<h3>📦 Reproducible Builds</h3>
<code>spwn check</code> validates the project tree. <code>spwn build</code> compiles it and bakes the result into a project-specific Docker image - pushable to any registry, reproducible anywhere. <b>Byte-identical agents across environments.</b>
</td>
</tr>
</table>

> *"The next breakthrough isn't smarter models. It's richer worlds."*

<br/>

## How spwn works

Three ideas to hold in your head before you dive in:

- **[Agent orchestration as code](#agent-orchestration-as-code)** - agents, worlds, and tool composition are declarative files committed alongside your code. Like Terraform or `docker-compose.yaml`, but for the agents that work on your repo. Clone it, get the same agents byte-for-byte.
- **[An agent is a directory of markdown](#inside-an-agent)** - composed from blocks (tools, skills, profile) in `agent.yaml`. Human-readable, git-friendly, no database. Evolves through dream / sleep / fork.
- **[spwn is a compiler](#spwn-is-a-compiler-for-agents)** - `spwn build` compiles your provider-neutral source and bakes it into a reproducible Docker image. Like `tsc`, but targeting agent runtimes instead of JS engines.

<br/>

### Agent orchestration as code

**Your agents, your worlds, and your tool composition are declarative files committed alongside your code** - reviewed in PRs, versioned in git, diffed like any other config. Think Terraform for infrastructure, `docker-compose.yaml` for services, `package.json` for dependencies. Spwn plays the same role for the agents that work on your repo.

`spwn init` drops the scaffold into any directory, the way `git init` or `docker init` do:

```
my-project/
├── spwn.yaml               # manifest - declares worlds, like docker-compose.yaml
├── spwn/                   # committed project assets
│   ├── agents/             #   your agents - travel with the repo
│   ├── tools/              #   `spwn tool get @community/foo` → spwn/tools/foo/
│   └── skills/             #   `spwn skill get @community/review` → spwn/skills/review/
└── .spwn/                  # gitignored local state (live world IDs, cache)
```

`spwn.yaml` is the declarative entry point. Worlds live **inline** under `worlds:` - each one names the agents it deploys and the workspace it mounts. No imperative setup scripts, no "works on my machine": whoever clones the repo gets the same agents with the same tools, byte-for-byte.

```yaml
# spwn.yaml
version: 2
name: acme-api

worlds:
  matrix:
    agents: [neo]
    workspaces: [.]
```

**`~/.spwn/` holds only your user identity** - credentials, daemon state, activity log. It's the equivalent of `~/.aws/` or `~/.docker/config.json`: personal to the machine, never the source of truth for what runs. To share an agent across projects, publish it (`spwn agent publish`) and pull it in the next repo with `spwn agent get`.

<br/>

### Inside an agent

Each agent is a directory of markdown files - **human-readable, git-friendly, no database**:

```
spwn/agents/neo/
├── agent.yaml                # composition: tools, skills, runtime
├── AGENT.md                  # entry point (provider-neutral; compiled per runtime)
├── identity/                 # who the agent is - profile.md, purpose.md, traits.md
├── skills/                   # procedures and checklists
├── knowledge/                # facts about the codebase
├── playbooks/                # workflows promoted from experience
└── journal/                  # session history - one file per run
```

**Two kinds of blocks: tools and skills.** Each block is a file. Stack them into `agent.yaml`:

```yaml
# spwn/agents/neo/agent.yaml
name: neo
runtime: claude-code

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

**If a tool isn't listed, it doesn't exist.** Not forbidden - physically absent. Browse the full [tool catalog](docs/tool-catalog.md).

Some tool packs are **plugins**: they target a runtime and inject
config (MCP servers, hooks, settings) at spawn time. They live under
a separate `plugins:` field that co-exists with `tools:`:

```yaml
plugins:
  - "@spwn/mempalace"   # memory palace for Claude Code
```

**Agents evolve through three mechanisms:**

- **Dream** (`spwn agent dream neo`) - analyze experience, promote successful patterns to playbooks. Failed ones are discarded. Natural selection for behavior.
- **Sleep** (`spwn agent sleep neo`) - graceful shutdown. Raw experience consolidates into durable knowledge. Stale strategies get pruned.
- **Fork** (`spwn agent fork neo neo-v2`) - clone an agent with everything it knows. Run copies in different environments, keep the branch that performs best.

> *"Every task leaves a trace. Every trace becomes knowledge. Every knowledge shapes the next decision."*

<br/>

### spwn is a compiler for agents

Think of spwn the way you think of `tsc` or `babel`. You write in one clean source language; a compiler emits whatever flavor your target runtime actually wants. You never touch the output by hand.

```
 spwn/           spwn build          Docker image
 (source)   ──────────────────▶     (artifact you run)
             compile  +  bake
```

- **Source** is provider-neutral. `AGENT.md`, `core/`, `skills/`, `agent.yaml` - nothing in your repo mentions Claude Code, Codex, or any runtime by name.
- **Compile** renders that source into the exact file layout your chosen runtime expects. Claude Code wants `CLAUDE.md` in a particular place? The claude-code backend emits it. Codex wants something else? Its backend emits that. Same source, different targets - like compiling TypeScript to ES5 vs ES2022.
- **Bake** links the compiled tree with the tools your agent declared and produces a normal Docker image. Push it, pull it, run it anywhere - byte-identical on every machine.

`spwn check` is the type-checker: it runs the compile step in dry-run to catch broken imports, missing skills, and invalid tool refs before you ever touch Docker.

Switching runtimes is a one-line change in `agent.yaml` - no source edits, no lock-in. See [`packages/compile/README.md`](packages/compile/README.md) for internals and how to add a new backend.

<br/>

## Use cases

### Compose a scientist from blocks

```bash
spwn init
spwn agent add curie --tool @spwn/python --tool @spwn/qmd
spwn agent add curie --skill paper-reading --skill hypothesis-testing
spwn up
spwn agent talk curie "reproduce the results in notebooks/exp-042.qmd and flag anomalies"
```

> Stack `@spwn/python` + `@spwn/qmd` + the right skills and you have an autonomous lab partner. Edit `identity/profile.md` tomorrow - same mind, new voice. **Docker, but for minds.**

### Ship an agent with your repo

```bash
cd acme-api
spwn init
spwn agent add neo --tool @spwn/node --tool @spwn/git

git add spwn.yaml spwn/
git commit -m "add neo, our repo maintainer"
git push

# every teammate who clones the repo gets the same mind, byte-for-byte
```

> Agents as code, shared like code. PR-review a behavior change. Bisect an agent's memory like bisecting a bug. **The Dockerfile metaphor, all the way.**

### Fork a mind, throw it away if it breaks

```bash
spwn agent fork neo neo-migration       # clone composition + memory
spwn up --agent neo-migration
spwn agent talk neo-migration "migrate the whole repo from Jest to Vitest"

# worked? promote.  didn't? neo is untouched, no regrets.
spwn agent rm neo-migration
```

> The only AI assistant that lets you `git checkout -b` your agent. Run a destructive refactor in a branch; keep or discard based on the diff. **Natural selection for behavior.**

### Unleash untrusted code in a sealed room

```bash
git clone https://github.com/someone/sus-repo /tmp/sus && cd /tmp/sus
spwn init
spwn up                    # no network, hard limits on CPU/mem/disk/time
spwn agent talk neo "run every test and benchmark, tell me what the code actually does"
```

> No network interface means the sandbox **can't phone home**. Kernel-enforced CPU, memory, disk, and time caps mean it **can't melt your machine**. A safe room for running code you don't yet trust. **Security by absence.**

<br/>

## CLI at a glance

Grammar is consistent: `spwn <noun> <verb>`. Compose-style
shortcuts exist for the 80% cases: `spwn up`, `spwn ls`, `spwn down`.
With no arguments they act on every world declared in `spwn.yaml`.
Name-only shortcuts start a single entity by name: `spwn agent neo`
starts the world that contains `neo`; `spwn world matrix` starts the
world named `matrix`.

Status legend: 🟢 working · 🟡 in dev / rough edges · 🔴 planned

```
# ── Shortcuts (compose-style) ────────────────────────────────────
🟢 spwn up                                      Bring up every world in spwn.yaml
🟢 spwn up matrix                               Bring up one world by name
🟢 spwn agent neo                               Start the world that contains neo
🟢 spwn ls                                      Agent-centric status (running / stopped / orphan)
🟢 spwn down                                    Stop every world in spwn.yaml
🟢 spwn down matrix                             Stop one world by name

# ── Project (per-repo manifest) ──────────────────────────────────
🟢 spwn init                                    Scaffold a blank project in the current dir
🟢 spwn init @spwn/matrix                       Install a bundled template (matrix, startup, ...)
🟢 spwn check                                   Validate the project tree against 15 rules
🟢 spwn compile                                 Render the project tree to ./dist (preview / debug)
🟢 spwn build                                   Compile + bake into a project-specific Docker image

# ── Agents ───────────────────────────────────────────────────────
# Lifecycle
🟢 spwn agent create neo                        Create a blank agent (auto-creates a single-agent world)
🟢 spwn agent ls                                List your agents
🟢 spwn agent rm neo                            Delete an agent
🟢 spwn agent start neo                         Start the world that contains neo
🟢 spwn agent stop  neo                         Stop the world that contains neo
🟢 spwn agent neo                               Shortcut for `spwn agent start neo`
🟡 spwn agent fork neo neo-v2                   Clone + evolve independently

# Observe
🟢 spwn agent inspect neo                       Inspect composition, memory, history
🟢 spwn agent logs neo                          Event log for this agent

# Compose blocks
🟢 spwn agent add neo --tool @spwn/python       Add a tool block
🟡 spwn agent add neo --skill paper-reading     Add a skill block (must exist locally)
🟢 spwn agent rm  neo --tool @spwn/python       Remove a block

# Talk + messaging
🟢 spwn agent talk  neo "refactor auth"         Full form of `spwn talk`
🟡 spwn agent send  neo "do this" --from me     Async message to an agent's inbox
🟡 spwn agent inbox neo                         Show neo's inbox
🟡 spwn agent watch neo                         Tail neo's inbox live

# Evolution
🟡 spwn agent dream neo                         Analyze experience, promote playbooks
🟡 spwn agent sleep neo                         Consolidate memory, prune stale patterns

# Portability
🟡 spwn agent export neo                        Archive to neo.tar.gz
🟡 spwn agent import ./neo.tar.gz               Install from archive
🔴 spwn agent get @community/sci                Install a shared agent from a registry
🔴 spwn agent publish neo                       Ship to registry (memory stripped)

# ── Worlds ───────────────────────────────────────────────────────
# Lifecycle - worlds are inline entries in spwn.yaml (not files)
🟢 spwn world create matrix --agent neo         Declare a new world in spwn.yaml
🟢 spwn world rm matrix                         Remove a world declaration from spwn.yaml
🟢 spwn world ls                                List declared worlds (with status column)
🟢 spwn world start [name]                      Start a world (no-arg: every world in spwn.yaml)
🟢 spwn world stop  [name]                      Stop a world
🟢 spwn world matrix                            Shortcut for `spwn world start matrix`
🟢 spwn world rename <id> <name>                Rename a running world

# Observe
🟢 spwn world inspect <id>                      Inspect composition + runtime state
🟢 spwn world logs <id>                         Event log for a world
🟢 spwn world enter <id>                        Interactive shell inside the world

# Snapshots
🟡 spwn world snap save <id>                    Save world state
🟡 spwn world snap ls                           List snapshots
🟡 spwn world snap restore <snap-id>            Rollback to a snapshot
🟡 spwn world snap rm <snap-id>                 Remove a snapshot

# Shared knowledge
🟢 spwn world knowledge ls <id>                 List a world's shared knowledge files
🟢 spwn world knowledge show <id> <path>        Read a knowledge file

# ── Tools / Skills / Profiles ────────────────────────────────────
🟢 spwn tool    ls                              Installed tool packs
🟢 spwn tool    show <pack>                     Inspect a tool pack
🟢 spwn tool    rm   <pack>                     Uninstall a local tool pack
🔴 spwn tool    get     @community/rust-fuzzer  Install a community tool pack
🔴 spwn tool    search  python                  Search the registry
🔴 spwn tool    publish ./my-tool               Ship to registry

🟢 spwn skill   ls                              Your skill files
🟢 spwn skill   new  paper-reading              Author a new skill
🟡 spwn skill   edit paper-reading              Open in $EDITOR
🟢 spwn skill   show paper-reading              Display a skill
🟢 spwn skill   rm   paper-reading              Delete a skill
🔴 spwn skill   get  @community/rust-review     Install a shared skill
🔴 spwn skill   publish paper-reading           Ship to registry

🟢 spwn profile ls                              Your personality templates
🟢 spwn profile new  researcher                 Author a new profile
🟡 spwn profile edit researcher                 Open in $EDITOR
🟢 spwn profile show researcher                 Display a profile
🟢 spwn profile rm   researcher                 Delete a profile
🔴 spwn profile get  @community/pragmatic-dev   Install a shared profile
🔴 spwn profile publish researcher              Ship to registry

# ── Teams & orgs ─────────────────────────────────────────────────
🟡 spwn team new     acme                       Create a team
🟡 spwn team ls                                 List teams
🟡 spwn team assign  neo acme                   Attach an agent to a team
🟡 spwn team members acme                       List a team's agents
🟡 spwn organization ls                         List organizations
🟡 spwn organization inspect <name>             Show roles in an organization

# ── Architect daemon ─────────────────────────────────────────────
🟡 spwn architect start                         Start the always-on daemon
🟡 spwn architect stop                          Stop it
🟡 spwn architect status                        Show status and active worlds
🟡 spwn architect talk "..."                    Talk to the Architect
🟡 spwn architect logs                          Show the Architect's event log

# ── System ───────────────────────────────────────────────────────
🟡 spwn web                                     Open the local web UI
🟢 spwn status                                  Global status (worlds, auth, version)
🟢 spwn auth login                              Connect Anthropic / OpenAI
🟢 spwn auth logout                             Clear cached credentials
🟢 spwn auth token <value>                      Set a token directly (CI)
🟢 spwn auth check                              Validate credentials across providers
🟢 spwn upgrade                                 Self-update the CLI
```

Full CLI reference → [`docs/cli/`](docs/cli/spwn.md)

<br/>

## Ecosystem

Every layer is a swappable Go interface. Same status legend as the CLI table:
🟢 working · 🟡 in dev · 🔴 planned.

| Layer | Shipping today | On the roadmap |
|---|---|---|
| **Agent runtime** | 🟢 Claude Code | 🔴 Codex, Aider, Cline, Continue, OpenCode, Gemini CLI, Amazon Q, Goose |
| **LLM provider** | 🟢 Anthropic · 🟡 OpenAI | 🔴 Google, Mistral, Groq, Together, Ollama, AWS Bedrock |
| **World runtime** | 🟢 Docker | 🔴 Spwn Cloud, K3s, Firecracker, Fly.io, gVisor, Podman |
| **Memory** | 🟢 Markdown filesystem | 🔴 RAG over Chroma, Qdrant, Pinecone, Weaviate, Turbopuffer |
| **Tool ecosystem** | 🟢 `@spwn/*` built-in packs · 🟡 local custom packs | 🔴 MCP servers, LangChain tools |
| **Orchestrator** | 🟡 built-in chief / worker hierarchy | 🔴 Hermes, CrewAI, AutoGen, LangGraph, Swarm, Mastra |
| **Observability** | 🟡 Web UI | 🔴 Langfuse, LangSmith, Helicone, OpenTelemetry |

Want something else? [Open an issue](https://github.com/jterrazz/spwn/issues) - every adapter is a single Go file.

<br/>

## Documentation

| Topic | Link |
|---|---|
| **Principles** - why spwn is built this way | [`docs/principles.md`](docs/principles.md) |
| **Architecture** - module map, core abstractions, invariants | [`docs/architecture.md`](docs/architecture.md) |
| **Worlds** - spawning, isolation, tools-as-structure | [`docs/worlds.md`](docs/worlds.md) |
| **Tool catalog** - how tool packs work, how to add one | [`docs/tool-catalog.md`](docs/tool-catalog.md) |
| **CLI reference** - every command, auto-generated | [`docs/cli/`](docs/cli/spwn.md) |
| **Releasing** - release runbook | [`docs/releasing.md`](docs/releasing.md) |
| **Update system** - CLI + Tauri auto-update, channels | [`docs/update-system.md`](docs/update-system.md) |
| **Contributing** - setup, testing, conventions | [`CONTRIBUTING.md`](CONTRIBUTING.md) |

<br/>

## Community

- [Website](https://spwn.sh) &middot; [Docs](https://spwn.sh/docs) &middot; [Manifesto](https://spwn.sh/manifesto) &middot; [Issues](https://github.com/jterrazz/spwn/issues)

---

<p align="center">
  <sub>Open source. Self-hosted. Built for people who want to give agents a world, not a wrapper.</sub>
</p>
