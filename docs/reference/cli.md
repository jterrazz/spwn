# CLI

The `spwn` binary (`apps/cli`) is a thin surface over the domain packages: parse flags, call a domain API, format output. This chapter is the grammar and a task-oriented map. The **per-command reference is generated** from Cobra into [`reference/`](./); regenerate it with `make docs`, never hand-edit those pages.

## Grammar

**`spwn <noun> <verb>`**, plus two top-level verbs (`up`, `down`) that bring every world declared in `spwn.yaml` up and down, and one name-only shortcut: `spwn agent neo` starts the world that holds `neo` and opens a session with it.

Design rules:

- Strict noun-first grammar. The only top-level verbs are `up` and `down`, and no command has an alias.
- `rm` is contextual: `spwn agent rm neo` deletes the agent; `spwn agent rm neo --dependency X` removes a dep from it.
- Inside a project, commands resolve against `./spwn/` first. Outside a project, they operate on user-level paths.
- Names for creation, IDs for reference: creation takes a name, and a running instance is addressed by the ID it was given ([Concepts](../concepts/domain-model.md#ids)).
- `--json` wherever output is worth consuming programmatically.

Two invariants hold across the surface. Commands speak the domain's vocabulary (worlds, agents, souls, dreams), never the backend's. And the manifest is the input while the running world is the output: no command mutates a world image in place, so changing reality means changing `spwn.yaml` or a block file and rebuilding.

## Command map

```bash
# ── Project workflow ─────────────────────────────────────────────
spwn init                     # scaffold spwn.yaml + ./spwn/ + .spwn/
spwn check                    # validate the tree
spwn build --tree-only        # render the project tree to ./dist (preview/debug)
spwn build                    # transpile + compile into a project-specific Docker image
spwn up [name]                # bring up every world in spwn.yaml (or one by name)
spwn agent neo                # spawn the world that contains neo and open a session
spwn down [name]              # stop every world (or one)

# ── Agents ───────────────────────────────────────────────────────
spwn agent create neo         # create a blank agent in ./spwn/agents/
spwn agent ls                 # list project agents
spwn agent inspect neo        # inspect composition, memory, history
spwn agent fork neo neo-v2    # clone memory + composition
spwn agent rm neo             # delete an agent
spwn agent talk  neo "..."    # interactive session
spwn agent dream neo          # analyze experience, promote playbooks
spwn agent sleep neo          # consolidate memory, prune stale patterns

# ── Dependencies (compose) ───────────────────────────────────────
spwn install python                            # catalog dep, every agent
spwn install python --agent neo                # catalog dep, only neo
spwn install skill/paper-reading --agent neo   # local skill, only neo
spwn install tool/ffmpeg --agent neo           # local tool, only neo
spwn install hook/pre-spawn --agent neo        # local hook, only neo
spwn install command/refactor --agent neo      # local command, only neo
spwn uninstall python --agent neo              # detach from one agent
spwn skill new|edit|show|rm <name>             # bare-markdown skill authoring

# ── Worlds ───────────────────────────────────────────────────────
spwn world list               # list running worlds
spwn world inspect <id>       # inspect a running world
spwn world enter   <id>       # interactive shell inside the world
spwn world snap save|ls|restore|rm   # world snapshots

# ── Automations ──────────────────────────────────────────────────
spwn automation ls            # list declared automations + last-fired
spwn automation status        # per-automation rollup (fires/ok/fail)
spwn automation logs [-f] [-n N]   # tail .spwn/runs.jsonl receipts
spwn automation daemon        # run the engine until interrupted

# ── System ───────────────────────────────────────────────────────
spwn gate start|stop|status   # host-side broker for cookie-bearing tools (see the Gate concept)
spwn web                      # open the local web UI
spwn auth login|logout|token  # provider credentials

# ── Registry (planned) ───────────────────────────────────────────
spwn install github:acme/fuzzer         # install from GitHub      [planned]
```

## Related

- [`reference/`](./): the generated per-command reference (regenerate with `make docs`).
- [Concepts](../concepts/domain-model.md): what the nouns mean.
- [Primitives](primitives.md): the `install` targets (`spwn:`, `skill/`, `tool/`, `hook/`, `command/`).
- [Automations](../guides/automations.md): the automation subsystem in depth.
