# The catalog's item format

An item of the catalog is one YAML file in the schema `station.catalog/v1`: what a thing is, and how it is installed or set on each system. This page is the format, every key of it; what station does with an app is [Apps and skills](apps-and-skills.md), with a setting [Settings](settings.md).

| Section             | Answers                                                         |
| ------------------- | --------------------------------------------------------------- |
| The file            | Where an item lives, how strictly it is read, its schema        |
| Item keys           | Every key of an item, and which kinds read it                   |
| The selector        | Which machines a method or a disk row is for                    |
| Methods             | The manager keys, and the overrides a method may carry          |
| Managers            | What a manager installs by, and each system's priority          |
| Options             | The typed values an item asks, and how a value reads one        |
| One item per method | An example of each manager key, across systems                  |
| What is read today  | The keys declared for later lots, which this station holds only |

## The file

The built-in items are embedded in the binary, `apps/engine/catalog/items/<category>/<name>.yaml`: the folder is the item's category (`git`, `ai-agents`, `security`...), the file's name its `name`. Its first key names the schema, `item: station.catalog/v1`. The owner's own items, `~/.config/station/catalog/<category>/<name>.yaml`, and a shared source's are layers above them: a same-name item replaces the built-in one whole, never merged ([Machinefiles](machinefiles.md#sources-and-layers)), and an owner's file refused stops every command that reads the catalog at its line ([what is read today](machinefiles.md#what-is-read-today)).

A file is read strictly, so a typo is a refusal with its line, never a silent default:

- the schema first: a file naming a newer one, `station.catalog/v2`, is refused with `update station` before anything else is judged;
- one YAML 1.2 document, every key known (`line 9: field tint not found`);
- no anchor, alias or merge key: a value is written out;
- no plain scalar that YAML 1.1 reads otherwise (`yes`, `on`, `0755`): write `true`, quote it, or `0o755`;
- every enumerated value one of its list, every check of this page passed.

The JSON Schema an editor checks a file against, [`schemas/station.catalog.v1.json`](../../schemas/station.catalog.v1.json), is projected from the decoder's Go types by `make generate`. It checks the keys, their types, the required ones and the enumerated values. Only station checks the rest: a key that belongs to another kind, a method's one manager key, an option's default against its type, a version range, an `${option.<name>}` reference, and what judges the catalog whole (a requirement that names no app, two managers of one method key), whose refusal names the file it is in.

The lists stand by category, in the categories' order (`Package Managers`, `Runtimes`, `Shell & Terminal`... for the apps; `Shell`, `Terminal`, `Security`... for the settings), then by name. A file has no order of its own.

## Item keys

| Key               | Kinds        | Value                                                                                                                                                                                                                              |
| ----------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `item`            | all          | the schema, `station.catalog/v1`                                                                                                                                                                                                   |
| `kind`            | all          | `manager` (installs other items), `app` (run or used), `setting` (a state of the system)                                                                                                                                           |
| `name`            | all          | the item's name, its file's                                                                                                                                                                                                        |
| `description`     | all          | one line, the owner's words                                                                                                                                                                                                        |
| `requires`        | all          | items installed before it, for it; a setting requires one app                                                                                                                                                                      |
| `after`           | all          | items applied before it when both are, never installed for it                                                                                                                                                                      |
| `phase`           | all          | `image` (baked into a VM image), `boot` (a VM's first start), `apply` (the default)                                                                                                                                                |
| `needs`           | all          | what every method asks of the owner: `sudo`, `terminal`, `license`, `app-store-login`                                                                                                                                              |
| `after-apply`     | all          | what has to follow for it to take: `restart-dock`, `restart-finder`, `restart-menu-bar`, `logout`, `reboot`                                                                                                                        |
| `renamed-from`    | all          | the names it had before                                                                                                                                                                                                            |
| `deprecated`      | all          | `why` it goes (required), and the item to `use` instead                                                                                                                                                                            |
| `options`         | all          | the values it asks, typed ([Options](#options))                                                                                                                                                                                    |
| `methods`         | all          | the ordered list of ways to install or set it; the first that matches the machine wins                                                                                                                                             |
| `manages`         | manager      | the method keys it installs by ([Managers](#managers))                                                                                                                                                                             |
| `priority`        | manager      | where it stands in each system's default priority: `macos`, `ubuntu`, `debian`                                                                                                                                                     |
| `provides`        | manager, app | the commands it puts on the search path, by which its presence is read; each is one file name: no `/`, never `.` nor `..`, never opening with `-`                                                                                  |
| `then`            | manager, app | settings proposed once it is installed                                                                                                                                                                                             |
| `replaces`        | manager, app | the default it replaces, as the toolbelt says it (`cat (for humans)`)                                                                                                                                                              |
| `bundle`          | manager, app | its `/Applications` bundle, without `.app`                                                                                                                                                                                         |
| `version-read`    | manager, app | how its installed version is read: `args`, a `pattern`, or a `path` glob                                                                                                                                                           |
| `supersedes`      | manager, app | the formula it replaces, uninstalled once it is in                                                                                                                                                                                 |
| `accepts`         | manager, app | another source whose copy stands for its install, never managed: `app-store`                                                                                                                                                       |
| `updates`         | manager, app | who moves its version: `self`, its own updater (Claude, Ghostty, a browser): station checks it is there, never reads it behind, never updates it; its lock line keeps the version it was installed at, for a new Mac to start from |
| `live`            | manager, app | how it is read running: a `process` name, or a `command`                                                                                                                                                                           |
| `ports`           | manager, app | the ports it serves on, each a `port` and its `label`, a hint a listening port is shown with ("probably Vite"), and what a shared port sets: `http`, `env`                                                                         |
| `disk`            | manager, app | its Disk row ([Disk](disk.md)): `paths`, `installed`, `removal`, `size`, `free`, `erase`, `no-erase`, `quit`, `relocate`, `misplaced`                                                                                              |
| `help`            | setting      | the hint a form shows                                                                                                                                                                                                              |
| `requires-one-of` | setting      | apps of which any one serves, never beside `requires`                                                                                                                                                                              |
| `guided`          | setting      | it opens System Settings for the owner to finish                                                                                                                                                                                   |
| `reversible`      | setting      | `station settings off` takes it back                                                                                                                                                                                               |
| `run-once`        | setting      | an action, never judged after it ran                                                                                                                                                                                               |
| `declares`        | setting      | the port it opens to the machine's network: `port`, `via`, `label`                                                                                                                                                                 |
| `sets`            | setting      | the values it sets, as its check reads them back: `key`, `value`                                                                                                                                                                   |
| `writes`          | setting      | the files it writes, each a `path`, its `level`, a `note`, and a selector                                                                                                                                                          |

A disk action (`free`, `erase`, a misplaced row's `free`) runs a command, deletes or prunes paths: `run`, `env`, `delete`, `prune`, `uses` (the command it runs with, which must be on the search path), `exists`, and a selector.

A port's `env` maps a variable name to a template the app's server is started with once the port is shared: `{url}` is the shared URL and `{host}` the name it is reached by, the only two braces a value may hold (any other is refused at decode); `http: true` marks a dev server that speaks HTTP.

## The selector

A selector says which machines a method, or one disk row, is for. Each key narrows; an absent key keeps every machine:

| Key       | Values                                                     |
| --------- | ---------------------------------------------------------- |
| `os`      | `macos`, `ubuntu`, `debian`, `any` (every system)          |
| `family`  | `linux` (every Linux), `debian` (Debian and Ubuntu)        |
| `version` | a range of the os's versions: `>=24.04`, `>=14, <27`, `15` |
| `arch`    | `arm64`, `amd64`                                           |
| `machine` | `computer`, `workspace`, `server`, or a non-empty list     |

A method names its system by `os` or by `family`, one of them; a version range belongs to one os, never to `any` nor to a family. A disk row may carry no selector at all: it is then on every machine.

```yaml
- os: macos
  machine: computer # the Mac itself, not a macOS VM
  cask: tailscale-app
- family: linux
  machine: [workspace, server]
  arch: arm64
  github-release: { repo: jesseduffield/lazygit, asset: 'lazygit_*_linux_arm64.tar.gz' }
```

Station's computers today are Macs and its VMs run on them, so `os: macos` is a Mac and a macOS VM, `family: linux` and `os: ubuntu` a Linux VM. `os: debian`, a Linux computer and `arch: amd64` match no machine station runs yet, and a version range is not judged before the Machinefile's resolution reads a machine's system.

## Methods

A method is a selector, exactly one manager key, and the overrides that keep inside it.

| Manager key      | Kinds        | Value                                                                                                                                                                                                                                                                                                                                              |
| ---------------- | ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `brew`           | manager, app | the formula                                                                                                                                                                                                                                                                                                                                        |
| `cask`           | manager, app | the cask                                                                                                                                                                                                                                                                                                                                           |
| `mas`            | manager, app | `id` and `name` in the App Store                                                                                                                                                                                                                                                                                                                   |
| `apt`            | manager, app | its `packages`, a list of package names (lowercase letters, digits and `+.-`, from a letter or digit, an `:arch` at most); on the debian family alone (`family: debian`, `os: ubuntu`, `os: debian`), and its method says `needs: [sudo]`                                                                                                          |
| `npm`            | manager, app | the package                                                                                                                                                                                                                                                                                                                                        |
| `uv`             | manager, app | the package                                                                                                                                                                                                                                                                                                                                        |
| `bun`            | manager, app | the package, or the name of what `package` fetches                                                                                                                                                                                                                                                                                                 |
| `mise`           | manager, app | `name` and `version`, what `station apps install` hands mise (`lts`, `temurin-21`); a Machinefile's range replaces it: every version is `latest`, or the vendor alone where it names one (`temurin`), and a range keeps that vendor (`temurin-17`)                                                                                                 |
| `github-release` | manager, app | `repo` (`owner/name`: letters, digits and `-`, the name `._` too, never `.` nor `..`), `asset` (a pattern, `{os}` and `{arch}` the machine's), `checksums` (the asset of checksums, when the release names it otherwise), `unverified` (why a release with no checksum is installed all the same); the commands it lands are the item's `provides` |
| `script`         | manager, app | `check`, then `run` (a shell line) or `url` (an installer, with `args`, `env`, `on-terminal`, `off-terminal`), `undo`; or `hint` alone                                                                                                                                                                                                             |
| `skill`          | manager, app | `repo` and `name`                                                                                                                                                                                                                                                                                                                                  |
| `provided-by`    | manager, app | the item whose install brings its commands                                                                                                                                                                                                                                                                                                         |
| `file`           | setting      | `touch` a path, or `copy` embedded files (`from`, `to`, `permissions`), its `mode` (`file`: the whole file; `block`: station's delimited block in a file the owner keeps), `reload`                                                                                                                                                                |
| `builtin`        | setting      | a procedure station carries (`dns`, `ssh-key`...); one this station does not carry decodes, and its apply says `update station`                                                                                                                                                                                                                    |
| `macos-defaults` | setting      | a `domain`, `this-mac` (its keys of this Mac alone), and its `keys`, each `key`, `type` (`string`, `integer`, `float`, `boolean`), `value`                                                                                                                                                                                                         |
| `git-aliases`    | setting      | the git config `file` and its `aliases`                                                                                                                                                                                                                                                                                                            |
| `shell`          | setting      | the identified blocks of `~/.zshrc` the setting writes, `id` and `zsh` each (alone, it is the setting's effect; beside an app's manager key it is a modifier, below)                                                                                                                                                                               |
| `agent-hooks`    | setting      | the `apps` whose agents are hooked                                                                                                                                                                                                                                                                                                                 |
| `mux-machines`   | setting      | `{}`                                                                                                                                                                                                                                                                                                                                               |

A setting, like an app, may carry a method per system.

Every manager type has its own check, upgrade and undo; an override is written only where that does not fit:

| Override   | On                   | Value                                                                              |
| ---------- | -------------------- | ---------------------------------------------------------------------------------- |
| `needs`    | any method           | what this method alone asks of the owner (apt's `sudo`, mas's `app-store-login`)   |
| `provides` | an app's             | the commands this method lands when they differ from the item's (`fdfind`)         |
| `package`  | a bun's              | the source bun fetches when it differs from the name (`github:tobi/qmd`)           |
| `upgrade`  | an app's             | the shell line that upgrades what it installed                                     |
| `pin`      | an app's             | what it can hold: `none` (only its newest) or `version` (what its install carries) |
| `path`     | an app's             | absolute folders it lands commands in, which station's search adds at once         |
| `trusted`  | a brew's or a cask's | a formula or a cask from a third-party tap the owner trusts                        |
| `shell`    | an app's             | the blocks of `~/.zshrc` this method writes once the app is there                  |

A `script` with neither `run` nor `url` is installed by the owner, as its `hint` says; a `run` line takes no `args`, `env` nor `upgrade`. An `env` value that opens with `~/` is given under the home (`HELM_INSTALL_DIR=~/.local/bin`), since an installer reads its variables as they are. A `script`'s `check` is the shell line whose exit 0 says the item is there, run with `~/.local/bin`, the folder a script lands its commands in, as its whole PATH, so it answers for the copy station installed and never for one ahead of it on yours (a mise shim); without one, its `provides` are looked up. The check's output is read for a version by the item's `version-read` `pattern` (`pattern: 'Terraform v(\d+\.\d+\.\d+)'` with `check: CHECKPOINT_DISABLE=1 terraform version`), unless the item reads its version by a command of its own (`version-read` `args`). Its `undo` is the shell line that takes the app back: `station apps remove` and `station apply --remove` run it, and without one station cannot remove the app (`remove it by hand`). Station runs both as it runs a `run` line, `bash -c <line>` from the home, with no value interpolated into it; a `url` script's `undo` runs with its `env`, and a `check` line still running at its 5 seconds is stopped with everything it started. A `run` line or a `url` installer is also its own update: `station apps update <app>` runs it again.

## Managers

A manager item says the method keys it installs by, `manages`, each key one manager's, and where it stands in each system's default priority, `priority`, lowest first, each rank one manager's. The first manager of a system's order that has a method for an item installs it, unless a Machinefile says `prefer:`, and the order is one: `apps install` and a Machinefile's resolution choose by it alike, so the method a machine carries an app by is the one its plan shows; a manager with no priority is chosen by its method alone (mise for a runtime). A `script` method has no manager and no rank: it is always tried after every manager of the system, and a `github-release` method takes the rank of the item that manages it. A `priority` ranks at least one system.

```yaml
item: station.catalog/v1
kind: manager
name: homebrew
description: macOS package manager
manages: [brew, cask]
priority: { macos: 1 }
provides: [brew]
```

## Options

An option is a value the owner gives an item, a form field in the menu bar. Its `name`, `label`, `help` and `type`, then the keys its type allows:

| Type      | Keys                                               |
| --------- | -------------------------------------------------- |
| `string`  | `default`, `examples`, `secret` (never shown back) |
| `integer` | `default`, `examples`, `min`, `max`                |
| `boolean` | `default` alone: a boolean has no examples         |
| `enum`    | `values` (required), `default`, `examples`         |

Every type takes `required` and `default-env` (the environment variable whose value is the default). A default or an example is a value of its type, within an integer's bounds, one of an enum's values. A form asks a secret string hidden as it is typed, an enum as a choice, a boolean as a yes or no.

A method's value reads an option as `${option.<name>}`, in any of its strings: a script's line, an env pair, a copied file's path, a defaults value. A reference that names no option of the item is refused at decode. The braces without a dollar (`{cache}`, `{state}`, `{os}`, `{arch}`) are station's own placeholders, never an option, and a shell's `${HOME}` stays the shell's.

## One item per method

Each block is one file, or the methods of one; the first method that matches a machine wins.

```yaml
# brew on macOS, apt on Ubuntu with Debian's own command name, github-release elsewhere on Linux
item: station.catalog/v1
kind: app
name: fd
description: Fast find alternative
provides: [fd]

methods:
    - os: macos
      brew: fd
    - os: ubuntu
      version: '>=22.04'
      provides: [fdfind]
      needs: [sudo]
      apt: { packages: [fd-find] }
    - family: linux
      arch: arm64
      pin: version
      github-release: { repo: sharkdp/fd, asset: 'fd-*-aarch64-unknown-linux-gnu.tar.gz' }
```

```yaml
# a mas app on the Mac itself, whose install needs the App Store signed in
item: station.catalog/v1
kind: app
name: speedtest
description: Internet speed test
bundle: Speedtest
methods:
    - os: macos
      machine: computer
      needs: [app-store-login]
      mas: { id: 1153157709, name: Speedtest }
```

```yaml
# a dev server's port: its label, and what it sets once shared
item: station.catalog/v1
kind: app
name: vite
description: Frontend build tool and dev server
ports:
    - port: 5173
      label: Vite
      http: true
      env: { __VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS: '{host}' }
methods:
    - os: any
      bun: vite
```

```yaml
# npm and uv on macOS, bun and mise everywhere
- os: macos
  npm: '@playwright/mcp'
- os: macos
  uv: markitdown[all]
- os: any
  package: github:tobi/qmd
  bun: qmd
- os: any
  mise: { name: node, version: lts }
```

```yaml
# a script: an installer fetched and run, its own upgrade; and one the owner installs
- os: any
  upgrade: ZSH="$HOME/.oh-my-zsh" exec sh "$HOME/.oh-my-zsh/tools/upgrade.sh"
  script:
      check: test -d "$HOME/.oh-my-zsh"
      url: https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh
      env: [RUNZSH=no, CHSH=no, KEEP_ZSHRC=yes]
- os: macos
  machine: computer
  script:
      hint: download it from its website
```

```yaml
# a skill, and a command another item's install brings
- os: any
  skill: { repo: shadcn-ui/ui, name: shadcn }
- os: macos
  machine: computer
  provided-by: herdr-app
```

```yaml
# a setting of macOS defaults, grouped by subject, its values read from its options
item: station.catalog/v1
kind: setting
name: macos-dock
description: The Dock hides itself, at the size and on the side asked
reversible: true
after-apply: [restart-dock]

options:
    - { name: autohide, label: Hide the Dock, type: boolean, default: true }
    - { name: size, label: Tile size, type: integer, min: 16, max: 128, default: 48 }
    - { name: position, label: Position, type: enum, values: [left, right], default: left }

methods:
    - os: macos
      macos-defaults:
          domain: com.apple.dock
          keys:
              - { key: autohide, type: boolean, value: '${option.autohide}' }
              - { key: tilesize, type: integer, value: '${option.size}' }
              - { key: orientation, type: string, value: '${option.position}' }
```

```yaml
# the other setting methods, a file written whole or as station's block
- os: any
  file: { touch: ~/.hushlogin }
- os: any
  file: { copy: [{ from: zsh/aliases.zsh, to: ~/.zshrc }], mode: block }
- os: macos
  machine: computer
  builtin: dns

- os: any
  git-aliases: { file: ~/.gitconfig, aliases: [{ name: st, command: status }] }
- os: any
  shell: [{ id: station:docker-shortcuts, zsh: 'dclean() { docker rm -vf $(docker ps -aq); }' }]
- os: any
  machine: [workspace, server]
  agent-hooks: { apps: [claude, codex] }
- os: macos
  machine: computer
  mux-machines: {}
```

### The shell key

`shell` is a list of `{ id, zsh }`, one per block of `~/.zshrc` a method writes: on an app's method, beside its manager key, the blocks that go with the app; on a setting's method, alone, the setting's effect. The `id` is stable for life and names the block's two marker lines: `station:<item>`, or `station:<item>/<sub>` when an item writes several, never another item's name, written once in a method. The `zsh` is plain text: shell variables stay the shell's (`$HOME`, `$HOMEBREW_PREFIX`, resolved at each shell start), and station's variables between double braces are filled once, when apply writes the block, by plain substitution of a fixed set of names and nothing evaluated.

| Variable              | Is                                                                                         |
| --------------------- | ------------------------------------------------------------------------------------------ |
| `{{ brew.prefix }}`   | Homebrew's prefix on this machine: `/opt/homebrew` on Apple silicon, `/usr/local` on Intel |
| `{{ home }}`          | the machine's home folder                                                                  |
| `{{ os }}`            | the machine's system: `macos`, `ubuntu` or `debian`                                        |
| `{{ arch }}`          | the machine's architecture: `arm64` or `amd64`                                             |
| `{{ option.<name> }}` | an app's option, as the Machinefile gives it, else its default; never a secret option      |

A name outside this set, an option the item does not declare or a secret one, a marker line inside `zsh` and a lone `{{` are refused at decode; an option with no value, neither given nor defaulted, is refused when the plan is made, never written as an empty string. A setting's block takes the machine's variables alone, since its values arrive at apply. What the blocks do in the file, their order, their removal and a hand edit are [Dotfiles](dotfiles.md#package-blocks)'s.

## What is read today

The format is whole now, so a file a newer station writes in v1 decodes in this one; some keys are declared for the lots that read them and held, unread, until then:

| Key                                                                                     | Read by                                                                                                           |
| --------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `after`, `phase`, `renamed-from`, `deprecated`, `after-apply`                           | the Machinefile's plan and apply                                                                                  |
| `needs: license`, `needs: app-store-login`, a method's `needs`                          | the plan, at its head (an item's `needs` is said at its install today)                                            |
| `ports`                                                                                 | the engine: a reserved port's `http`, a VM's env at each run, a row's label ([Ports](ports.md#what-an-item-sets)) |
| the selector's `version`                                                                | the Machinefile's resolution reads it against the system, and `explain`; `apps install` does not yet              |
| `pin`, `upgrade` on a method other than a `script`'s `url`                              | `pin` by the resolution (a range its method cannot hold is refused); `upgrade` by `station upgrade`               |
| a version asked of `npm` or `uv`, a `pin: version` on a method station runs             | npm's and uv's installs ride brew's bundle, whose lines carry no version: a range asked of them is refused        |
| a method's `provides`                                                                   | presence, which reads the item's today                                                                            |
| a file's `mode`                                                                         | the dotfile writes (whole files today)                                                                            |
| an app's `options`, an option's `default`, `min`, `max`, `examples`, `${option.<name>}` | the forms and the apply                                                                                           |
| a setting's methods after its first                                                     | the settings of each system: today it lives where its first method does                                           |
| `macos-defaults`                                                                        | its Applier: until then applying it is refused as unhandled                                                       |
