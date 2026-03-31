---
title: "spwn world watch"
slug: "spwn-world-watch"
---

## spwn world watch

Watch for new messages in a world

### Synopsis

Run in the foreground, polling inbox directories every 5 seconds.
When new unread messages are found, prints a notification and wakes
the recipient agent via 'spwn agent talk'.

```
spwn world watch <world-id> [flags]
```

### Options

```
  -h, --help   help for watch
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn world](/docs/cli/spwn-world)	 - Spawn a world — an isolated reality for agents

