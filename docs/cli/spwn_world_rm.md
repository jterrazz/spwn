---
title: "spwn world rm"
slug: "spwn-world-rm"
---

## spwn world rm

Remove a world declaration from spwn.yaml

### Synopsis

Remove the worlds.<name> entry from spwn.yaml.

This only edits config - it does NOT stop a running container. If
the world is currently running, stop it first with "spwn world stop
<name>".

The agents listed by the world stay on disk; their minds are
preserved. Other worlds may still reference them.

```
spwn world rm <name> [flags]
```

### Options

```
  -h, --help   help for rm
```

### SEE ALSO

* [spwn world](./spwn_world.md)	 - Manage worlds - ephemeral runtime instances for agents

