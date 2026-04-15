---
title: "spwn agent add"
slug: "spwn-agent-add"
---

## spwn agent add

Add plugins to an agent

### Synopsis

Compose an agent by attaching plugins.

Examples:
  spwn agent add neo --plugin @spwn/python
  spwn agent add neo --plugins @spwn/unix --plugins @spwn/git
  spwn agent add neo --plugin @spwn/unix --plugin @spwn/git

```
spwn agent add <agent-name> [flags]
```

### Options

```
  -h, --help                  help for add
      --plugin stringArray    Plugin ref to add (repeatable, e.g. @spwn/python)
      --plugins stringArray   Plural alias for --plugin
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

