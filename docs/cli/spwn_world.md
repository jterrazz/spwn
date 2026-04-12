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
  spwn world --agent neo -w .                  Single agent in current directory
  spwn world --agent morpheus --agent neo -w .  Multi-agent (morpheus is chief)
  spwn world --name "Big Refactor" --agent neo  Ephemeral (no host mount)
  spwn world --no-agent                        Empty world (no agent)
```

### Options

```
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
  -c, --config string           Named world config (default: default)
      --gate stringArray        Bridge tool from Host: "source:as:cap1,cap2"
  -h, --help                    help for world
  -i, --interactive             Attach to agent interactively
  -n, --name string             Display name for the world
      --no-agent                Create the world without spawning an agent
      --runtime string          Agent runtime (claude-code, pi, codex, opencode, gemini, aider) (default "claude-code")
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
* [spwn world list](./spwn_world_list.md)	 - List all active worlds
* [spwn world logs](./spwn_world_logs.md)	 - Show agent output from a world
* [spwn world rename](./spwn_world_rename.md)	 - Rename a world (omit name to clear)
* [spwn world restore](./spwn_world_restore.md)	 - Restore a world from a snapshot
* [spwn world show](./spwn_world_show.md)	 - Show world details and agent status
* [spwn world snapshot](./spwn_world_snapshot.md)	 - Save a running world as a snapshot
* [spwn world snapshots](./spwn_world_snapshots.md)	 - List all world snapshots

