# Getting started

What a spwn project holds, and the four ideas the rest of the manual builds on. The front page, and why spwn exists, is the [README](../../README.md); worked sessions are [Use cases](use-cases.md).

## The layout

Install spwn and bring up neo as the [quickstart](../../README.md#quickstart) shows. Here's what `spwn init` lays down:

```text
my-project/
├── spwn.yaml              # manifest (the thing that ties everything together)
├── spwn.lock              # lockfile (pinned catalog deps)
├── spwn/                  # committed project assets
│   ├── agents/            # one subdir per agent
│   ├── skills/            # reusable skill files (markdown blocks)
│   ├── tools/             # local tool definitions
│   └── hooks/             # shell hooks the runtime fires
├── knowledge/             # opt-in world-scoped knowledge base
└── .spwn/                 # gitignored local state
```

## Four ideas

spwn turns your repo into a **portable agent artifact**, consumed by the spwn CLI today, and by web UIs, apps, and embedded SDKs on the roadmap. One bundle format, many future surfaces.

```text
   repo  ──▶  spwn build  ──▶  artifact  ──▶  anywhere
```

Four ideas to hold in your head before you dive in:

- **[One file, one agent](#one-file-one-agent)**: `agent.yaml` lists the runtime, tools, skills, hooks. Human-readable. Git-friendly. No database.
- **[Lives in your repo, not a SaaS](#lives-in-your-repo-not-a-saas)**: every agent is a folder in your project. Commit it, review it, diff it like any other code.
- **[A world is one `spwn up` away](#a-world-is-one-spwn-up-away)**: group agents, workspaces, and knowledge; launch them together; tear them down together.
- **[Runtime-agnostic](#runtime-agnostic)**: Claude Code and Codex ship today. Swap backends with one line.

### One file, one agent

An agent **is** a composition of blocks, declared in one file:

```yaml
# spwn/agents/neo/agent.yaml
name: neo
runtime:
    backend: 'spwn:claude-code'

dependencies:
    - 'spwn:unix' # catalog: shell + coreutils
    - 'spwn:python' # catalog: python 3 + pip
    - 'skill/code-review' # local:   ./spwn/skills/code-review.md
    - 'tool/greet' # local:   ./spwn/tools/greet/
    - 'hook/welcome' # local:   ./spwn/hooks/welcome.yaml
    - 'command/refactor' # local:   ./spwn/commands/refactor.md
```

**Every dependency declares its source and type explicitly.** Two source-prefixed schemes (`spwn:`, `github:`) plus four local path-style forms (`skill/`, `tool/`, `hook/`, `command/`):

| Scheme                  | Resolves to                                   |
| ----------------------- | --------------------------------------------- |
| `spwn:<name>`           | Built-in catalog dep compiled into the binary |
| `github:<owner>/<repo>` | Community registry _(planned)_                |
| `skill/<name>`          | `./spwn/skills/<name>.md`                     |
| `tool/<name>`           | `./spwn/tools/<name>/` (with `tool.yaml`)     |
| `hook/<name>`           | `./spwn/hooks/<name>.yaml`                    |
| `command/<name>`        | `./spwn/commands/<name>.md`                   |

Add one with `spwn install <ref> --agent neo`: the ref lands in `agent.yaml` and pins in `spwn.lock`. Browse the full [dependency catalog](../reference/dependency-catalog.md).

The rest of the agent directory sits next to the manifest. Identity and memory live as plain files:

```text
spwn/agents/neo/
├── agent.yaml       # composition (the file above)
├── SOUL.md          # identity (who the agent is)
├── AGENTS.md        # boot-time prompt (what it should do)
├── playbooks/       # memory: procedures the agent has learned
└── journal/         # memory: session history
```

### Lives in your repo, not a SaaS

**Your agents and their composition are declarative files committed alongside your code** - reviewed in PRs, versioned in git, diffed like any other config. Think Terraform for infrastructure, `docker-compose.yaml` for services, `package.json` for dependencies. Spwn plays the same role for the agents that work on your repo.

`spwn init` drops the scaffold into any directory, the way `git init` or `docker init` do; see [the layout](#the-layout) it lays down. Whoever clones the repo gets the same agents with the same tools, byte-for-byte. No imperative setup scripts, no "works on my machine".

**`~/.spwn/` holds only your user identity** - credentials, daemon state, activity log. It's the equivalent of `~/.aws/` or `~/.docker/config.json`: personal to the machine, never the source of truth for what runs. To share an agent across projects, export it (`spwn agent export`) and import the archive in the next repo (`spwn agent import`).

### A world is one `spwn up` away

An agent defines **what** can think. A **world** defines _where_ and _with whom_ they run. Worlds are the runtime unit: one long-running container per world, one shared filesystem, one declared set of agents talking to each other and to the mounted workspace.

Worlds live **inline** under `spwn.yaml#worlds:`. Each entry names the agents it deploys, the workspaces it mounts, and the optional knowledge base it exposes.

```yaml
# spwn.yaml
version: 1
name: acme-api

worlds:
    matrix:
        agents: [neo]
        workspaces: [.] # host paths mounted under /workspaces/. Use `name=path` to name them.
        knowledge: ./spwn/knowledge # optional; bind into /world/knowledge/. Omit for no mount.
```

`spwn up` materialises every world in the manifest; `spwn down` tears them down. A single agent can appear in many worlds; each world keeps its own runtime state (sessions, inbox, shared scratchpad), separate from the agent's long-lived memory on disk. Destroying a world doesn't destroy the agent.

### Runtime-agnostic

Think of spwn the way you think of `tsc` or `babel`. You write in one clean, provider-neutral source; a transpiler adapts it to whatever runtime you target and emits exactly what that runtime expects. You never touch the output by hand.

```text
   YOUR REPO             BUILD                ARTIFACT
  ───────────          ─────────             ──────────────────────────
   spwn.yaml                                ┌──▶  Docker image
   spwn/agents/         spwn build          │     (push, pull, run anywhere)
   spwn/skills/    ──▶  transpile     ──▶  ─┤
   spwn/tools/          + compile           │
   spwn/hooks/                              └──▶  runtime-native tree
                                                  (claude-code, codex; no Docker)
```

- **Source** is provider-neutral. `AGENTS.md`, `SOUL.md`, `skills/`, `agent.yaml` - nothing in your repo mentions Claude Code, Codex, or any runtime by name.
- **Transpile** renders that source into the exact file layout your chosen runtime expects. Claude Code wants `CLAUDE.md` in a particular place? The claude-code backend emits it. Codex wants something else? Its backend emits that. Same source, different targets - like transpiling TypeScript to ES5 vs ES2022.
- **Compile** links the transpiled tree with the tools your agent declared and produces a normal Docker image. Push it, pull it, run it anywhere - byte-identical on every machine.

`spwn check` is the type-checker: it runs the transpile step in dry-run to catch broken imports, missing skills, and invalid tool refs before you ever touch Docker.

Switching runtimes is a one-line change in `agent.yaml` - no source edits, no lock-in. See [`packages/transpile/README.md`](../../packages/transpile/README.md) for internals and how to add a new backend.

<br/>

## Where next

- [Use cases](use-cases.md): four sessions, end to end.
- [Primitives](../reference/primitives.md): every file of a project, field by field.
- [The CLI](../reference/cli.md): the `spwn <noun> <verb>` grammar, and [every command](../reference/commands.md).
