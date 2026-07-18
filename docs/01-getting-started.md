# Getting started

spwn is the operating system for autonomous agent worlds: compose tools, skills, and identity into agents, then spawn them into isolated Docker worlds where they wake up, find their tools, and get to work. A spwn project lives **in your repo**, not in a SaaS — every agent is a folder you commit, review, and diff like any other code.

This chapter gets you from install to a running agent. For the model behind the words (Agent, World, Architect, Mind…) read [Concepts](02-concepts.md); for the CLI surface read [CLI](03-cli.md).

## Install

```bash
curl -fsSL https://spwn.sh/install.sh | bash
```

Requirements: **Docker** (worlds are containers). For building from source see [`../CONTRIBUTING.md`](../CONTRIBUTING.md).

## Three commands, one agent

| Step | Command | What it does |
| ---- | ------- | ------------ |
| Log in | `spwn auth` | Confirms you are signed in to Claude Code (or another supported runtime). |
| Scaffold | `spwn init` | Drops `spwn.yaml` + a starter `neo` agent into the current directory. |
| Talk | `spwn agent neo` | Opens an interactive session with `neo` inside a sandboxed Docker world; container lifecycle is handled for you. |

`spwn init <template>` (e.g. `spwn init matrix`) drops a ready-made multi-agent world instead — swap the slug for any entry in [`../catalog/`](../catalog/).

## What lands in your project

A spwn project is per-repo. `~/.spwn/` holds only user-level credentials and daemon state.

```
my-project/
├── spwn.yaml                    # manifest — version, name, inline worlds map, project-wide deps
├── spwn.lock                    # lockfile — pinned catalog deps
├── spwn/                        # committed project assets
│   ├── agents/
│   │   └── neo/
│   │       ├── agent.yaml        # composition: dependencies + runtime.backend
│   │       ├── AGENTS.md          # provider-neutral entry point (compiled per runtime)
│   │       ├── SOUL.md            # who the agent is (purpose, voice, values)
│   │       ├── playbooks/         # promoted patterns (auto-indexed from name:/description: headers)
│   │       └── journal/           # per-run history
│   ├── knowledge/                # world-scoped facts, bind-mounted at /world/knowledge/ (default path)
│   ├── skills/                   # project-scoped skills   (skill/<name> → spwn/skills/<name>.md)
│   ├── tools/                    # project-scoped tools    (tool/<name>  → spwn/tools/<name>/)
│   ├── hooks/                    # project-scoped hooks    (hook/<name>  → spwn/hooks/<name>.yaml)
│   └── commands/                 # project-scoped commands (command/<name> → spwn/commands/<name>.md)
└── .spwn/                        # gitignored local state
    ├── state.json               # live world IDs bound to this project
    ├── runs.jsonl               # automation receipts (one line per fire)
    ├── automations/state.json   # last-fired cursor per automation (catch-up math)
    └── cache/
```

```
~/.spwn/                         # USER-LEVEL only, never per-project
├── credentials/                 # auth material surfaced to containers at /credentials
├── activity.jsonl               # global activity log
└── state/                       # architect daemon state
```

## Config hierarchy

`spwn.yaml` is the manifest; `agent.yaml` is each agent's composition. The two compose:

- **`spwn.yaml`** declares project-wide `dependencies:`, an optional `runtime.backend` default, and the inline `worlds:` map (each world names its agents, workspace mounts, optional `knowledge:` path, and optional tool overrides). Worlds are inline map entries — there is no `spwn/worlds/` directory.
- **`agent.yaml`** declares one agent's `dependencies:` list and `runtime.backend`. Its deps are **unioned** with the project-wide pool — an agent cannot remove a project-level dep, only add to it.

The union of project-wide and agent-specific dependencies is exactly what materializes inside that agent's container. Full field reference and the dependency grammar are in [Primitives](04-primitives.md).

## Everyday workflow

```bash
spwn init            # scaffold spwn.yaml + ./spwn/ + .spwn/
spwn check           # validate the tree (bad refs, missing files, lockfile drift)
spwn build           # transpile + compile into a project-specific Docker image
spwn up              # spawn a world from the current project
spwn ls              # agent-centric status (running / stopped / orphan)
spwn down            # stop every world
```

`spwn build --tree-only` renders the project tree to `./dist` for preview/debug without building an image. The complete command surface is in [CLI](03-cli.md).

## Related

- [Concepts](02-concepts.md) — the world/agent model and vocabulary.
- [Primitives](04-primitives.md) — `spwn.yaml`, agents, tools, skills, hooks, commands.
- [Automations](automations.md) — waking agents on cron or filesystem triggers.
- [Recipes](recipes.md) — worked examples.
