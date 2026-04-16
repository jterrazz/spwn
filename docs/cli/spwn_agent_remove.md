---
title: "spwn agent remove"
slug: "spwn-agent-remove"
---

## spwn agent remove

Remove packs from an agent

### Synopsis

Remove packs from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent remove <name> --pack X' removes just that entry.

Examples:
  spwn agent remove neo --pack @spwn/python
  spwn agent remove neo --packs @spwn/mempalace

```
spwn agent remove <agent-name> [flags]
```

### Options

```
  -h, --help                help for remove
      --pack stringArray    Pack ref to remove (repeatable)
      --packs stringArray   Plural alias for --pack
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

