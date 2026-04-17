---
title: "spwn agent add"
slug: "spwn-agent-add"
---

## spwn agent add

Add dependencies to an agent

### Synopsis

Compose an agent by attaching catalog.

Examples:
  spwn agent add neo --dep spwn:python
  spwn agent add neo --deps spwn:unix --deps spwn:git
  spwn agent add neo --dep spwn:unix --dep spwn:git

```
spwn agent add <agent-name> [flags]
```

### Options

```
      --dep stringArray    Dependency ref to add (repeatable, e.g. spwn:python)
      --deps stringArray   Plural alias for --dep
  -h, --help               help for add
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

