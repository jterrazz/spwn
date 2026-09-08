---
title: "spwn world snap restore"
slug: "spwn-world-snap-restore"
---

## spwn world snap restore

Restore a world from a snapshot

### Synopsis

Creates a new world from a previously saved snapshot. The snapshot format is: w-{id}--{name}

```
spwn world snap restore <snapshot> [flags]
```

### Options

```
  -a, --agent string            Agent name (omit for an empty world)
  -c, --config string           Named world config (default: default)
  -h, --help                    help for restore
  -w, --workspace stringArray   Host dir to mount. Repeatable: "path", "name=path", "name=path:ro". Omit for ephemeral.
```

### SEE ALSO

* [spwn world snap](./spwn_world_snap.md)	 - World snapshots - save, ls, restore, rm

