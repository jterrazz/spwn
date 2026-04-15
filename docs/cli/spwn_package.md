---
title: "spwn package"
slug: "spwn-package"
---

## spwn package

Manage reusable packages (e.g. @spwn/unix, @spwn/mempalace)

### Synopsis

Packages are the unified building blocks that agents plug into their worlds:
tools, plugins, and skills all share one schema.

Install a catalog package into the project's agents + lockfile with:
  spwn package install @spwn/python

Remove it with:
  spwn package uninstall @spwn/python

List what's installed with:
  spwn package ls

Local packages authored under spwn/packages/<name>/ are referenced by
bare name in agent.yaml and do NOT go through the install verb — they
are authored in place.

### Options

```
  -h, --help   help for package
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn package install](./spwn_package_install.md)	 - Install a package into the project
* [spwn package ls](./spwn_package_ls.md)	 - List installed packages
* [spwn package show](./spwn_package_show.md)	 - Inspect a package
* [spwn package uninstall](./spwn_package_uninstall.md)	 - Uninstall a package from the project

