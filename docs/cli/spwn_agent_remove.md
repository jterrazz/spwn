---
title: "spwn agent remove"
slug: "spwn-agent-remove"
---

## spwn agent remove

Remove dependencies from an agent

### Synopsis

Remove dependencies from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent remove <name> --dep X' removes just that entry.

Examples:
  spwn agent remove neo --dep spwn:python
  spwn agent remove neo --deps spwn:mempalace

```
spwn agent remove <agent-name> [flags]
```

### Options

```
      --dep stringArray    Dependency ref to remove (repeatable)
      --deps stringArray   Plural alias for --dep
  -h, --help               help for remove
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

