---
title: "spwn agent remove"
slug: "spwn-agent-remove"
---

## spwn agent remove

Remove packages from an agent

### Synopsis

Remove packages from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent remove <name> --package X' removes just that entry.

Examples:
  spwn agent remove neo --package @spwn/python
  spwn agent remove neo --pkg @spwn/mempalace

```
spwn agent remove <agent-name> [flags]
```

### Options

```
  -h, --help                  help for remove
      --package stringArray   Package ref to remove (repeatable)
      --pkg stringArray       Short alias for --package
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

