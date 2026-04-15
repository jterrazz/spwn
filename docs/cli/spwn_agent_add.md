---
title: "spwn agent add"
slug: "spwn-agent-add"
---

## spwn agent add

Add packages to an agent

### Synopsis

Compose an agent by attaching packages.

Examples:
  spwn agent add neo --package @spwn/python
  spwn agent add neo --pkg @spwn/mempalace
  spwn agent add neo --package @spwn/unix --package @spwn/git

```
spwn agent add <agent-name> [flags]
```

### Options

```
  -h, --help                  help for add
      --package stringArray   Package ref to add (repeatable, e.g. @spwn/python)
      --pkg stringArray       Short alias for --package
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

