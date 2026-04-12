---
title: "spwn world"
slug: "spwn-world"
---

## spwn world

Spawn a world — an isolated reality for agents

### Synopsis

Spawn a world — the Big Bang.

Creates an isolated Docker environment and brings an agent to life inside it.
Uses a named world config from ~/.spwn/worlds/ (default: default.yaml).

```
spwn world [flags]
```

### Examples

```
  spwn world -w .                         Spawn with current directory
  spwn world -w web=./frontend -w api=./backend   Multi-workspace
  spwn world -w docs=./docs:ro -w code=./src      Read-only reference
  spwn world --name "Big Refactor"        Ephemeral (no host mount)
  spwn world --leader morpheus            With a leader agent
  spwn world --no-agent                   Empty world (no agent)
```

### Options

```
  -a, --agent string            Agent name (default "default")
  -c, --config string           Named world config (default: default)
      --gate stringArray        Bridge tool from Host: "source:as:cap1,cap2"
  -h, --help                    help for world
  -i, --interactive             Attach to agent interactively
      --leader string           Leader agent for this world (gets the top role in the organization)
  -n, --name string             Display name for the world
      --no-agent                Create the world without spawning an agent
      --organization string     Organization to use for role assignment (default "default")
      --runtime string          Agent runtime (claude-code, pi, codex, opencode, gemini, aider) (default "claude-code")
      --team string             Deploy all agents in a team (team slug)
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn — create realities for things that can think
* [spwn world attach](./spwn_world_attach.md)	 - Open interactive session into a running world
* [spwn world destroy](./spwn_world_destroy.md)	 - Destroy a world
* [spwn world inspect](./spwn_world_inspect.md)	 - Show world details, physics, and agent status
* [spwn world list](./spwn_world_list.md)	 - List all active worlds
* [spwn world logs](./spwn_world_logs.md)	 - Show agent output from a world
* [spwn world rename](./spwn_world_rename.md)	 - Rename a world (omit name to clear)
* [spwn world restore](./spwn_world_restore.md)	 - Restore a world from a snapshot
* [spwn world snapshot](./spwn_world_snapshot.md)	 - Save a running world as a snapshot
* [spwn world snapshots](./spwn_world_snapshots.md)	 - List all world snapshots

