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
| Phases             | What an image bakes, a first start, and a bare apply          |
| The lock           | What a machine carries of it, and what `apply` installs again |
| upgrade            | Every app it carries updated, the managers first, every night |
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
    - { name: jq, manager: apt }
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

| Key              | Value                                                                                                                                                                                                                                                                                 |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `machinefile`    | the schema, `station.machinefile/v1`                                                                                                                                                                                                                                                  |
| `name`           | its file's name                                                                                                                                                                                                                                                                       |
| `description`    | one line, the owner's words                                                                                                                                                                                                                                                           |
| `machine.kind`   | `computer` (a machine you own: it runs the engine and hosts VMs), `workspace` (a VM station sets up), `server` (a VM that runs the owner's own tooling, kept alive by station); required in `machine`                                                                                 |
| `machine.system` | `os` (`macos`, `ubuntu`, `debian`), `version` (`24.04`, `26`: of the os it names), `arch` (`arm64`, `amd64`); what it does not say is this machine's                                                                                                                                  |
| `vm.lifetime`    | `persistent` (kept until deleted) or `ephemeral` (deleted once powered off from inside)                                                                                                                                                                                               |
| `vm.resources`   | `cpus` (a count), `memory` and `disk` (a size: `512M`, `32G`, `1T`)                                                                                                                                                                                                                   |
| `vm.network`     | `nat` (behind its computer, no address on the LAN) or `bridged` (its own LAN address)                                                                                                                                                                                                 |
| `vm.image`       | `from`, the image its disk is made from, and `bake`, the items baked into it: the image phase installs them and what they require ([Phases](#phases))                                                                                                                                 |
| `extends`        | the Machinefiles it inherits, in order                                                                                                                                                                                                                                                |
| `prefer`         | method keys (`brew`, `mise`, `apt`, `github-release`...) tried first, in this order, before the system's priority                                                                                                                                                                     |
| `apps`           | a list: an item's name, or `{ name, version, optional, manager, options }`: how far it moves ([Versions](#versions); its method must hold that range), `optional` (no method for the system skips it), the method key it is installed by instead of the priority's, its option values |
| `settings`       | a mapping by setting: its option values by option (`{}` for none), `from: computer` (its values read on the computer a VM runs on), `optional`                                                                                                                                        |
| `accounts.share` | each account shared with a VM, `{ account, for }`, `for` a term (`7d`, `12h`, `30m`, `2w`)                                                                                                                                                                                            |
| `accounts.keep`  | the accounts a computer keeps signed in                                                                                                                                                                                                                                               |
| `ports.open`     | the ports it may open: one port, a range (`3000-9999`), or a list of them                                                                                                                                                                                                             |
| `ports.to`       | who they are shared with: `local` (its computer alone), `private` (your devices on your private network), `public` (the internet): a public port is named one by one, never in a range, three at most                                                                                 |
| `ports.except`   | ports never shared                                                                                                                                                                                                                                                                    |
| `overrides`      | an overlay's alone: the Machinefile it changes                                                                                                                                                                                                                                        |

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

| Layer      | Source                                                                                                                                                                  |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `user`     | `~/.config/station/catalog/<category>/<name>.yaml` and `~/.config/station/machinefiles/<name>.yaml`, the owner's files                                                  |
| `shared`   | a git repository named in `preferences.json`'s `machinefile` group ([Preferences](preferences.md)), cloned into `~/.cache/station/sources/<name>` at its default branch |
| `built-in` | station's own, embedded in the binary: the catalog's items, and the Machinefiles `station-linux` and `station-macos`                                                    |

A same-name file of a stronger layer replaces a weaker one whole, an item as a Machinefile: there is no merge between layers. Only a `.yaml` file of a source's `catalog/` and `machinefiles/` is read: a README, a lockfile or any other entry beside them is skipped whatever its type (a link, a pipe), and a `.yml` file is refused, since its owner meant it to be read. Of the entries read, only regular files are, and anything else is refused with its path: a pipe, a device, a link to a folder, and every link of the shared source, whose content is the remote's. In your own folders a link to a regular file is followed, so your Machinefiles and items may live in a dotfiles repository. The `overlay` layer is the overlay above, and `this machine` is a system value a Machinefile does not say.

The shared source follows its default branch, and you never name a commit, a branch or a tag: it is cloned at that branch the first time a command reads it, and read where its clone stands after that, with nothing fetched. `station upgrade`, and the engine's night that runs it, fetches the branch's newest commit from its `url` and checks it out once the upgrade is judged where the clone stood and before anything is installed, so a `script` method there that changed runs at the next upgrade. It moves forward only: a newest commit that does not descend from the clone's (a force-push, a rewritten history, another default branch) is refused, the clone stays where it stood, and the upgrade goes on from it, saying why (`its branch is at eeeeeee, which does not follow 1a2b3c4 … delete ~/.cache/station/sources/studio to take the branch as it is now`); deleting the clone is the one gesture that takes such a branch. A newest commit that does not resolve, or a fetch that fails (offline), leaves the clone the same way. A source that cannot be read stops the command with its name and the cause.

Reading a source runs no code of it. Its `name` is a Machinefile's kind of name (lower-case words joined by dashes), so its clone is one folder of the cache and never a path or a flag; its `url` is passed after `--`; git runs with no hook (`core.hooksPath=/dev/null`), no submodule, and no transport but https and ssh (`protocol.allow=never`): a `url` that is not `https://`, `ssh://` or `user@domain:path` (an `ext::` command, a `file://` folder) is refused before git runs. A clone already there is trusted only untouched: one holding a change, or a file git does not track or ignores, is refused, and station clones it again once it is deleted.

The built-in Machinefiles are workspaces to extend: `station-linux` (Ubuntu 24.04 arm64, `nat`, mise, git where it has a method, Oh My Zsh, the git shortcuts and `hushlogin`) and `station-macos` (macOS arm64, bridged, Homebrew, git, Oh My Zsh, the same settings).

## The resolution

A Machinefile resolves into the steps of a plan, in this order:

1. **The files.** Its `extends`, in order and depth first, then itself, each file followed by its overlay; a file is read once however many extend it, and a cycle of extends is refused, printed around its loop (`a extends b extends a`).
2. **The merge.** A later file's value replaces an earlier one's: a key of `machine` or `vm` by key, an app or a setting whole by name, `prefer`, `accounts` and `ports` whole. A Machinefile that itself says no `machine` is a mixin, refused if applied; a `computer` with `vm`, itself or from a file it extends, is refused naming that file.
3. **The method of each item.** Its methods whose selector matches the system ([The selector](catalog.md#the-selector)); a version range is met by the system's version, never by an unknown one. Among them, prefer's keys first in its order, then the managers this system ranks by their rank (`priority`), then the other managers, then a `script`; a tie keeps the file's order; `apps install` chooses an app's method on each kind of machine by the same order. An app's `manager` keeps only that key's methods. A step names its method as station runs and records it, so a `script` that runs a line is `shell`, as `station apps` and the ledger say it.
4. **The errors.** No method for the system stops the plan (`ghostty has no method for ubuntu 24.04`) unless the item is `optional`, which skips it and says so: an entry that asks a `manager` the item has no method by there is skipped naming the methods it has, so a mistaken pin is never a silent skip. A range the chosen method cannot hold is refused plainly, naming what the method holds and the method of the item that holds it (`jq 1.7: brew holds only its newest, never 1.7`), or, with none, that dropping the version is the way out (`postgres 17: brew holds only its newest of 18 (postgresql@18), never 17`); an item the catalog does not hold, listed or baked into the image (`vm.image.bake`), or a setting listed as an app, is refused. A refused entry is named at its file and line (`~/.config/station/machinefiles/studio-linux.yaml:6`).
5. **What the lock holds.** An app the machine's lock holds is carried by the lock's method when it is one of the matching methods (the `manager` an entry asks included), at the lock's version where that method pins one and the range asked holds it (`22` holds `22.11.0`, never `220.1`); otherwise the order above chooses, and the next run writes the lock anew ([The lock](#the-lock)).
6. **What an item brings.** Every item it leans on: its `requires`, its method's provider (`provided-by: herdr-app` brings `herdr-app`), and the manager its method installs by (`brew` brings `homebrew`), each resolved like a listed one; a setting's `requires-one-of` is met by any the plan holds, else by its first. `apps install` brings the same links of an app, by the method it installs it with here, its manager only while that manager is missing, and the manager is the one the catalog names (a missing `npm` brings `node`, never `homebrew` itself).
7. **The order.** What an item leans on before it (what it requires, its provider, its manager), `after` between two items the plan holds; among the steps ready, the first in the plan's order runs next. A cycle of requires and after is refused, printed around its loop (`left needs right needs left`).

The resolution is pure: it reads files and the catalog, never the machine. It installs nothing either: `apps`, `settings`, `accounts` and `ports` do, called by the plan.

## explain

`station explain [<key>] [--machinefile <name>]` resolves the Machinefile this machine carries (the one named, else the one it is bound to, else the one named after it) and prints each value, its layer, and the Machinefile or item that says it: `apps.jq` (listed, by which file), `apps.jq.method` (the key chosen, by the item's own layer), `apps.jq.because`. A key explains itself and every key under it (`vm`, `apps.jq`); a key the resolution holds no value of is refused. Every refusal of the resolution above is its own code ([Commands](commands.md)). With `--json` it prints `station.machinefile.explain/v1`: `machinefile`, the `files` read, and every `value` with its `key`, `value`, `layer`, `from` and `file`.

## plan, apply, drift

A machine carries one Machinefile: `station init --machinefile <name|path>` or `station apply --machinefile <name|path>` binds it, in `preferences.json`'s `machinefile` group ([Preferences](preferences.md)), and a bare `plan`, `apply`, `drift` or `explain` reads it. A Machinefile named by its file is copied into `~/.config/station/machinefiles/<name>.yaml`, under the name it says, and the machine is bound to that name, never to the path: every later read finds it in the user layer, wherever the file came from ([ADR-034](../decisions/034-apply-carries-a-machinefile-the-ledger-bounds-removal.md)). The copy never replaces your own: the same words copy nothing, and other words under a name the folder already holds are refused before any effect (`edit that file, or remove it to copy <path> instead`). Until it is copied, the file is shown where you named it.

- **plan** reads each step through its family (an app through apps' inventory, a setting through settings' judge), then says what it asks of you, once per need (`asks your password once`, only where this machine's sudo asks one: read once, never prompting, and none in a VM station made), what it installs, sets and adopts, what it leaves as it is, what is extra, and what has to happen once applied, each once. An app present by another way than its method is adopted, never installed again: held by another of its methods (a `brew` copy where the step chose `mise`), the plan names the method that holds it, by which `apps update` and `apps remove` then reach it, and `station apps move <app> --to <manager>` switches it; a setting changed by hand since station set it is left as it is (`station settings diff` shows how); an app that no longer answers (`git --version` failing once a macOS update took its developer kit) is installed again; a step of the `image` or `boot` phase is left to it.
- **apply** prints the plan, then carries it out one step at a time, through `apps install` and `settings apply`, each app by the method and at the version its step shows, the method the ledger then records: a step whose requirement did not install is not run, naming it, every other one goes on; an interruption says `station apply resumes`, which the next `station apply` does, from the plan read again, an item that finished just before it being kept as its install answered; one whose install handed nothing back is in the ledger when a fresh read of the apps gives its row within 15 seconds of the interrupt (unrecorded otherwise, so never removed); off a terminal, a plan that asks you anything is refused before any step. A `vault:` value is read from the vault only to hand its setting the secret, and is shown as the reference everywhere.
- **drift** is the plan read as conditions: `present`, `absent`, `outdated`, `changed`, `not managed`, `extra`, `unknown`. It never acts.

What `apply` installed is in the ledger, `~/.local/state/station/ledger.json`: each app, its method, the Machinefile that listed it and when, and each block of `~/.zshrc` it wrote, by id, with the digest of its text ([Dotfiles](dotfiles.md#package-blocks)). Two applies at once take its lock in turn (`ledger.json.lock`), and each change is written whole, so neither loses the other's record. An app the ledger holds for this Machinefile that it lists no longer is extra (one installed for another Machinefile is not, nor an `optional` one it still lists that this system skips); what was there before station, adopted, or installed by `station apps install`, is never in the ledger, so never removed. Settings are not in the ledger; `station settings off` turns one off.

`apply --remove` asks on a terminal and needs `--yes` off one; off a terminal, an extra whose uninstall asks your password is refused before any step, as a plan's need is. It then takes the extras back, what another one leans on (what it requires, its provider, its manager) after it, and each as `plan` says:

| An extra that reads                                                                                                                                | `plan` says                                                                   | `apply --remove`                              |
| -------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- | --------------------------------------------- |
| there by the method that installed it, whoever put it back                                                                                         | `listed no longer`                                                            | uninstalls it by that method, then forgets it |
| not there                                                                                                                                          | `already gone`                                                                | forgets it                                    |
| there by another method since                                                                                                                      | `present another way since station installed it: kept, and yours`             | keeps it, yours now, and forgets it           |
| there or not, its manager missing here (it cannot be read by its method)                                                                           | `its manager <manager> is missing: kept`                                      | keeps it, and its record for the next run     |
| there, but station cannot uninstall it (a script's with no `undo`), asking nothing                                                                 | `station cannot uninstall it: kept, and yours: remove it by hand if you want` | keeps it, yours now, and forgets it           |
| there by its method, a manager holding packages of yours (what no app of the catalog installs, `npm i -g typescript`), or whose list does not read | `holds <n> packages of yours: kept` (`what it holds does not read: kept`)     | keeps it, and its record for the next run     |
| failing, or not readable now                                                                                                                       | `reads <condition>: kept`                                                     | keeps it, and its record for the next run     |
| needed by another app there (it requires it, provides it, or manages it), one not taken back with it                                               | `required by <app>: kept`                                                     | keeps it, and its record for the next run     |

`plan` and the run say these in the same words. A plan whose only extras `--remove` would let go of (already gone, held another way since, or one station cannot uninstall) is not calm: it ends on `station apply --remove  lets go of what is yours now`. A manager's packages are what each method it manages lists (`npm ls -g --prefix <folder>` for each node mise holds globally, `uv tool list`, `bun pm ls -g`, `mise ls --global`) that no app of the catalog placed here installs by that method, npm and corepack being node's own, read from your home with mise's shims first and mise pinned to its global file, so neither the folder you run station from nor the PATH your shell gave it picks which node answers; once you remove the last of them, the next `apply --remove` takes it back. What an app there leans on is judged by its method as `station apps` shows it (the one chosen here, even for an app present another way); an app `station apps` has no row of (one not for this machine) holds nothing, and a setting's `requires` holds no app. An extra station cannot uninstall is never counted as taken back, so what it needs is `required by` it in the plan already, and when another app there needs it, it is `required by <app>: kept` like any other, with its record; an extra whose uninstall fails at the run is still there too, so each extra it leans on that comes after it is `required by <app>: kept, judged again next run`, with its record.

An apt app station recorded adding none of the packages of (`~/.local/share/station/apt`, [Apps and skills](apps-and-skills.md)) is yours, whoever the ledger says installed it: the plan says `listed no longer`, and the run keeps it, `station cannot uninstall it: kept, and yours: remove it by hand if you want`, and forgets it. Only `station apps remove <app>`, your own word for it, takes its package.

## Phases

A Machinefile is carried out in three phases, each `plan --phase` and `apply --phase` read alone ([Commands](commands.md#station-itself)):

| Phase   | Carries                                                                                                                          | Run by                                      |
| ------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| `image` | each item `vm.image.bake` lists, and what a baked item requires (its `requires`, its provider, its manager), in the plan's order | `station images build`, inside its build VM |
| `boot`  | the items whose `phase` is `boot`                                                                                                | a VM's first start                          |
| `apply` | every other item, the baked ones included: on a VM made from the image they read present and are kept                            | a bare `station apply`, the default         |

A step of another phase than the one run is left as it is, and only `apply`'s phase has extras, so `apply --remove` with another phase is refused before any effect. A baked item the Machinefile does not list under `apps` is carried all the same, `baked` as why.

## The lock

What a machine carries of its Machinefile is its lock, `~/.config/station/machinefiles/<name>.lock` beside it, in the schema `station.lock/v1`: the Machinefile's name, the shared source's name and the commit it was read at, and each app with the method that holds it and the version there. No source reads it (only a `.yaml` file is read), it is read as strictly as a Machinefile, and one that does not decode is refused at its file: removing it is the way back, since the next `apply` writes it anew from what is there. It is station's file: written whole, its apps by name, so two locks differ only where something moved, and written as a move under a file lock (`.writing`, beside it): a run applies what it changed to the lock as it stands, so two runs at once (two applies, an apply and an upgrade) keep each other's lines.

```yaml
lock: station.lock/v1
machinefile: jterrazz-macbook-pro
source: { name: studio, commit: 1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d }
apps:
    - name: gh
      method: brew
      version: 2.98.0
```

`apply` and `upgrade` write it once their run ends, interrupted or not, and only when it moved. It then names, for every app the Machinefile lists that reads there by a method, that method and the version it reads, an adopted app by the method that holds it; an app the run did not bring there (it failed, was interrupted, was not run, or does not read) keeps what the lock held of it, so the next `apply` installs it as locked; a copy no method holds is yours and leaves the lock, as does an app the Machinefile lists no longer. Settings are not locked. An app the lock holds is installed by the lock's method, at its version where that method pins one (mise's runtimes, an apt package as `package=<version>`, a release as its tag; a brew formula takes its newest), and the plan says so (`○ python  mise  locked 3.12.7`), `explain apps.<app>` naming the lock as the layer of its method; where the method cannot pin (brew), the plan says only what the lock records (`○ ripgrep  brew  recorded 14.0.0`), and `explain` has no `locked` line, since the install takes the newest. A lock of another kind of machine, whose method this one has not, gives way to this machine's choice. The lock is what `apply` installs on every path: fresh, locked, upgraded, interrupted, adopted, and from another machine's lock ([ADR-041](../decisions/041-the-lock-is-what-apply-installs.md)).

A lock written from a shared source holds the commit each run read, recorded and never asked of you: no run is refused for it, and the upgrade that moved the source says it in its moves (`shared source  studio@fffffff, was studio@1a2b3c4`).

## Versions

An app's `version` says how far it moves, in npm's semver words; it is read once, and a word that is none of its forms is refused at its line (`version ">=22" is no version: latest, 24, 24.11, 24.11.3, ^24.2.0 or ~24.11.0`).

| `version`             | Holds                | Moves by itself            |
| --------------------- | -------------------- | -------------------------- |
| nothing, `latest`     | every version        | majors, minors and patches |
| `'24'`, `^24.2.0`     | `^24.0.0`, `^24.2.0` | minors and patches of 24   |
| `'24.11'`, `~24.11.0` | `~24.11.0`           | patches of 24.11           |
| `'24.11.3'`           | that version         | nothing                    |

A range holds released versions alone: a prerelease (`25.0.0-rc.1`) is never moved to and never said waiting. Three numbers read through a vendor and a build their manager writes around them: `'21.0.13'` holds java's `temurin-21.0.13+9.0.LTS`, and a newer one waits as `change '21.0.13' to '27.0.1'`.

A method holds what its install carries to its manager: mise any range (by its prefix: `node@24`; every version is `node@latest`, never the catalog's narrower `lts`, and a catalog version that names a vendor keeps it, `java@temurin` for every version and `java@temurin-21` for `'21'`), bun any range (`pkg@^24.2.0`), apt and a release one exact version (`package=<version>`, a tag), every other method (brew, a cask, npm and uv through brew's bundle, a script) only its newest. An app whose item says `updates: self` (Claude, Ghostty, a browser: a cask whose app runs its own updater) is checked present and nothing more: no range moves it, `upgrade` leaves it (`updates itself: station checks it is there`), and its lock line keeps the version it was installed at, so a new Mac starts from it. A range a method cannot hold is refused by the resolution, never faked; brew holds a major only by a versioned formula, and an item whose formula is one holds that major (`postgres` installs `postgresql@18`, so `version: '18'` is held and `'17'` refused).

## upgrade

`station upgrade` reads the plan where the shared source's clone stands and refuses, before any effect, what it cannot run here; it then moves the source forward to its default branch's newest commit and reads the plan there ([Sources and layers](#sources-and-layers)), going on from the clone as it stood, in a `Source` section, when the source does not move. A clone whose own commit no longer resolves has no plan to refuse there: the source moves first and its newest commit is the plan, refused off a terminal as any other (the clone then put back), and the upgrade fails only when that commit does not resolve either, or the source cannot move; then it moves every app the Machinefile lists that reads there by one of its methods as far as its range lets it, by that method, the managers first and then the rest, each in the plan's order, so a manager is fresh before what it installs is updated: homebrew's own index by its `upgrade:` line, `brew update`, since station runs every bundle with `HOMEBREW_NO_AUTO_UPDATE=1` and an install never refreshes the index behind your back. A method that holds any range installs the app again at it (`mise use --global node@22`), whose newest it then holds (`node  22.14.0  mise  within 22`; exact, `held at 22.11.0`); one exact version asked of a method that holds no range around it (apt, a release) is kept (`pinned 2.43.0 by jterrazz-macbook-pro: kept`); every other app is updated as its manager does. Where the method reads the newest version its manager offers (mise's `mise latest`), a newer one past the range waits for your gesture, in a `Waiting` section after the lock (`node  25.1.0 available: change '22' to '25' in ~/.config/station/machinefiles/jterrazz-macbook-pro.yaml`); a method with no such read leaves it unsaid. An app a copy no method holds is yours and left as it is, an app of another phase is its phase's, and an app not there is `apply`'s, except one the lock holds: a locked version that no longer installs (a yanked mise version, an archive that dropped a package) fails `apply` with the way out in its line (`locked at 22.11.0 in ~/.config/station/machinefiles/m.lock: station upgrade installs the newest the Machinefile allows, or edit that line`), and `upgrade` installs that app by the lock's method at the newest the Machinefile allows, and relocks it. It then writes the lock anew and prints what moved in it (`gh  2.98.0 by brew, was 2.97.0 by brew`, the shared source's commit). Off a terminal, an upgrade whose updates ask you anything is refused before any effect, as `apply` is; an interruption says `station upgrade resumes`, and an app whose update landed just before it is read again, as `apply` reads an install, so the lock names the version the machine holds. It does not update station itself: re-running the install does ([Release](../guides/release.md#how-a-machine-takes-it)).

### Every night

A Mac's engine runs the same upgrade every night, unattended, for the Machinefile the Mac is bound to (by `station init --machinefile` or `station apply --machinefile`): at the first 03:00 of its own clock past the last night that completed, so a Mac asleep or off at that hour upgrades when it is back, and a Machinefile bound in the day waits for the next 03:00. An app whose update asks you something (a password) is left for your own `station upgrade`; a night the engine's stop interrupts runs again at its next start. What the night did is station's memory, kept in the ledger (`~/.local/state/station/ledger.json`): the apps it moved, each version waiting past a range, the apps it left for your run, the apps whose update failed, why the shared source was not pulled, and why the night did not run to its end (its Machinefile does not resolve, another run held the machine). A night that did not run to its end is tried again an hour later. The apps line of the Overview and of the menu bar says it: `2 moved last night` the day after, and one Needs you line per thing the night left you: a version waiting (`node 25.1.0 available: change '22' to '25'`, the file to edit beside it), an app left (`node waits for your station upgrade: its update asks you`), an app failed, the source not pulled, the night failed (`station upgrade` beside each), until a night, or your own `station upgrade` of that Machinefile run to its end, reads none. A night that did not run to its end keeps beside it the last one that did, so what that one moved and left waiting is still said. A `station apply` run to its end reads each waiting version against the ranges it applied, so a line moved to hold one (`'22'` to `'25'`) clears it at once. Only one apply, upgrade or night runs on a machine at a time, the turn a file lock of station's state (`~/.local/state/station/runs.lock`): a run that finds it held waits a few seconds, then is refused (`another station run holds this machine … one runs at a time`), and a night is tried again an hour later; a run interrupted while it waits is interrupted, exit 130, never refused. A Mac bound to no Machinefile has no night: `station apply --machinefile <name>` binds it.

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
| 5. Already present by another way       | adopted by the plan by the method that holds it, never installed again                            |
| 7. Installed outside station            | `not managed` in drift; `apply --remove` never removes it (the ledger)                            |
| 8. Interactive installs                 | `needs`, said at the head of the plan; refused off a terminal                                     |
| 9. A restart needed                     | `after-apply`, said once at the end of the plan and of the run                                    |
| 11. Interruption, partial failure       | one step at a time; what waits on a failure is not run, the rest goes on; `station apply resumes` |
| 14. Secrets                             | a `vault:` reference, read only to hand the setting its value                                     |
| 15. A dotfile edited by hand            | `changed` in drift, left as it is by apply                                                        |
| 16. A macOS update breaks an app        | `changed` in drift, installed again by apply                                                      |
| 17. An old image                        | the lock its image phase wrote, kept in the image record by `images build`                        |
| 18. Upgrade order                       | `station upgrade`: the managers first, then the rest; brew's own index refresh off during both    |

## What is read today

The format is whole now, so a Machinefile a newer station writes in v1 decodes in this one; some keys are declared for the lots that read them and held, unread, until then:

| Key                                                | Read today                                                                                                                                                                                                                                                                                                                                                               | Read by                                      |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------- |
| `machine`, `extends`, `prefer`, `apps`, `settings` | by the resolution, `explain`, `plan`, `apply` and `drift`                                                                                                                                                                                                                                                                                                                |                                              |
| an app's `options`, a setting's values, `vault:`   | checked against the item's options, shown by `explain`; a setting's values handed to it by `apply`                                                                                                                                                                                                                                                                       | an app's options: the lots that read them    |
| `vm`                                               | refused on a computer, its `image.bake` checked against the catalog, shown by `explain`; `image.from` and `image.bake` read by `images build` and `--phase image`                                                                                                                                                                                                        | `machines create --from`                     |
| `accounts`                                         | shown by `explain`                                                                                                                                                                                                                                                                                                                                                       | a VM's first start, the accounts a Mac keeps |
| `ports`                                            | checked (a public port one by one, three at most), shown; the engine reads a VM's by its name (see [Ports](ports.md#rights-and-audiences))                                                                                                                                                                                                                               | the ports' rights and audiences              |
| `description`                                      | decoded                                                                                                                                                                                                                                                                                                                                                                  | the menu bar                                 |
| the overlay                                        | merged by the resolution                                                                                                                                                                                                                                                                                                                                                 | the menu bar, which writes it                |
| the user catalog                                   | by every command: an item of `~/.config/station/catalog/` replaces the built-in one, and the folders its methods land commands in (`path`) are searched; a refused file stops every command that reads the catalog at its line, while `--version`, help and the engine (which the menu bar must keep reaching) run on the built-in items and print the refusal on stderr |                                              |
| the shared source                                  | by `explain`, `plan`, `apply` and `drift`: every other command reads the user and built-in catalog                                                                                                                                                                                                                                                                       |                                              |
