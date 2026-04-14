---
title: "spwn build"
slug: "spwn-build"
---

## spwn build

Flatten the project into a reproducible build artifact

### Synopsis

Flatten the project into .spwn/build/ - every agent file, the world
config, and a normalized manifest, all copied into one self-contained
tree that spwn up can consume directly.

Runs spwn check first (unless --skip-validate is set). Errors abort
the build; warnings are printed but not blocking.

```
spwn build [flags]
```

### Options

```
  -h, --help            help for build
      --skip-validate   Build even if spwn check finds errors
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

