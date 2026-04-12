---
title: "spwn agent add"
slug: "spwn-agent-add"
---

## spwn agent add

Add tools, skills, or a profile to an agent

### Synopsis

Compose an agent by attaching reusable blocks.

Examples:
  spwn agent add neo --tool @spwn/python
  spwn agent add neo --skill paper-reading --skill refactoring
  spwn agent add neo --profile researcher
  spwn agent add neo --tool @spwn/unix --tool @spwn/git --profile dev

```
spwn agent add <agent-name> [flags]
```

### Options

```
  -h, --help                help for add
      --profile string      Profile template to apply
      --skill stringArray   Skill to add (repeatable)
      --tool stringArray    Tool pack to add (repeatable, e.g. @spwn/python)
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent — a living identity that inhabits a world

