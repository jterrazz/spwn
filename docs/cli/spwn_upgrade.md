---
title: "spwn upgrade"
slug: "spwn-upgrade"
---

## spwn upgrade

Upgrade spwn to the latest version

### Synopsis

Downloads and installs the latest spwn release from GitHub.

Fetches the release binary for your OS/architecture, verifies its SHA256
against the checksums published with the release, and atomically replaces
the current binary. Running worlds are stopped gracefully before the swap.

```
spwn upgrade [flags]
```

### Examples

```
  spwn upgrade              # install the latest stable release
  spwn upgrade --check      # just check, don't install
  spwn upgrade --channel beta
  spwn upgrade --force      # reinstall current version
```

### Options

```
      --channel string   Release channel: stable or beta (default "stable")
      --check            Check for updates but do not install
      --force            Install even if already up to date
  -h, --help             help for upgrade
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn — create realities for things that can think

