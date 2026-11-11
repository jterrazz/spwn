# Machinefiles

A Machinefile is one YAML file in the schema `station.machinefile/v1` that says what a machine carries: the catalog defines each item, a Machinefile chooses them ([ADR-033](../decisions/033-a-machine-is-declared-by-its-machinefile.md)). This page is the format, every key of it, where the files are read from, how they resolve against the catalog, and what this station reads of them today; an item's own format is [The catalog's item format](catalog.md).

| Section            | Answers                                                       |
| ------------------ | ------------------------------------------------------------- |
| The file           | Its name, how strictly it is read, its schema                 |
| The keys           | Every key, in reading order                                   |
| The overlay        | `<name>.local.yaml`, the one file the menu bar will write     |
| Sources and layers | Where files are read from, and which one wins                 |
| The resolution     | How a Machinefile becomes the steps of a plan, and its errors |
| explain            | Each value and the layer it comes from                        |
| The hard cases     | Where the format absorbs each of them                         |
| What is read today | The keys declared for the lots that read them                 |

## The file

A Machinefile is `machinefiles/<name>.yaml`, its `name` its file's: lower-case words joined by dashes, `<whose>-<what>` by convention (`jterrazz-macbook-pro`, `studio-linux`, `jterrazz-base`). A file that says no `machine:` is a mixin: it is only ever extended, never applied.

It is read as strictly as an item:

- the schema first: `machinefile: station.machinefile/v2` is refused with `update station` before anything else is judged;
- one YAML 1.2 document, every key known, at every depth (`line 5: field tint not found`);
- no anchor, alias or merge key, no plain scalar YAML 1.1 reads otherwise (`yes`, `on`, `0755`);
- every enumerated value one of its list, every check of the file passed (a name, a size, a term, a port range).

What needs the catalog or the files a Machinefile extends (an item, an option, a method for the system, `vm` on a computer) is judged by the resolution, never by the file alone.

The JSON Schema an editor checks a file against, [`schemas/station.machinefile.v1.json`](../../schemas/station.machinefile.v1.json), is projected from the decoder's Go types by `make generate`, and a test fails when it drifts. It covers a Machinefile and its overlay alike; station alone checks which keys each may carry.

## The keys

```yaml
machinefile: station.machinefile/v1
name: studio-linux
description: The owner's Linux workspace
machine: # WHAT it is
    kind: workspace # computer | workspace | server
    system: { os: ubuntu, version: '24.04', arch: arm64 }

vm: # HOW it runs (a VM's alone)
    lifetime: persistent # persistent | ephemeral
    resources: { cpus: 8, memory: 32G, disk: 80G }
    network: nat # nat | bridged
    image: { from: 'ghcr.io/cirruslabs/ubuntu:24.04', bake: [node, bun] }

extends: [station-linux, jterrazz-base] # WHAT it inherits
prefer: [mise] # method keys tried first (rare)

apps: # WHAT it carries
    - docker
    - { name: node, version: '24' }
    - { name: ghostty, optional: true }
    - { name: jq, manager: mise }
settings:
    git-identity: { from: computer }
    autologin: { password: 'vault:agent-password' }
    hushlogin: {}

accounts: # WHAT accounts it is shared, and for how long
    share: [{ account: claude, for: 7d }]
    keep: [github]
ports: # WHAT it may open, and to whom
    open: 3000-9999
    to: private # local | private | public
    except: [22]
```

| Key              | Value                                                                                                                                                                                                                                                  |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `machinefile`    | the schema, `station.machinefile/v1`                                                                                                                                                                                                                   |
| `name`           | its file's name                                                                                                                                                                                                                                        |
| `description`    | one line, the owner's words                                                                                                                                                                                                                            |
| `machine.kind`   | `computer` (a machine you own: it runs the engine and hosts VMs), `workspace` (a VM station sets up), `server` (a VM that runs the owner's own tooling, kept alive by station); required in `machine`                                                  |
| `machine.system` | `os` (`macos`, `ubuntu`, `debian`), `version` (`24.04`, `26`: of the os it names), `arch` (`arm64`, `amd64`); what it does not say is this machine's                                                                                                   |
| `vm.lifetime`    | `persistent` (kept until deleted) or `ephemeral` (deleted once powered off from inside)                                                                                                                                                                |
| `vm.resources`   | `cpus` (a count), `memory` and `disk` (a size: `512M`, `32G`, `1T`)                                                                                                                                                                                    |
| `vm.network`     | `nat` (behind its computer, no address on the LAN) or `bridged` (its own LAN address)                                                                                                                                                                  |
| `vm.image`       | `from`, the image its disk is made from, and `bake`, the items baked into it                                                                                                                                                                           |
| `extends`        | the Machinefiles it inherits, in order                                                                                                                                                                                                                 |
| `prefer`         | method keys (`brew`, `mise`, `apt`, `github-release`...) tried first, in this order, before the system's priority                                                                                                                                      |
| `apps`           | a list: an item's name, or `{ name, version, optional, manager, options }`: the version asked (its method must pin it), `optional` (no method for the system skips it), the method key it is installed by instead of the priority's, its option values |
| `settings`       | a mapping by setting: its option values by option (`{}` for none), `from: computer` (its values read on the computer a VM runs on), `optional`                                                                                                         |
| `accounts.share` | each account shared with a VM, `{ account, for }`, `for` a term (`7d`, `12h`, `30m`, `2w`)                                                                                                                                                             |
| `accounts.keep`  | the accounts a computer keeps signed in                                                                                                                                                                                                                |
| `ports.open`     | the ports it may open: one port, a range (`3000-9999`), or a list of them                                                                                                                                                                              |
| `ports.to`       | who they are shared with: `local` (its computer alone), `private` (your devices on your private network), `public` (the internet): a public port is named one by one, never in a range, three at most                                                  |
| `ports.except`   | ports never shared                                                                                                                                                                                                                                     |
| `overrides`      | an overlay's alone: the Machinefile it changes                                                                                                                                                                                                         |

A secret is never written in a Machinefile: a secret option's value is a reference, `vault:<name>`, which plans and logs show and never resolve. An option value that is none of the item's options, or does not fit its type, is refused.

## The overlay

`~/.config/station/machinefiles/<name>.local.yaml` changes the Machinefile `<name>` without touching it: it names it in `overrides`, and carries `apps`, `settings`, `accounts` and `ports` alone. It is the owner's only (an overlay in another source is refused), and it is the one file the menu bar will write, so a file written by hand is never rewritten.

```yaml
machinefile: station.machinefile/v1
overrides: jterrazz-macbook-pro
apps: { add: [lazygit], remove: [jq] }
settings: { set: { macos-dock: { size: 36 } }, remove: [dock-spacer] }
```

`apps` is what it adds (`add`, the entries of a list) and the names it removes (`remove`); `settings` is what it sets (`set`, replacing a same-name setting whole) and removes. `accounts` and `ports` replace their Machinefile's.

## Sources and layers

Files are read from three sources, strongest first:

| Layer      | Source                                                                                                                                                                                 |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `user`     | `~/.config/station/catalog/<category>/<name>.yaml` and `~/.config/station/machinefiles/<name>.yaml`, the owner's files                                                                 |
| `shared`   | a git repository named in `preferences.json`'s `machinefile` group ([Preferences](preferences.md)), cloned into `~/.cache/station/sources/<name>` and checked out at its pinned commit |
| `built-in` | station's own, embedded in the binary: the catalog's items, and the Machinefiles `station-linux` and `station-macos`                                                                   |

A same-name file of a stronger layer replaces a weaker one whole, an item as a Machinefile: there is no merge between layers. Only a `.yaml` file of a source's `catalog/` and `machinefiles/` is read: a README or a lockfile beside them is skipped, and a `.yml` file is refused, since its owner meant it to be read. Only regular files are read, and anything else is refused with its path: a pipe, a device, a link to a folder, and every link of the shared source, whose content is the remote's. In your own folders a link to a regular file is followed, so your Machinefiles and items may live in a dotfiles repository. The `overlay` layer is the overlay above, and `this machine` is a system value a Machinefile does not say.

The shared source is pinned by a whole commit (40 characters), never a branch or a tag: a clone already at it is read with nothing fetched, one at another commit is fetched from its `url` and checked out, and a source that cannot be read stops the command with its name and its commit. A `script` method there runs on the machine, which is why the pin is the owner's to move.

Reading a source runs no code of it. Its `name` is a Machinefile's kind of name (lower-case words joined by dashes), so its clone is one folder of the cache and never a path or a flag; its `url` is passed after `--`; git runs with no hook (`core.hooksPath=/dev/null`) and no submodule. A clone already there is trusted only untouched: one holding a change, or a file git does not track or ignores, is refused, and station clones it again once it is deleted.

The built-in Machinefiles are workspaces to extend: `station-linux` (Ubuntu 24.04 arm64, `nat`, mise, git where it has a method, Oh My Zsh, the git shortcuts and `hushlogin`) and `station-macos` (macOS arm64, bridged, Homebrew, git, Oh My Zsh, the same settings).

## The resolution

A Machinefile resolves into the steps of a plan, in this order:

1. **The files.** Its `extends`, in order and depth first, then itself, each file followed by its overlay; a file is read once however many extend it, and a cycle of extends is refused, printed around its loop (`a extends b extends a`).
2. **The merge.** A later file's value replaces an earlier one's: a key of `machine` or `vm` by key, an app or a setting whole by name, `prefer`, `accounts` and `ports` whole. A Machinefile that itself says no `machine` is a mixin, refused if applied; a `computer` with `vm`, itself or from a file it extends, is refused naming that file.
3. **The method of each item.** Its methods whose selector matches the system ([The selector](catalog.md#the-selector)); a version range is met by the system's version, never by an unknown one. Among them, prefer's keys first in its order, then the managers this system ranks by their rank (`priority`), then the other managers, then a `script`; a tie keeps the file's order. An app's `manager` keeps only that key's methods.
4. **The errors.** No method for the system stops the plan (`ghostty has no method for ubuntu 24.04`) unless the item is `optional`, which skips it and says so; a version the chosen method cannot pin is refused, naming the method of the item that can; an item the catalog does not hold, or a setting listed as an app, is refused naming the file.
5. **What an item brings.** Its `requires`, and the manager its method installs by (`brew` brings `homebrew`), each resolved like a listed one; a setting's `requires-one-of` is met by any the plan holds, else by its first.
6. **The order.** Requires first, a manager before what it installs, `after` between two items the plan holds; among the steps ready, the first in the plan's order runs next. A cycle of requires and after is refused, printed around its loop (`left needs right needs left`).

The resolution is pure: it reads files and the catalog, never the machine. It installs nothing either: `apps`, `settings`, `accounts` and `ports` do, called by the plan.

## explain

`station explain [<key>] [--machinefile <name>]` resolves the Machinefile named after the machine (or the one named) and prints each value, its layer, and the Machinefile or item that says it: `apps.jq` (listed, by which file), `apps.jq.method` (the key chosen, by the item's own layer), `apps.jq.because`. A key explains itself and every key under it (`vm`, `apps.jq`); a key the resolution holds no value of is refused. Every refusal of the resolution above is its own code ([Commands](commands.md)). With `--json` it prints `station.machinefile.explain/v1`: `machinefile`, the `files` read, and every `value` with its `key`, `value`, `layer`, `from` and `file`.

## The hard cases

| Case                                    | Where                                                                      |
| --------------------------------------- | -------------------------------------------------------------------------- |
| 1. Per-system upgrade                   | the item's method (`upgrade`), never a Machinefile                         |
| 2. A package renamed per distro         | the item's method (`provides`, `apt.packages`)                             |
| 3. Versions and arch                    | `machine.system`, matched by the selector and its version range            |
| 4. No method for the system             | a plan error, unless `optional: true`                                      |
| 5. Several managers                     | the system's priority, `prefer`, an app's `manager`                        |
| 6. A version a manager cannot pin       | `version`, refused with the method that can                                |
| 10. A dependency cycle                  | refused, printed around its loop                                           |
| 12. The format changes between stations | `station.machinefile/v1`, `update station` for a newer one                 |
| 13. Fixing an item locally              | `~/.config/station/catalog/`, the user layer; `explain` says which applies |
| 14. Secrets                             | `vault:<name>`, references only                                            |
| 7, 8, 9, 11, 15, 16, 17, 18, 19         | the plan, apply, drift, the lockfile and upgrade, which read these keys    |

## What is read today

The format is whole now, so a Machinefile a newer station writes in v1 decodes in this one; some keys are declared for the lots that read them and held, unread, until then:

| Key                                                | Read today                                                                                                                                                                                                                                                                                                                                                               | Read by                                      |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------- |
| `machine`, `extends`, `prefer`, `apps`, `settings` | by the resolution and `explain`                                                                                                                                                                                                                                                                                                                                          | the plan and apply                           |
| an app's `options`, a setting's values, `vault:`   | checked against the item's options, shown by `explain`                                                                                                                                                                                                                                                                                                                   | apply                                        |
| `vm`                                               | refused on a computer, shown by `explain`                                                                                                                                                                                                                                                                                                                                | `machines create --from`, `images build`     |
| `accounts`                                         | shown by `explain`                                                                                                                                                                                                                                                                                                                                                       | a VM's first start, the accounts a Mac keeps |
| `ports`                                            | checked (a public port one by one, three at most), shown                                                                                                                                                                                                                                                                                                                 | the ports' rights and audiences              |
| `description`                                      | decoded                                                                                                                                                                                                                                                                                                                                                                  | the menu bar                                 |
| the overlay                                        | merged by the resolution                                                                                                                                                                                                                                                                                                                                                 | the menu bar, which writes it                |
| the user catalog                                   | by every command: an item of `~/.config/station/catalog/` replaces the built-in one, and the folders its methods land commands in (`path`) are searched; a refused file stops every command that reads the catalog at its line, while `--version`, help and the engine (which the menu bar must keep reaching) run on the built-in items and print the refusal on stderr |                                              |
| the shared source                                  | by `explain` alone: every other command reads the user and built-in catalog                                                                                                                                                                                                                                                                                              | the plan and apply                           |
