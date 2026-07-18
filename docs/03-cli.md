# CLI

The `spwn` binary (`apps/cli`) is a thin surface over the domain packages: parse flags → call a domain API → format output. This chapter is the grammar and a task-oriented map. The **per-command reference is generated** from Cobra into [`cli/`](cli/) — regenerate it with `make docs`; never hand-edit those pages.

## Grammar

**`spwn <noun> <verb>`**, plus three top-level shortcuts (`up`, `ls`, `down`/`talk`) and name-only shortcuts (`spwn agent neo`, `spwn world default`). With no args, the shortcuts act on every world declared in `spwn.yaml`.

Design rules:

- Strict noun-first grammar. The only top-level verbs are the shortcuts `up`, `ls`, `talk`.
- `rm` is contextual: `spwn agent rm neo` deletes the agent; `spwn agent rm neo --dependency X` removes a dep from it.
- Inside a project, commands resolve against `./spwn/` first. Outside a project, they operate on user-level paths.

## Command map

```bash
# ── Project workflow ─────────────────────────────────────────────
spwn init                     # scaffold spwn.yaml + ./spwn/ + .spwn/
spwn check                    # validate the tree
spwn build --tree-only        # render the project tree to ./dist (preview/debug)
spwn build                    # transpile + compile into a project-specific Docker image
spwn up                       # spawn a world from the current project

# ── Compose-style shortcuts ──────────────────────────────────────
spwn up [name]                # bring up every world (or one by name)
spwn agent neo                # start the world that contains neo
spwn ls                       # agent-centric status (running/stopped/orphan)
spwn down                     # stop every world

# ── Agents ───────────────────────────────────────────────────────
spwn agent new neo            # create a blank agent in ./spwn/agents/
spwn agent ls                 # list project agents
spwn agent inspect neo        # inspect composition, memory, history
spwn agent fork neo neo-v2    # clone memory + composition
spwn agent rm neo             # delete an agent
spwn agent talk  neo "…"      # interactive session (full form of `spwn talk`)
spwn agent send  neo "…" --from morpheus   # async message to an agent's inbox
spwn agent inbox neo          # show neo's inbox
spwn agent watch neo          # tail neo's inbox live
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
spwn world start [name]       # start a world (no arg: every world in spwn.yaml)
spwn world stop  [name]       # stop a world
spwn world ls                 # list running worlds
spwn world inspect <id>       # inspect a running world
spwn world enter   <id>       # interactive shell inside the world
spwn world snap save|ls|restore|rm   # world snapshots

# ── Automations ──────────────────────────────────────────────────
spwn automation ls            # list declared automations + last-fired
spwn automation status        # per-automation rollup (fires/ok/fail)
spwn automation logs [-f] [-n N]   # tail .spwn/runs.jsonl receipts
spwn automation daemon        # run the engine until interrupted

# ── System ───────────────────────────────────────────────────────
spwn architect start|stop|status|talk|logs   # always-on orchestration daemon
spwn gate start|stop|status   # host-side broker for cookie-bearing tools (see 06-gate)
spwn web                      # open the local web UI
spwn auth login|logout|token  # provider credentials

# ── Registry (planned) ───────────────────────────────────────────
spwn agent   get github:community/sci   # install a shared agent   [planned]
spwn install github:acme/fuzzer         # install from GitHub      [planned]
spwn *       publish <name>             # push to registry         [planned]
```

## Related

- [`cli/`](cli/) — the generated per-command reference (regenerate with `make docs`).
- [Concepts](02-concepts.md) — what the nouns mean.
- [Primitives](04-primitives.md) — the `install` targets (`spwn:`, `skill/`, `tool/`, `hook/`, `command/`).
- [Automations](automations.md) — the automation subsystem in depth.
