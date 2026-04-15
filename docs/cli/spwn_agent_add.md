---
title: "spwn agent add"
slug: "spwn-agent-add"
---

## spwn agent add

Add tools or skills to an agent

### Synopsis

Compose an agent by attaching reusable blocks.

Examples:
  spwn agent add neo --tool @spwn/python
  spwn agent add neo --plugin @spwn/mempalace
  spwn agent add neo --skill paper-reading --skill refactoring
  spwn agent add neo --tool @spwn/unix --tool @spwn/git

```
spwn agent add <agent-name> [flags]
```

### Options

```
  -h, --help                 help for add
      --plugin stringArray   Plugin pack to add (repeatable, e.g. @spwn/mempalace)
      --skill stringArray    Skill to add (repeatable)
      --tool stringArray     Tool pack to add (repeatable, e.g. @spwn/python)
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

