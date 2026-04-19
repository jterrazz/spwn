<p align="center">
  <strong>spwn</strong>
</p>

<p align="center">
  Compose AI agents as code. One source, run anywhere.
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

**The building blocks of artificial life.** Compose any mix of tools, skills, hooks, and an identity into a living agent; spawn it into a world; commit the whole thing to git. Every block is a declarative file — reviewed in PRs, pinned in lockfiles, reproducible on any machine.

The real power of AI isn't the model — it's the model *plus everything around it*. Oppenheimer in a chatbox can answer questions; Oppenheimer in a lab, surrounded by instruments, notebooks, colleagues, and years of memory, can change the world. **The environment is the multiplier.**

That's what spwn gives you. If Terraform is infrastructure as code, spwn is **agents as code** — the same discipline, now for the minds that work on your repo. One `spwn build`, one portable artifact. **Docker for intelligence.**

<br/>

## Quickstart

```bash
curl -fsSL https://spwn.sh/install.sh | bash
```

Three commands. One working agent.

|        | Step                 | Command            | What it does                                                                                                |
| ------ | -------------------- | ------------------ | ----------------------------------------------------------------------------------------------------------- |
| **01** | Log in               | `spwn auth`        | Checks you're signed in to Claude Code (or any supported runtime).                                          |
| **02** | Scaffold a project   | `spwn init`        | Drops `spwn.yaml` + a starter `neo` agent into the current directory.                                       |
| **03** | Talk to your agent   | `spwn agent neo`   | Opens an interactive session with neo inside a sandboxed Docker world — container lifecycle handled for you. |

Prefer a bundled demo? `spwn init spwn:matrix` drops a ready-made multi-agent world into the current directory (swap `matrix` for any slug under `catalog/examples/`).

> **Requirements:** Docker

<br/>

## Features

<table>
<tr>
<td align="center" width="33%">
<h3>🧾 Agents as Code</h3>
Commit agents alongside your app, review behavior changes in PRs, ship the same mind to every machine.
</td>
<td align="center" width="33%">
<h3>🧩 Composable</h3>
Stack tools, skills, hooks, and a soul into an agent — Lego for minds.
</td>
<td align="center" width="33%">
<h3>📦 Reproducible</h3>
One <code>spwn build</code> produces a Docker image — or a runtime-native tree (claude-code, codex, …) if you skip Docker. Byte-identical on every machine.
</td>
</tr>
<tr>
<td align="center">
<h3>🪐 Worlds</h3>
Bundle agents, workspaces, and knowledge into a world — <code>spwn up</code> deploys them together, <code>spwn down</code> tears it all down.
</td>
<td align="center">
<h3>🧠 Persistent</h3>
Memory is a folder of markdown files — readable, diffable, and alive across restarts.
</td>
<td align="center">
<h3>🧐 Checked</h3>
<code>spwn check</code> walks your project and surfaces bad refs, missing files, or lockfile drift before spawn.
</td>
</tr>
</table>

> *"The next breakthrough isn't smarter models. It's richer worlds."*

<br/>

## How spwn works

spwn turns your repo into a **portable agent artifact** — consumed by the spwn CLI today, and by web UIs, apps, and embedded SDKs on the roadmap. One bundle format, many future surfaces.

```
   repo  ──▶  spwn build  ──▶  artifact  ──▶  anywhere
```

Four ideas to hold in your head before you dive in:

- **[One file, one agent](#one-file-one-agent)** — `agent.yaml` lists the runtime, tools, skills, hooks. Human-readable. Git-friendly. No database.
- **[Lives in your repo, not a SaaS](#lives-in-your-repo-not-a-saas)** — every agent is a folder in your project. Commit it, review it, diff it like any other code.
- **[A world is one `spwn up` away](#a-world-is-one-spwn-up-away)** — group agents, workspaces, and knowledge; launch them together; tear them down together.
- **[Runtime-agnostic](#runtime-agnostic)** — works with Claude Code today, Codex tomorrow. Swap backends with one line.

<br/>

### One file, one agent

An agent **is** a composition of blocks, declared in one file:

```yaml
# spwn/agents/neo/agent.yaml
name: neo
runtime:
  backend: "spwn:claude-code"

dependencies:
  - "spwn:unix"          # catalog: shell + coreutils
  - "spwn:python"        # catalog: python 3 + pip
  - "skill:code-review"  # local:   ./spwn/skills/code-review.md
  - "tool:greet"         # local:   ./spwn/tools/greet/
  - "hook:pre-spawn"     # local:   ./spwn/hooks/pre-spawn.sh
```

**Every dependency is a `scheme:target` ref.** Five schemes, nothing else:

| Scheme | Resolves to |
|---|---|
| `spwn:<name>` | Built-in catalog dep compiled into the binary |
| `github:<owner>/<repo>` | Community registry *(planned)* |
| `skill:<name>` | `./spwn/skills/<name>.md` |
| `tool:<name>` | `./spwn/tools/<name>/` (with `tool.yaml`) |
| `hook:<name>` | `./spwn/hooks/<name>.sh` |

Add one with `spwn install <ref> --agent neo` — the ref lands in `agent.yaml` and pins in `spwn.lock`. Browse the full [dependency catalog](docs/dependency-catalog.md).

The rest of the agent directory sits next to the manifest — identity and memory as plain files:

```
spwn/agents/neo/
├── agent.yaml       # composition (the file above)
├── SOUL.md          # identity — who the agent is
├── AGENTS.md        # boot-time prompt — what it should do
├── playbooks/       # memory — procedures the agent has learned
└── journal/         # memory — session history
```

<br/>

### Lives in your repo, not a SaaS

**Your agents and their composition are declarative files committed alongside your code** - reviewed in PRs, versioned in git, diffed like any other config. Think Terraform for infrastructure, `docker-compose.yaml` for services, `package.json` for dependencies. Spwn plays the same role for the agents that work on your repo.

`spwn init` drops the scaffold into any directory, the way `git init` or `docker init` do:

```
my-project/
├── spwn.yaml              # manifest (the thing that ties everything together)
├── spwn.lock              # lockfile (pinned catalog deps)
├── spwn/                  # committed project assets
│   ├── agents/            # one subdir per agent (the block you saw above)
│   ├── skills/            # reusable skill files (markdown blocks)
│   ├── tools/             # local tool definitions
│   └── hooks/             # shell hooks the runtime fires
├── knowledge/             # opt-in world-scoped knowledge base
└── .spwn/                 # gitignored local state
```

Whoever clones the repo gets the same agents with the same tools, byte-for-byte. No imperative setup scripts, no "works on my machine".

**`~/.spwn/` holds only your user identity** - credentials, daemon state, activity log. It's the equivalent of `~/.aws/` or `~/.docker/config.json`: personal to the machine, never the source of truth for what runs. To share an agent across projects, publish it (`spwn agent publish`) and pull it in the next repo with `spwn agent get`.

<br/>

### A world is one `spwn up` away

An agent defines **what** can think. A **world** defines *where* and *with whom* they run. Worlds are the runtime unit: one long-running container per world, one shared filesystem, one declared set of agents talking to each other and to the mounted workspace.

Worlds live **inline** under `spwn.yaml#worlds:` — each entry names the agents it deploys, the workspaces it mounts, and the optional knowledge base it exposes.

```yaml
# spwn.yaml
version: 1
name: acme-api

worlds:
  matrix:
    agents: [neo]
    workspaces: [.]          # host paths mounted under /workspaces/. Use `name=path` to name them.
    knowledge: ./knowledge   # optional; bind into /world/knowledge/. Omit for no mount.
```

`spwn up` materialises every world in the manifest; `spwn down` tears them down. A single agent can appear in many worlds — each world keeps its own runtime state (sessions, inbox, shared scratchpad), separate from the agent's long-lived memory on disk. Destroying a world doesn't destroy the agent.

<br/>

### Runtime-agnostic

Think of spwn the way you think of `tsc` or `babel`. You write in one clean, provider-neutral source; a transpiler adapts it to whatever runtime you target and emits exactly what that runtime expects. You never touch the output by hand.

```
   YOUR REPO             BUILD                ARTIFACT
  ───────────          ─────────             ──────────────────────────
   spwn.yaml                                ┌──▶  Docker image
   spwn/agents/         spwn build          │     (push, pull, run anywhere)
   spwn/skills/    ──▶  transpile     ──▶  ─┤
   spwn/tools/          + compile           │
   spwn/hooks/                              └──▶  runtime-native tree
                                                  (claude-code, codex — no Docker)
```

- **Source** is provider-neutral. `AGENTS.md`, `SOUL.md`, `skills/`, `agent.yaml` - nothing in your repo mentions Claude Code, Codex, or any runtime by name.
- **Transpile** renders that source into the exact file layout your chosen runtime expects. Claude Code wants `CLAUDE.md` in a particular place? The claude-code backend emits it. Codex wants something else? Its backend emits that. Same source, different targets - like transpiling TypeScript to ES5 vs ES2022.
- **Compile** links the transpiled tree with the tools your agent declared and produces a normal Docker image. Push it, pull it, run it anywhere - byte-identical on every machine.

`spwn check` is the type-checker: it runs the transpile step in dry-run to catch broken imports, missing skills, and invalid tool refs before you ever touch Docker.

Switching runtimes is a one-line change in `agent.yaml` - no source edits, no lock-in. See [`packages/transpile/README.md`](packages/transpile/README.md) for internals and how to add a new backend.

<br/>

## Use cases

### Compose a scientist from blocks

```bash
spwn init
spwn install python --agent curie
spwn install qmd --agent curie
spwn install skill:paper-reading --agent curie
spwn up
spwn agent talk curie "reproduce the results in notebooks/exp-042.qmd and flag anomalies"
```

> Stack `spwn:python` + `spwn:qmd` + the right skills and you have an autonomous lab partner. Edit `SOUL.md` tomorrow - same mind, new voice. **Docker, but for minds.**

### Ship an agent with your repo

```bash
cd acme-api
spwn init
spwn install node --agent neo
spwn install git --agent neo

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

The day-one surface — twelve commands that take you from empty
directory to running agents. Everything else (teams, snapshots,
evolution, …) lives in [Implementation status](#implementation-status)
below.

```
# ── Start ────────────────────────────────────────────────────────
spwn init                             Scaffold a project
spwn init spwn:matrix                Install a bundled example
spwn check                            Validate the project tree
spwn up                               Bring up every world in spwn.yaml
spwn down                             Stop every world

# ── Compose ──────────────────────────────────────────────────────
spwn install python                   Install a catalog dep (every agent)
spwn install qmd --agent neo          Install a catalog dep (one agent)
spwn install skill:focus --agent neo  Attach a local skill
spwn agent create neo                 Create an agent + its world
spwn agent neo                        Interactive session with neo

# ── Observe ─────────────────────────────────────────────────────
spwn ls                               Agent-centric status
spwn status                           Global status (worlds, auth, version)
spwn inspect [agent]                  Per-agent composition tree
```

Full CLI reference → [`docs/cli/`](docs/cli/spwn.md)

<br/>

## Ecosystem

Every layer is a swappable Go interface. The table below is what
actually ships today — the full roadmap across every adapter lives
in [Implementation status](#implementation-status).

| Layer | Shipping today |
|---|---|
| **Agent runtime** | Claude Code |
| **LLM provider** | Anthropic · OpenAI (partial) |
| **World runtime** | Docker |
| **Memory** | Markdown filesystem |
| **Tool ecosystem** | `spwn:*` built-in dependencies, local custom dependencies |

Want something else? [Open an issue](https://github.com/jterrazz/spwn/issues) - every adapter is a single Go file.

<br/>

## Implementation status

A full ledger of every command and every adapter slot. Expand a
group to see the list. Each summary shows a progress bar
(`█` done, `▓` in dev, `░` planned) plus the shipped count.

**Legend** 🟢 shipping · 🟡 in dev · 🔴 planned

### Domains

The bird's-eye view. Each row is a whole area of the system; the
commands and adapters below belong to one or more of these.

| Domain | Scope | Status |
|---|---|:---:|
| **Transpile** | Source tree → rendered Tree (SOUL, deps, system skills woven in) | 🟢 |
| **Compile** | Tree → reproducible Docker image | 🟢 |
| **Compose** | `install` / `uninstall` / pinning (project-wide + `--agent` scoping) | 🟢 |
| **Identity** | `SOUL.md` at agent root — per-agent voice, purpose, principles | 🟢 |
| **Lint / check** | Static rules on manifests + tree (scheme grammar, one-agent-one-world, lockfile drift…) | 🟡 |
| **Mind** | 2-layer persistent memory: `playbooks/` `journal/` (skills are dependencies, not memory) | 🟡 |
| **Knowledge** | World-scoped `./knowledge/` bind-mount (opt-in per world) | 🟡 |
| **Runtimes** | `claude-code`, `codex` — swappable Go adapters | 🟡 |
| **Architect** | Always-on orchestration daemon — spawns worlds, routes inboxes, delegates | 🟡 |
| **Evolution** | `dream` / `sleep` / `fork` (playbook promotion, session replay) | 🟡 |
| **Observability** | Per-session journal, activity log, `spwn logs` | 🟡 |
| **Teams & orgs** | Group agents into coordinated units (chief / workers, role structures) | 🟡 |
| **Web dashboard** | Agent roster + composition viewer (`apps/web`) | 🟡 |
| **Apps / SDK** | Programmatic Go SDK for embedding spwn in external tools | 🔴 |
| **Managed agents** | Autonomous daemon mode (`agent start` / `agent stop`, hosted) | 🔴 |
| **Evaluation** | Task-level pass/fail, quality metrics, replay diffing | 🔴 |
| **Registry** | `agent publish` / `agent get` — shared agents on the hub | 🔴 |

### CLI

<details>
<summary><b>Shortcuts</b> &middot; <code>██████ 6/6</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn up` | Bring up every world in spwn.yaml | 🟢 |
| `spwn up <name>` | Bring up one world by name | 🟢 |
| `spwn down` | Stop every world | 🟢 |
| `spwn down <name>` | Stop one world | 🟢 |
| `spwn agent <name>` | Start the world containing an agent + attach | 🟢 |
| `spwn ls` | Agent-centric status (running / stopped / orphan) | 🟢 |

</details>

<details>
<summary><b>Project</b> &middot; <code>█████ 5/5</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn init` | Scaffold a blank project | 🟢 |
| `spwn init spwn:<template>` | Install a bundled example | 🟢 |
| `spwn check` | Validate the project tree (16 rules) | 🟢 |
| `spwn build` | Transpile + compile the project image | 🟢 |
| `spwn build --tree-only` | Render the transpiled tree to ./dist | 🟢 |

</details>

<details>
<summary><b>Agents · lifecycle</b> &middot; <code>████▓▓░░ 4/8</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn agent create <name>` | Create a blank agent (auto-creates single-agent world) | 🟢 |
| `spwn agent ls` | List agents | 🟢 |
| `spwn agent rm <name>` | Delete an agent | 🟢 |
| `spwn agent <name>` | Interactive session (boots world if needed) | 🟢 |
| `spwn agent fork <src> <dst>` | Clone + evolve independently | 🟡 |
| `spwn agent dream <name>` | Analyze experience, promote playbooks | 🟡 |
| `spwn agent start <name>` | Run agent as autonomous daemon | 🔴 |
| `spwn agent stop <name>` | Kill agent's daemon loop | 🔴 |

</details>

<details>
<summary><b>Agents · observe</b> &middot; <code>████ 2/2</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn agent inspect <name>` | Composition, memory, history | 🟢 |
| `spwn agent logs <name>` | Event log for one agent | 🟢 |

</details>

<details>
<summary><b>Compose</b> &middot; <code>████ 3/3</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn install <ref>` | Install a dep into every agent (npm-style) | 🟢 |
| `spwn install <ref> --agent <name>` | Install a dep into one specific agent | 🟢 |
| `spwn uninstall <ref> [--agent <name>]` | Detach a dep; project-wide or per-agent | 🟢 |

</details>

<details>
<summary><b>Agents · talk + messaging</b> &middot; <code>█▓▓▓ 1/4</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn agent talk <name> "..."` | Full form of `spwn talk` | 🟢 |
| `spwn agent send <name> "..." --from <sender>` | Async message to inbox | 🟡 |
| `spwn agent inbox <name>` | Show agent's inbox | 🟡 |
| `spwn agent watch <name>` | Tail agent's inbox live | 🟡 |

</details>

<details>
<summary><b>Agents · portability</b> &middot; <code>▓▓░░ 0/4</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn agent export <name>` | Archive to `<name>.tar.gz` | 🟡 |
| `spwn agent import <path>` | Install from archive | 🟡 |
| `spwn agent get github:<owner>/<repo>` | Install shared agent from registry | 🔴 |
| `spwn agent publish <name>` | Ship to registry (memory stripped) | 🔴 |

</details>

<details>
<summary><b>Agents · evolution</b> &middot; <code>▓▓ 0/2</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn agent dream <name>` | Analyze experience, promote playbooks | 🟡 |
| `spwn agent sleep <name>` | Consolidate memory, prune stale patterns | 🟡 |

</details>

<details>
<summary><b>Worlds · lifecycle</b> &middot; <code>███████ 7/7</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn world create <name> --agent <name>` | Declare a world in spwn.yaml | 🟢 |
| `spwn world rm <name>` | Remove a world declaration | 🟢 |
| `spwn world ls` | List declared worlds | 🟢 |
| `spwn world start [name]` | Start world(s); no-arg starts all | 🟢 |
| `spwn world stop [name]` | Stop world(s) | 🟢 |
| `spwn world <name>` | Shortcut for `world start <name>` | 🟢 |
| `spwn world rename <id> <name>` | Rename a running world | 🟢 |

</details>

<details>
<summary><b>Worlds · observe</b> &middot; <code>███ 3/3</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn world inspect <id>` | Composition + runtime state | 🟢 |
| `spwn world logs <id>` | Event log for a world | 🟢 |
| `spwn world enter <id>` | Interactive shell inside the world | 🟢 |

</details>

<details>
<summary><b>Worlds · snapshots</b> &middot; <code>▓▓▓▓ 0/4</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn world snap save <id>` | Save world state | 🟡 |
| `spwn world snap ls` | List snapshots | 🟡 |
| `spwn world snap restore <snap-id>` | Rollback to a snapshot | 🟡 |
| `spwn world snap rm <snap-id>` | Remove a snapshot | 🟡 |

</details>

<details>
<summary><b>Worlds · shared knowledge</b> &middot; <code>██ 2/2</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn world knowledge ls <id>` | List shared knowledge files | 🟢 |
| `spwn world knowledge show <id> <path>` | Read a knowledge file | 🟢 |

</details>

<details>
<summary><b>Dependencies &amp; authoring</b> &middot; <code>██████▓ 6/7</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn install spwn:<pkg>` | Install (adds to agents + lockfile) | 🟢 |
| `spwn uninstall spwn:<pkg>` | Remove a dep | 🟢 |
| `spwn inspect [agent]` | Per-agent composition tree | 🟢 |
| `spwn skill new <name>` | Author a new bare-markdown skill | 🟢 |
| `spwn skill show <name>` | Display a skill | 🟢 |
| `spwn skill rm <name>` | Delete a skill | 🟢 |
| `spwn skill edit <name>` | Open in `$EDITOR` | 🟡 |

</details>

<details>
<summary><b>Teams &amp; organizations</b> &middot; <code>▓▓▓▓▓▓ 0/6</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn team new <name>` | Create a team | 🟡 |
| `spwn team ls` | List teams | 🟡 |
| `spwn team assign <agent> <team>` | Attach agent to a team | 🟡 |
| `spwn team members <team>` | List a team's agents | 🟡 |
| `spwn organization ls` | List organizations | 🟡 |
| `spwn organization inspect <name>` | Show roles in an organization | 🟡 |

</details>

<details>
<summary><b>Architect daemon</b> &middot; <code>▓▓▓▓▓ 0/5</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn architect start` | Start the always-on daemon | 🟡 |
| `spwn architect stop` | Stop the daemon | 🟡 |
| `spwn architect status` | Show status and active worlds | 🟡 |
| `spwn architect talk "..."` | Talk to the Architect | 🟡 |
| `spwn architect logs` | Show the Architect's event log | 🟡 |

</details>

<details>
<summary><b>System</b> &middot; <code>██████▓ 6/7</code></summary>

| Command | Purpose | Status |
|---|---|:---:|
| `spwn status` | Global status (worlds, auth, version) | 🟢 |
| `spwn auth login` | Connect Anthropic / OpenAI | 🟢 |
| `spwn auth logout` | Clear cached credentials | 🟢 |
| `spwn auth token <value>` | Set a token directly (CI) | 🟢 |
| `spwn auth check` | Validate credentials across providers | 🟢 |
| `spwn upgrade` | Self-update the CLI | 🟢 |
| `spwn web` | Open the local web UI | 🟡 |

</details>

### Adapters

<details>
<summary><b>Agent runtimes</b> &middot; <code>█░░░░░░░░ 1/9</code></summary>

| Runtime | Status |
|---|:---:|
| Claude Code | 🟢 |
| Codex | 🔴 |
| Aider | 🔴 |
| Cline | 🔴 |
| Continue | 🔴 |
| OpenCode | 🔴 |
| Gemini CLI | 🔴 |
| Amazon Q | 🔴 |
| Goose | 🔴 |

</details>

<details>
<summary><b>LLM providers</b> &middot; <code>█▓░░░░░░ 1/8</code></summary>

| Provider | Status |
|---|:---:|
| Anthropic | 🟢 |
| OpenAI | 🟡 |
| Google | 🔴 |
| Mistral | 🔴 |
| Groq | 🔴 |
| Together | 🔴 |
| Ollama | 🔴 |
| AWS Bedrock | 🔴 |

</details>

<details>
<summary><b>World runtimes</b> &middot; <code>█░░░░░░ 1/7</code></summary>

| Runtime | Status |
|---|:---:|
| Docker | 🟢 |
| spwn Cloud | 🔴 |
| K3s | 🔴 |
| Firecracker | 🔴 |
| Fly.io | 🔴 |
| gVisor | 🔴 |
| Podman | 🔴 |

</details>

<details>
<summary><b>Memory backends</b> &middot; <code>█░░░░░ 1/6</code></summary>

| Backend | Status |
|---|:---:|
| Markdown filesystem | 🟢 |
| Chroma (RAG) | 🔴 |
| Qdrant | 🔴 |
| Pinecone | 🔴 |
| Weaviate | 🔴 |
| Turbopuffer | 🔴 |

</details>

<details>
<summary><b>Tool ecosystems</b> &middot; <code>█▓░░ 1/4</code></summary>

| Source | Status |
|---|:---:|
| `spwn:*` built-in catalog | 🟢 |
| Local project deps (`skill:`/`tool:`/`hook:`) | 🟡 |
| MCP servers | 🔴 |
| LangChain tools | 🔴 |

</details>

<details>
<summary><b>Orchestration</b> &middot; <code>▓░░░░░░ 0/7</code></summary>

| Orchestrator | Status |
|---|:---:|
| Built-in chief/worker hierarchy | 🟡 |
| Hermes | 🔴 |
| CrewAI | 🔴 |
| AutoGen | 🔴 |
| LangGraph | 🔴 |
| Swarm | 🔴 |
| Mastra | 🔴 |

</details>

<details>
<summary><b>Observability</b> &middot; <code>▓░░░░ 0/5</code></summary>

| Backend | Status |
|---|:---:|
| Web UI | 🟡 |
| Langfuse | 🔴 |
| LangSmith | 🔴 |
| Helicone | 🔴 |
| OpenTelemetry | 🔴 |

</details>

<br/>

## Documentation

| Topic | Link |
|---|---|
| **Recipes** — five worked examples that show spwn in action | [`docs/recipes.md`](docs/recipes.md) |
| **Dependency catalog** — the built-in `spwn:*` refs and how to author your own | [`docs/dependency-catalog.md`](docs/dependency-catalog.md) |
| **CLI reference** — every command, auto-generated | [`docs/cli/`](docs/cli/spwn.md) |
| **Contributing** — setup, testing, conventions | [`CONTRIBUTING.md`](CONTRIBUTING.md) |
| **Internals** — architecture, release runbook, update system | [`docs/contributing/`](docs/contributing/) |

<br/>

## Community

- [Website](https://spwn.sh) &middot; [Docs](https://spwn.sh/docs) &middot; [Manifesto](https://spwn.sh/manifesto) &middot; [Issues](https://github.com/jterrazz/spwn/issues)

---

<p align="center">
  <sub>Open source. Self-hosted. Built for people who want to give agents a world, not a wrapper.</sub>
</p>
