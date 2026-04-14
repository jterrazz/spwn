---
title: "spwn init"
slug: "spwn-init"
---

## spwn init

Scaffold a spwn project in the current directory

### Synopsis

Scaffold a spwn project in the current directory.

Creates spwn.yaml and a committed ./spwn/ tree containing a default
world and a default agent. Adds .spwn/ to .gitignore for local state.

Use --global to instead seed ~/.spwn/ with a world config (legacy
user-home mode, kept for backward compatibility).

```
spwn init [flags]
```

### Options

```
  -f, --force         Overwrite existing spwn.yaml
      --global        Initialise ~/.spwn/ (legacy user-home mode)
  -h, --help          help for init
      --name string   Project name (default: current directory name)
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

