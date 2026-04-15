---
title: "spwn plugin"
slug: "spwn-plugin"
---

## spwn plugin

Manage plugins (e.g. @spwn/unix, @spwn/mempalace)

### Synopsis

Plugins are the unified building blocks that agents plug into their worlds.
One schema covers what used to be split between tools, runtime-config providers, and skills.

Install a catalog plugin into the project's agents + lockfile with:
  spwn plugin install @spwn/python

Remove it with:
  spwn plugin uninstall @spwn/python

List what's installed with:
  spwn plugin ls

Local plugins authored under spwn/plugins/<name>/ are referenced by
bare name in agent.yaml and do NOT go through the install verb — they
are authored in place.

### Options

```
  -h, --help   help for plugin
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn plugin install](./spwn_plugin_install.md)	 - Install a plugin into the project
* [spwn plugin ls](./spwn_plugin_ls.md)	 - List installed plugins
* [spwn plugin show](./spwn_plugin_show.md)	 - Inspect a plugin
* [spwn plugin uninstall](./spwn_plugin_uninstall.md)	 - Uninstall a plugin from the project

