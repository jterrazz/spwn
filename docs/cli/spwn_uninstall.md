---
title: "spwn uninstall"
slug: "spwn-uninstall"
---

## spwn uninstall

Remove a dependency from the project

### Synopsis

Remove a dependency from agent manifests. When no agent still carries the
ref, the lockfile pin is dropped too.

Without --agent, the ref is removed from every agent. Pass --agent <name>
to detach it from a single agent while leaving others untouched.

Examples:
  spwn uninstall python                     # every agent
  spwn uninstall skill:refine --agent mark  # only mark

```
spwn uninstall <ref> [flags]
```

### Options

```
      --agent string   Target a single agent instead of every agent in the project
  -h, --help           help for uninstall
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

