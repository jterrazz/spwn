---
title: "spwn plugin"
slug: "spwn-plugin"
---

## spwn plugin

Manage runtime-targeted plugin packs (e.g. @spwn/mempalace)

### Synopsis

Plugin packs are tool packs that target specific runtimes and inject
configuration into the runtime at spawn time (e.g. MCP servers into
Claude Code's settings.json).

Attach one to an agent with:
  spwn agent add <agent> --plugin <pack>

Plugins coexist with --tool in the agent manifest. Both lists resolve
through the same builder registry, so plugins see the full tool
dependency graph and vice-versa.

### Options

```
  -h, --help   help for plugin
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn plugin install](./spwn_plugin_install.md)	 - Attach a plugin to every agent in the project
* [spwn plugin ls](./spwn_plugin_ls.md)	 - List installed plugin packs
* [spwn plugin show](./spwn_plugin_show.md)	 - Inspect a plugin pack
* [spwn plugin uninstall](./spwn_plugin_uninstall.md)	 - Detach a plugin from every agent in the project

