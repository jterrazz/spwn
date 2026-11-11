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
| plan, apply, drift | The machine read against it, carried out, and its ledger      |
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

A same-name file of a stronger layer replaces a weaker one whole, an item as a Machinefile: there is no merge between layers. Only a `.yaml` file of a source's `catalog/` and `machinefiles/` is read: a README, a lockfile or any other entry beside them is skipped whatever its type (a link, a pipe), and a `.yml` file is refused, since its owner meant it to be read. Of the entries read, only regular files are, and anything else is refused with its path: a pipe, a device, a link to a folder, and every link of the shared source, whose content is the remote's. In your own folders a link to a regular file is followed, so your Machinefiles and items may live in a dotfiles repository. The `overlay` layer is the overlay above, and `this machine` is a system value a Machinefile does not say.

The shared source is pinned by a whole commit (40 characters), never a branch or a tag: a clone already at it is read with nothing fetched, one at another commit is fetched from its `url` and checked out, and a source that cannot be read stops the command with its name and its commit. A `script` method there runs on the machine, which is why the pin is the owner's to move.

Reading a source runs no code of it. Its `name` is a Machinefile's kind of name (lower-case words joined by dashes), so its clone is one folder of the cache and never a path or a flag; its `url` is passed after `--`; git runs with no hook (`core.hooksPath=/dev/null`), no submodule, and no transport but https and ssh (`protocol.allow=never`): a `url` that is not `https://`, `ssh://` or `user@domain:path` (an `ext::` command, a `file://` folder) is refused before git runs. A clone already there is trusted only untouched: one holding a change, or a file git does not track or ignores, is refused, and station clones it again once it is deleted.

The built-in Machinefiles are workspaces to extend: `station-linux` (Ubuntu 24.04 arm64, `nat`, mise, git where it has a method, Oh My Zsh, the git shortcuts and `hushlogin`) and `station-macos` (macOS arm64, bridged, Homebrew, git, Oh My Zsh, the same settings).

## The resolution

A Machinefile resolves into the steps of a plan, in this order:

1. **The files.** Its `extends`, in order and depth first, then itself, each file followed by its overlay; a file is read once however many extend it, and a cycle of extends is refused, printed around its loop (`a extends b extends a`).
2. **The merge.** A later file's value replaces an earlier one's: a key of `machine` or `vm` by key, an app or a setting whole by name, `prefer`, `accounts` and `ports` whole. A Machinefile that itself says no `machine` is a mixin, refused if applied; a `computer` with `vm`, itself or from a file it extends, is refused naming that file.
3. **The method of each item.** Its methods whose selector matches the system ([The selector](catalog.md#the-selector)); a version range is met by the system's version, never by an unknown one. Among them, prefer's keys first in its order, then the managers this system ranks by their rank (`priority`), then the other managers, then a `script`; a tie keeps the file's order. An app's `manager` keeps only that key's methods.
4. **The errors.** No method for the system stops the plan (`ghostty has no method for ubuntu 24.04`) unless the item is `optional`, which skips it and says so: an entry that asks a `manager` the item has no method by there is skipped naming the methods it has, so a mistaken pin is never a silent skip. A version the chosen method cannot pin is refused, naming the method of the item that can; an item the catalog does not hold, listed or baked into the image (`vm.image.bake`), or a setting listed as an app, is refused. A refused entry is named at its file and line (`~/.config/station/machinefiles/studio-linux.yaml:6`).
5. **What an item brings.** Every item it leans on: its `requires`, its method's provider (`provided-by: herdr-app` brings `herdr-app`), and the manager its method installs by (`brew` brings `homebrew`), each resolved like a listed one; a setting's `requires-one-of` is met by any the plan holds, else by its first. `apps install` brings the same links of an app, by the method it installs it with here, its manager only while that manager is missing, and the manager is the one the catalog names (a missing `npm` brings `node`, never `homebrew` itself).
6. **The order.** What an item leans on before it (what it requires, its provider, its manager), `after` between two items the plan holds; among the steps ready, the first in the plan's order runs next. A cycle of requires and after is refused, printed around its loop (`left needs right needs left`).

The resolution is pure: it reads files and the catalog, never the machine. It installs nothing either: `apps`, `settings`, `accounts` and `ports` do, called by the plan.

## explain

`station explain [<key>] [--machinefile <name>]` resolves the Machinefile this machine carries (the one named, else the one it is bound to, else the one named after it) and prints each value, its layer, and the Machinefile or item that says it: `apps.jq` (listed, by which file), `apps.jq.method` (the key chosen, by the item's own layer), `apps.jq.because`. A key explains itself and every key under it (`vm`, `apps.jq`); a key the resolution holds no value of is refused. Every refusal of the resolution above is its own code ([Commands](commands.md)). With `--json` it prints `station.machinefile.explain/v1`: `machinefile`, the `files` read, and every `value` with its `key`, `value`, `layer`, `from` and `file`.

## plan, apply, drift

A machine carries one Machinefile: `station init --machinefile <name|path>` or `station apply --machinefile <name|path>` binds it, in `preferences.json`'s `machinefile` group ([Preferences](preferences.md)), and a bare `plan`, `apply`, `drift` or `explain` reads it. A Machinefile named by its file is copied into `~/.config/station/machinefiles/<name>.yaml`, under the name it says, and the machine is bound to that name, never to the path: every later read finds it in the user layer, wherever the file came from ([ADR-034](../decisions/034-apply-carries-a-machinefile-the-ledger-bounds-removal.md)). The copy never replaces your own: the same words copy nothing, and other words under a name the folder already holds are refused before any effect (`edit that file, or remove it to copy <path> instead`). Until it is copied, the file is shown where you named it.

- **plan** reads each step through its family (an app through apps' inventory, a setting through settings' judge), then says what it asks of you, once per need (`asks your password once`, only where this machine's sudo asks one: read once, never prompting, and none in a VM station made), what it installs, sets and adopts, what it leaves as it is, what is extra, and what has to happen once applied, each once. An app present by another way than its method is adopted, never installed again; a setting changed by hand since station set it is left as it is (`station settings diff` shows how); an app that no longer answers (`git --version` failing once a macOS update took its developer kit) is installed again; a step of the `image` or `boot` phase is left to it.
- **apply** prints the plan, then carries it out one step at a time, through `apps install` and `settings apply`: a step whose requirement did not install is not run, naming it, every other one goes on; an interruption says `station apply resumes`, which the next `station apply` does, from the plan read again, an item that finished just before it being kept as its install answered; one whose install handed nothing back is in the ledger when a fresh read of the apps gives its row within 15 seconds of the interrupt (unrecorded otherwise, so never removed); off a terminal, a plan that asks you anything is refused before any step. A `vault:` value is read from the vault only to hand its setting the secret, and is shown as the reference everywhere.
- **drift** is the plan read as conditions: `present`, `absent`, `outdated`, `changed`, `not managed`, `extra`, `unknown`. It never acts.

What `apply` installed is in the ledger, `~/.local/state/station/ledger.json`: each app, its method, the Machinefile that listed it and when. Two applies at once take its lock in turn (`ledger.json.lock`), and each change is written whole, so neither loses the other's record. An app the ledger holds for this Machinefile that it lists no longer is extra (one installed for another Machinefile is not, nor an `optional` one it still lists that this system skips); what was there before station, adopted, or installed by `station apps install`, is never in the ledger, so never removed. Settings are not in the ledger; `station settings off` turns one off.

`apply --remove` asks on a terminal and needs `--yes` off one; off a terminal, an extra whose uninstall asks your password is refused before any step, as a plan's need is. It then takes the extras back, what another one leans on (what it requires, its provider, its manager) after it, and each as `plan` says:

| An extra that reads                                                                                                                                | `plan` says                                                                   | `apply --remove`                              |
| -------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- | --------------------------------------------- |
| there by the method that installed it, whoever put it back                                                                                         | `listed no longer`                                                            | uninstalls it by that method, then forgets it |
| not there                                                                                                                                          | `already gone`                                                                | forgets it                                    |
| there by another method since                                                                                                                      | `present another way since station installed it: kept, and yours`             | keeps it, yours now, and forgets it           |
| there or not, its manager missing here (it cannot be read by its method)                                                                           | `its manager <manager> is missing: kept`                                      | keeps it, and its record for the next run     |
| there, but station cannot uninstall it (a script's), asking nothing                                                                                | `station cannot uninstall it: kept, and yours: remove it by hand if you want` | keeps it, yours now, and forgets it           |
| there by its method, a manager holding packages of yours (what no app of the catalog installs, `npm i -g typescript`), or whose list does not read | `holds <n> packages of yours: kept` (`what it holds does not read: kept`)     | keeps it, and its record for the next run     |
| failing, or not readable now                                                                                                                       | `reads <condition>: kept`                                                     | keeps it, and its record for the next run     |
| needed by another app there (it requires it, provides it, or manages it), one not taken back with it                                               | `required by <app>: kept`                                                     | keeps it, and its record for the next run     |

`plan` and the run say these in the same words. A plan whose only extras `--remove` would let go of (already gone, held another way since, or one station cannot uninstall) is not calm: it ends on `station apply --remove  lets go of what is yours now`. A manager's packages are what each method it manages lists (`npm ls -g`, `uv tool list`, `bun pm ls -g`, `mise ls --global`) that no app of the catalog placed here installs by that method, npm and corepack being node's own, read from your home with mise pinned to its global file, so the folder you run station from never picks which node answers; once you remove the last of them, the next `apply --remove` takes it back. What an app there leans on is judged by its method as `station apps` shows it (the one chosen here, even for an app present another way); an app `station apps` has no row of (one not for this machine) holds nothing, and a setting's `requires` holds no app. An extra station cannot uninstall is never counted as taken back, so what it needs is `required by` it in the plan already, and when another app there needs it, it is `required by <app>: kept` like any other, with its record; an extra whose uninstall fails at the run is still there too, so each extra it leans on that comes after it is `required by <app>: kept, judged again next run`, with its record.

## The hard cases

| Case                                    | Where                                                                                             |
| --------------------------------------- | ------------------------------------------------------------------------------------------------- |
| 1. Per-system upgrade                   | the item's method (`upgrade`), never a Machinefile                                                |
| 2. A package renamed per distro         | the item's method (`provides`, `apt.packages`)                                                    |
| 3. Versions and arch                    | `machine.system`, matched by the selector and its version range                                   |
| 4. No method for the system             | a plan error, unless `optional: true`                                                             |
| 5. Several managers                     | the system's priority, `prefer`, an app's `manager`                                               |
| 6. A version a manager cannot pin       | `version`, refused with the method that can                                                       |
| 10. A dependency cycle                  | refused, printed around its loop                                                                  |
| 12. The format changes between stations | `station.machinefile/v1`, `update station` for a newer one                                        |
| 13. Fixing an item locally              | `~/.config/station/catalog/`, the user layer; `explain` says which applies                        |
| 5. Already present by another way       | adopted by the plan, never installed again                                                        |
| 7. Installed outside station            | `not managed` in drift; `apply --remove` never removes it (the ledger)                            |
| 8. Interactive installs                 | `needs`, said at the head of the plan; refused off a terminal                                     |
| 9. A restart needed                     | `after-apply`, said once at the end of the plan and of the run                                    |
| 11. Interruption, partial failure       | one step at a time; what waits on a failure is not run, the rest goes on; `station apply resumes` |
| 14. Secrets                             | a `vault:` reference, read only to hand the setting its value                                     |
| 15. A dotfile edited by hand            | `changed` in drift, left as it is by apply                                                        |
| 16. A macOS update breaks an app        | `changed` in drift, installed again by apply                                                      |
| 17, 18, 19                              | the lockfile and upgrade                                                                          |

## What is read today

The format is whole now, so a Machinefile a newer station writes in v1 decodes in this one; some keys are declared for the lots that read them and held, unread, until then:

| Key                                                | Read today                                                                                                                                                                                                                                                                                                                                                               | Read by                                      |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------- |
| `machine`, `extends`, `prefer`, `apps`, `settings` | by the resolution, `explain`, `plan`, `apply` and `drift`                                                                                                                                                                                                                                                                                                                |                                              |
| an app's `options`, a setting's values, `vault:`   | checked against the item's options, shown by `explain`; a setting's values handed to it by `apply`                                                                                                                                                                                                                                                                       | an app's options: the lots that read them    |
| `vm`                                               | refused on a computer, its `image.bake` checked against the catalog, shown by `explain`                                                                                                                                                                                                                                                                                  | `machines create --from`, `images build`     |
| `accounts`                                         | shown by `explain`                                                                                                                                                                                                                                                                                                                                                       | a VM's first start, the accounts a Mac keeps |
| `ports`                                            | checked (a public port one by one, three at most), shown                                                                                                                                                                                                                                                                                                                 | the ports' rights and audiences              |
| `description`                                      | decoded                                                                                                                                                                                                                                                                                                                                                                  | the menu bar                                 |
| the overlay                                        | merged by the resolution                                                                                                                                                                                                                                                                                                                                                 | the menu bar, which writes it                |
| the user catalog                                   | by every command: an item of `~/.config/station/catalog/` replaces the built-in one, and the folders its methods land commands in (`path`) are searched; a refused file stops every command that reads the catalog at its line, while `--version`, help and the engine (which the menu bar must keep reaching) run on the built-in items and print the refusal on stderr |                                              |
| the shared source                                  | by `explain`, `plan`, `apply` and `drift`: every other command reads the user and built-in catalog                                                                                                                                                                                                                                                                       |                                              |
