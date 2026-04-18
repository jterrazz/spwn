---
title: "spwn init"
slug: "spwn-init"
---

## spwn init

Scaffold a spwn project in the current directory

### Synopsis

Scaffold a spwn project in the current directory.

Without arguments, creates a blank spwn.yaml plus a default ./spwn/
tree (one world, one agent) and adds .spwn/ to .gitignore.

A positional example ref installs one of the bundled gallery entries
into the current directory. Bare names resolve through the catalog:

    spwn init matrix          # shorthand for spwn init spwn:matrix
    spwn init spwn:matrix     # explicit form

Use --global to instead seed ~/.spwn/ with a world config (legacy
user-home mode, kept for backward compatibility).

```
spwn init [example-ref] [flags]
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

