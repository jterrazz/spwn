---
title: "spwn agent remove"
slug: "spwn-agent-remove"
---

## spwn agent remove

Remove tools, skills, or profile from an agent

### Synopsis

Remove composable blocks from an agent's composition.

Note: 'spwn agent rm <name>' (without flags) deletes the entire agent.
'spwn agent remove <name> --tool X' removes just that block.

Examples:
  spwn agent remove neo --tool @spwn/python
  spwn agent remove neo --skill paper-reading
  spwn agent remove neo --profile

```
spwn agent remove <agent-name> [flags]
```

### Options

```
  -h, --help                help for remove
      --profile             Clear the agent's profile attachment
      --skill stringArray   Skill to remove (repeatable)
      --tool stringArray    Tool pack to remove (repeatable)
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent — a living identity that inhabits a world

