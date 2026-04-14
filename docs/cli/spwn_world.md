---
title: "spwn world"
slug: "spwn-world"
---

## spwn world

Manage worlds - ephemeral runtime instances for agents

```
spwn world [name] [flags]
```

### Options

```
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
  -c, --config string           Named world config (default: default)
  -h, --help                    help for world
  -i, --interactive             Drop into the agent's session after spawn
  -n, --name string             Display name for the world
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn world create](./spwn_world_create.md)	 - Declare a new world in spwn.yaml
* [spwn world destroy](./spwn_world_destroy.md)	 - Destroy a world
* [spwn world enter](./spwn_world_enter.md)	 - Open an interactive shell inside a running world
* [spwn world inspect](./spwn_world_inspect.md)	 - Inspect a running world - agents, workspaces, status
* [spwn world knowledge](./spwn_world_knowledge.md)	 - Read a world's shared knowledge
* [spwn world list](./spwn_world_list.md)	 - List declared worlds and their running status
* [spwn world logs](./spwn_world_logs.md)	 - Show the event log for a specific world
* [spwn world rename](./spwn_world_rename.md)	 - Rename a world (omit name to clear)
* [spwn world rm](./spwn_world_rm.md)	 - Remove a world declaration from spwn.yaml
* [spwn world snap](./spwn_world_snap.md)	 - World snapshots - save, ls, restore, rm
* [spwn world start](./spwn_world_start.md)	 - Start a world (alias for `spwn up`)
* [spwn world stop](./spwn_world_stop.md)	 - Stop a world (alias for `spwn down`)
* [spwn world up](./spwn_world_up.md)	 - Spawn a world - an isolated reality for agents

