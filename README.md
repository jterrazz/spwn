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
  <img src="docs/assets/hero-v2.gif" alt="spwn - spawning an agent" width="560" />
</p>

<br/>

## Play god with AI agents.

**The building blocks of artificial life.** Assemble tools, skills, and minds into **living worlds** — one command away.

The real power of AI isn't the model. It's the model *plus everything around it*. Einstein in a chatbox can answer questions. Einstein in a lab — with instruments, notebooks, colleagues, and years of memory — can change the world. **The environment is the multiplier.**

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
<code>spwn check</code> validates the project tree. <code>spwn build</code> flattens it into a content-hashed artifact - pinned Docker image digest, flattened agent tree, normalized manifest. <b>Byte-identical agents across environments.</b>
</td>
</tr>
</table>

> *"The next breakthrough isn't smarter models. It's richer worlds."*

<br/>

## Projects are per-repository

**A spwn project lives in the repo, not in your home directory.**
`spwn init` turns any directory into a project - the same way `git init`
turns any directory into a git repo, or `docker init` drops a Dockerfile
and compose file. Commit your agents, your worlds, your tool composition
alongside your code.

```
my-project/
├── spwn.yaml               # manifest - the "package.json" of spwn
├── spwn/                   # committed project assets
│   ├── agents/             #   your agents - committed, travel with the repo
│   ├── tools/              #   `spwn tool get @community/foo` → spwn/tools/foo/
│   └── skills/             #   `spwn skill get @community/review` → spwn/skills/review/
└── .spwn/                  # gitignored local state (live world IDs, cache)
```

`spwn.yaml` is tiny. Worlds live **inline** as map entries under
`worlds:` — no separate `spwn/worlds/*.yaml` files. Each world names
the agents it deploys and the workspace it mounts. Everyone who
clones the repo gets the same agents and the same tool composition.
**Reproducibility by construction.**

```yaml
# spwn.yaml
version: 2
name: acme-api

worlds:
  default:
    agents: [neo]
    workspaces: [.]
```

**`~/.spwn/` is for your user identity only** - credentials, daemon
state, activity log. Agents and worlds don't live there. If you want to
share an agent across projects, publish it to a registry (`spwn agent
publish`) and `spwn agent get` it in the next project.

<br/>

## Inside an agent

Each agent is a directory of markdown files - **human-readable, git-friendly, no database**:

```
spwn/agents/neo/
├── agent.yaml                # composition: tools, skills, profile, runtime
├── CLAUDE.md                 # entry point the runtime reads on startup
├── core/                     # identity - profile.md, purpose.md, traits.md
├── skills/                   # procedures and checklists
├── knowledge/                # facts about the codebase
├── playbooks/                # workflows promoted from experience
└── journal/                  # session history - one file per run
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

**If a tool isn't listed, it doesn't exist.** Not forbidden - physically absent. Browse the full [tool catalog](docs/tool-catalog.md).

**Agents evolve through three mechanisms:**

- **Dream** (`spwn agent dream neo`) - analyze experience, promote successful patterns to playbooks. Failed ones are discarded. Natural selection for behavior.
- **Sleep** (`spwn agent sleep neo`) - graceful shutdown. Raw experience consolidates into durable knowledge. Stale strategies get pruned.
- **Fork** (`spwn agent fork neo neo-v2`) - clone an agent with everything it knows. Run copies in different environments, keep the branch that performs best.

> *"Every task leaves a trace. Every trace becomes knowledge. Every knowledge shapes the next decision."*

<br/>

## Use cases

### Compose a scientist from blocks

```bash
spwn init
spwn agent add curie --tool @spwn/python --tool @spwn/qmd
spwn agent add curie --skill paper-reading --profile researcher
spwn up
spwn agent talk curie "reproduce the results in notebooks/exp-042.qmd and flag anomalies"
```

> Stack `@spwn/python` + `@spwn/qmd` + a `researcher` profile and you have an autonomous lab partner. Swap `researcher` for `skeptic` tomorrow - same mind, new voice. **Docker, but for minds.**

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
starts the world that contains `neo`; `spwn world default` starts the
world named `default`.

Status legend: 🟢 working · 🟡 in dev / rough edges · 🔴 planned

```
# ── Shortcuts (compose-style) ────────────────────────────────────
🟢 spwn up                                      Bring up every world in spwn.yaml
🟢 spwn up default                              Bring up one world by name
🟢 spwn agent neo                               Start the world that contains neo
🟢 spwn ls                                      Agent-centric status (running / stopped / orphan)
🟢 spwn down                                    Stop every world in spwn.yaml
🟢 spwn down default                            Stop one world by name

# ── Project (per-repo manifest) ──────────────────────────────────
🟢 spwn init                                    Scaffold a blank project in the current dir
🟢 spwn init @spwn/matrix                       Install a bundled template (matrix, startup, ...)
🟢 spwn check                                   Validate the project tree against 15 rules
🟢 spwn build                                   Flatten to .spwn/build/ (pinned artifact)

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
🟡 spwn agent add neo --profile researcher      Apply a profile (must exist locally)
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
# Lifecycle — worlds are inline entries in spwn.yaml (not files)
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
