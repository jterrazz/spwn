---
title: "spwn tool"
slug: "spwn-tool"
---

## spwn tool

Manage reusable tool packs (e.g. @spwn/unix, @spwn/python)

### Synopsis

Tool packs are composable building blocks that agents plug into their worlds.

Attach one to an agent with:
  spwn agent add <agent> --tool <pack>

If a tool isn't in an agent's composition, it's physically absent from that
agent's world — not forbidden, absent.

### Options

```
  -h, --help   help for tool
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn — create realities for things that can think
* [spwn tool install](./spwn_tool_install.md)	 - Install a tool pack from the registry
* [spwn tool ls](./spwn_tool_ls.md)	 - List installed tool packs
* [spwn tool publish](./spwn_tool_publish.md)	 - Publish a tool pack to the registry
* [spwn tool rm](./spwn_tool_rm.md)	 - Remove an installed tool pack
* [spwn tool search](./spwn_tool_search.md)	 - Search the tool registry
* [spwn tool show](./spwn_tool_show.md)	 - Inspect a tool pack

