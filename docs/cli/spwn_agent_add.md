---
title: "spwn agent add"
slug: "spwn-agent-add"
---

## spwn agent add

Add packs to an agent

### Synopsis

Compose an agent by attaching packs.

Examples:
  spwn agent add neo --pack @spwn/python
  spwn agent add neo --packs @spwn/unix --packs @spwn/git
  spwn agent add neo --pack @spwn/unix --pack @spwn/git

```
spwn agent add <agent-name> [flags]
```

### Options

```
  -h, --help                help for add
      --pack stringArray    Pack ref to add (repeatable, e.g. @spwn/python)
      --packs stringArray   Plural alias for --pack
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

