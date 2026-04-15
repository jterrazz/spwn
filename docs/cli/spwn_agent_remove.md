---
title: "spwn agent remove"
slug: "spwn-agent-remove"
---

## spwn agent remove

Remove tools or skills from an agent

### Synopsis

Remove composable blocks from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent remove <name> --tool X' removes just that block.

Examples:
  spwn agent remove neo --tool @spwn/python
  spwn agent remove neo --plugin @spwn/mempalace
  spwn agent remove neo --skill paper-reading

```
spwn agent remove <agent-name> [flags]
```

### Options

```
  -h, --help                 help for remove
      --plugin stringArray   Plugin pack to remove (repeatable)
      --skill stringArray    Skill to remove (repeatable)
      --tool stringArray     Tool pack to remove (repeatable)
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

