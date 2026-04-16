---
title: "spwn pack"
slug: "spwn-pack"
---

## spwn pack

Manage packs (e.g. @spwn/unix, @spwn/mempalace)

### Synopsis

Plugins are the unified building blocks that agents plug into their worlds.
One schema covers what used to be split between tools, runtime-config providers, and skills.

Install a catalog pack into the project's agents + lockfile with:
  spwn pack install @spwn/python

Remove it with:
  spwn pack uninstall @spwn/python

List what's installed with:
  spwn pack ls

Local plugins authored under spwn/packs/<name>/ are referenced by
bare name in agent.yaml and do NOT go through the install verb — they
are authored in place.

### Options

```
  -h, --help   help for pack
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn pack install](./spwn_pack_install.md)	 - Install a pack into the project
* [spwn pack ls](./spwn_pack_ls.md)	 - List installed packs
* [spwn pack show](./spwn_pack_show.md)	 - Inspect a pack
* [spwn pack uninstall](./spwn_pack_uninstall.md)	 - Uninstall a pack from the project

