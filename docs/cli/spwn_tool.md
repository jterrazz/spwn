---
title: "spwn tool"
slug: "spwn-tool"
---

## spwn tool

Manage reusable tool packs (e.g. @spwn/unix, @spwn/python)

### Synopsis

Tool packs are composable building blocks that agents plug into their worlds.

Install a catalog pack into the project's agents + lockfile with:
  spwn tool install @spwn/python

Remove it with:
  spwn tool uninstall @spwn/python

List what's installed with:
  spwn tool ls

Local tool packs authored under spwn/tools/<name>/ are referenced by
bare name in agent.yaml and do NOT go through the install verb — they
are authored in place.

### Options

```
  -h, --help   help for tool
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn tool install](./spwn_tool_install.md)	 - Install a tool pack into the project
* [spwn tool ls](./spwn_tool_ls.md)	 - List installed tool packs
* [spwn tool show](./spwn_tool_show.md)	 - Inspect a tool pack
* [spwn tool uninstall](./spwn_tool_uninstall.md)	 - Uninstall a tool pack from the project

