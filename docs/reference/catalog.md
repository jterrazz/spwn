# The catalog's item format

An item of the catalog is one YAML file in the schema `station.catalog/v1`: what a thing is, and how it is installed or set on each system. This page is the format, every key of it; what station does with an app is [Apps and skills](apps-and-skills.md), with a setting [Settings](settings.md).

| Section             | Answers                                                         |
| ------------------- | --------------------------------------------------------------- |
| The file            | Where an item lives, how strictly it is read, its schema        |
| Item keys           | Every key of an item, and which kinds read it                   |
| The selector        | Which machines a method or a disk row is for                    |
| Methods             | The manager keys, and the overrides a method may carry          |
| Options             | The typed values an item asks, and their checks                 |
| One item per method | An example of each manager key, across systems                  |
| What is read today  | The keys declared for later lots, which this station holds only |

## The file

The built-in items are embedded in the binary, `src/catalog/items/<category>/<name>.yaml`: the folder is the item's category (`git`, `ai-agents`, `security`...), the file's name its `name`. Its first key names the schema, `item: station.catalog/v1`.

A file is read strictly, so a typo is a refusal with its line, never a silent default:

- one YAML 1.2 document, every key known (`line 9: field tint not found`);
- no anchor, alias or merge key: a value is written out;
- no plain scalar that YAML 1.1 reads otherwise (`yes`, `on`, `0755`): write `true`, quote it, or `0o755`;
- every enumerated value one of its list;
- a file naming a newer schema, `station.catalog/v2`, is refused with `update station`.

The JSON Schema an editor checks a file against, [`schemas/station.catalog.v1.json`](../../schemas/station.catalog.v1.json), is projected from the decoder's Go types by `make generate`, so the editor and station refuse the same files. A refusal that judges the catalog whole (a requirement that names no app, two items of one name) names the file it is in.

The lists stand by category, in the categories' order (`Package Managers`, `Runtimes`, `Shell & Terminal`... for the apps; `Shell`, `Terminal`, `Security`... for the settings), then by name. A file has no order of its own.

## Item keys

| Key               | Kinds        | Value                                                                                                                                 |
| ----------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `item`            | all          | the schema, `station.catalog/v1`                                                                                                      |
| `kind`            | all          | `manager` (installs other items), `app` (run or used), `setting` (a state of the system)                                              |
| `name`            | all          | the item's name, its file's                                                                                                           |
| `description`     | all          | one line, the owner's words                                                                                                           |
| `requires`        | all          | items installed before it, for it; a setting requires one app                                                                         |
| `after`           | all          | items applied before it when both are, never installed for it                                                                         |
| `phase`           | all          | `image` (baked into a VM image), `boot` (a VM's first start), `apply` (the default)                                                   |
| `needs`           | all          | what it asks of the owner: `sudo`, `terminal`, `license`, `app-store-login`                                                           |
| `after-apply`     | all          | what has to follow for it to take: `restart-dock`, `restart-finder`, `restart-menu-bar`, `logout`, `reboot`                           |
| `renamed-from`    | all          | the names it had before                                                                                                               |
| `deprecated`      | all          | why it goes, and what to take instead                                                                                                 |
| `options`         | all          | the values it asks, typed ([Options](#options))                                                                                       |
| `methods`         | all          | the ordered list of ways to install or set it; the first that matches the machine wins                                                |
| `provides`        | manager, app | the commands it puts on the search path, by which its presence is read                                                                |
| `then`            | manager, app | settings proposed once it is installed                                                                                                |
| `replaces`        | manager, app | the default it replaces, as the toolbelt says it (`cat (for humans)`)                                                                 |
| `bundle`          | manager, app | its `/Applications` bundle, without `.app`                                                                                            |
| `version`         | manager, app | how its installed version is read: `args`, a `pattern`, or a `path` glob                                                              |
| `supersedes`      | manager, app | the formula it replaces, uninstalled once it is in                                                                                    |
| `accepts`         | manager, app | another source whose copy stands for its install, never managed: `app-store`                                                          |
| `live`            | manager, app | how it is read running: a `process` name, or a `command`                                                                              |
| `disk`            | manager, app | its Disk row ([Disk](disk.md)): `paths`, `installed`, `removal`, `size`, `free`, `erase`, `no-erase`, `quit`, `relocate`, `misplaced` |
| `help`            | setting      | the hint a form shows                                                                                                                 |
| `requires-one-of` | setting      | apps of which any one serves                                                                                                          |
| `guided`          | setting      | it opens System Settings for the owner to finish                                                                                      |
| `reversible`      | setting      | `station settings off` takes it back                                                                                                  |
| `run-once`        | setting      | an action, never judged after it ran                                                                                                  |
| `declares`        | setting      | the port it opens to the machine's network: `port`, `via`, `label`                                                                    |
| `sets`            | setting      | the values it sets, as its check reads them back: `key`, `value`                                                                      |
| `writes`          | setting      | the files it writes, each a `path`, its `level`, a `note`, and a selector                                                             |

## The selector

A selector says which machines a method, or one disk row, is for. Each key narrows; an absent key keeps every machine:

| Key       | Values                                                     |
| --------- | ---------------------------------------------------------- |
| `os`      | `macos`, `ubuntu`, `debian`, `any` (every system)          |
| `family`  | `linux` (every Linux), `debian` (Debian and Ubuntu)        |
| `version` | a range of the os's versions: `>=24.04`, `>=14, <27`, `15` |
| `arch`    | `arm64`, `amd64`                                           |
| `kind`    | `computer`, `workspace`, `server`, or a list of them       |

A method names its system by `os` or by `family`, one of them; a version range belongs to one os, never to `any` nor to a family. A disk row may carry no selector at all: it is then on every machine.

```yaml
- os: macos
  kind: computer # the Mac itself, not a macOS VM
  cask: tailscale-app
- family: linux
  kind: [workspace, server]
  mise: { name: lazygit, version: latest }
```

Station's computers today are Macs and its VMs run on them, so `os: macos` is a Mac and a macOS VM, `family: linux` and `os: ubuntu` a Linux VM. `os: debian`, a Linux computer and `arch: amd64` match no machine station runs yet, and a version range is not judged before the Machinefile's resolution reads a machine's system.

## Methods

A method is a selector, exactly one manager key, and the overrides that keep inside it.

| Manager key      | Kinds        | Value                                                                                                                                  |
| ---------------- | ------------ | -------------------------------------------------------------------------------------------------------------------------------------- |
| `brew`           | manager, app | the formula                                                                                                                            |
| `cask`           | manager, app | the cask                                                                                                                               |
| `mas`            | manager, app | `id` and `name` in the App Store                                                                                                       |
| `apt`            | manager, app | the package; on the debian family                                                                                                      |
| `npm`            | manager, app | the package                                                                                                                            |
| `uv`             | manager, app | the package                                                                                                                            |
| `bun`            | manager, app | the package, or the name of what `package` fetches                                                                                     |
| `mise`           | manager, app | `name` and `version`                                                                                                                   |
| `github-release` | manager, app | `repo` (`owner/name`), `asset` (a pattern, `{os}` and `{arch}` the machine's), `checksums`                                             |
| `script`         | manager, app | `check`, then `run` (a shell line) or `url` (an installer, with `args`, `env`, `on-terminal`, `off-terminal`), `undo`; or `hint` alone |
| `skill`          | manager, app | `repo` and `name`                                                                                                                      |
| `provided-by`    | manager, app | the item whose install brings its commands                                                                                             |
| `file`           | setting      | `touch` a path, or `copy` embedded files (`from`, `to`, `mode`) and `reload`                                                           |
| `builtin`        | setting      | a procedure station carries (`dns`, `ssh-key`...)                                                                                      |
| `macos-defaults` | setting      | a `domain`, `this-mac` (its keys of this Mac alone), and its `keys`, each `key`, `type` and a `value` or an `option`                   |
| `git-aliases`    | setting      | the git config `file` and its `aliases`                                                                                                |
| `zsh-functions`  | setting      | the embedded script `from`, and its `functions`                                                                                        |
| `agent-hooks`    | setting      | the `apps` whose agents are hooked                                                                                                     |
| `mux-machines`   | setting      | `{}`                                                                                                                                   |

Every manager type has its own check, upgrade and undo; an override is written only where that does not fit, and only on an app's method:

| Override   | Value                                                                      |
| ---------- | -------------------------------------------------------------------------- |
| `provides` | the commands this method lands when they differ from the item's (`fdfind`) |
| `package`  | the source bun fetches when it differs from the name (`github:tobi/qmd`)   |
| `upgrade`  | the shell line that upgrades what it installed                             |
| `pin`      | what it can pin: `none` or `version`                                       |
| `path`     | absolute folders it lands commands in, which station's search adds at once |
| `trusted`  | a brew or a cask from a third-party tap the owner trusts                   |

A `script` with neither `run` nor `url` is installed by the owner, as its `hint` says; a `run` line takes no `args`, `env` nor `upgrade`. A `script`'s `check` is the shell line whose exit 0 says the item is there; without one, its `provides` are looked up.

## Options

An option is a value the owner gives an item, a form field in the menu bar. Its `name`, `label`, `help` and `type`, then the keys its type allows:

| Type      | Keys                                               |
| --------- | -------------------------------------------------- |
| `string`  | `default`, `examples`, `secret` (never shown back) |
| `integer` | `default`, `examples`, `min`, `max`                |
| `boolean` | `default`                                          |
| `enum`    | `values` (required), `default`, `examples`         |

Every type takes `required` and `default-env` (the environment variable whose value is the default). A default or an example is a value of its type, within an integer's bounds, one of an enum's values. A form asks a secret string hidden as it is typed, an enum as a choice, a boolean as a yes or no.

## One item per method

Each block is one file; the first method that matches a machine wins.

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
      apt: fd-find
    - family: linux
      arch: arm64
      pin: version
      github-release: { repo: sharkdp/fd, asset: 'fd-*-aarch64-unknown-linux-gnu.tar.gz' }
```

```yaml
# a cask on the Mac itself, a mas app whose install needs the App Store signed in
item: station.catalog/v1
kind: app
name: speedtest
description: Internet speed test
bundle: Speedtest
needs: [app-store-login]
methods:
    - os: macos
      kind: computer
      mas: { id: 1153157709, name: Speedtest }
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
  kind: computer
  script:
      hint: download it from its website
```

```yaml
# a skill, and a command another item's install brings
- os: any
  skill: { repo: shadcn-ui/ui, name: shadcn }
- os: macos
  kind: computer
  provided-by: herdr-app
```

```yaml
# a setting of macOS defaults, grouped by subject, its keys read from its options
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
              - { key: autohide, type: boolean, option: autohide }
              - { key: tilesize, type: integer, option: size }
              - { key: orientation, type: string, option: position }
```

```yaml
# the other setting methods
- os: any
  file: { touch: ~/.hushlogin }
- os: macos
  kind: computer
  builtin: dns
- os: any
  git-aliases: { file: ~/.gitconfig, aliases: [{ name: st, command: status }] }

- os: any
  zsh-functions: { from: zsh/docker-shortcuts.zsh, functions: [dclean] }
- os: any
  kind: [workspace, server]
  agent-hooks: { apps: [claude, codex] }
- os: macos
  kind: computer
  mux-machines: {}
```

## What is read today

The format is whole now, so a file a newer station writes in v1 decodes in this one; some keys are declared for the lots that read them and held, unread, until then:

| Key                                                                 | Read by                                                     |
| ------------------------------------------------------------------- | ----------------------------------------------------------- |
| `after`, `phase`, `renamed-from`, `deprecated`, `after-apply`       | the Machinefile's plan and apply                            |
| `needs: license`, `needs: app-store-login`                          | the plan, at its head                                       |
| the selector's `version`                                            | the Machinefile's resolution, against a machine             |
| `pin`, `upgrade` on a method other than a `script`'s `url`          | the lockfile and `station upgrade`                          |
| a method's `provides`                                               | presence, which reads the item's today                      |
| `script`'s `check` and `undo`                                       | presence and the ledger's undo                              |
| an app's `options`, an option's `default`, `min`, `max`, `examples` | the forms                                                   |
| `apt`, `github-release`                                             | their installers: until then the next method serves         |
| `macos-defaults`                                                    | its Applier: until then applying it is refused as unhandled |
