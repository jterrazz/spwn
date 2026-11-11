# Apps and skills

An app is installed content: software or an agent skill (v2 had its own word for it). The catalog declares each one in a file of its own, and `station apps`, `station overview` and the toolbelt skill all read it.

| Section        | Answers                                                |
| -------------- | ------------------------------------------------------ |
| The catalog    | How an app is declared, and where it lives             |
| Methods        | How each method installs, removes, updates and is read |
| Runtimes       | Why mise owns node, go, python, bun, rust and java     |
| Skills as apps | Agent skills installed with `station apps install`     |
| The toolbelt   | The committed projection agents read                   |

## The catalog

An app is one item file, `src/catalog/items/<category>/<name>.yaml`, in the schema `station.catalog/v1`: its `kind` (`manager` for one that installs other apps, `app` otherwise), name, description, its category (the folder), the commands it `provides`, what it `requires`, its `methods` and its disk row. A method is a selector (`os`, `family`, `version`, `arch`, `kind`) and exactly one manager key, the first method that matches the machine winning. Every key of the format, and how strictly a file is read, is [The catalog's item format](catalog.md):

```yaml
item: station.catalog/v1
kind: app
name: gh
description: GitHub CLI for repository management
provides: [gh]
then: [github-login]
methods:
    - os: macos
      brew: gh
    - family: linux
      mise: { name: gh, version: latest }
```

The lists stand by category, then by name. Where it lives, the kinds of machine it may be on (`mac`, `linux`, `macos`), is derived from its methods' selectors, so it cannot disagree with them; `station apps --json` says it as `lives_on` (`every-machine`, `mac-only`, `vms-only`). Dev apps live in VMs: Xcode, CocoaPods, XcodeGen, Android Studio and Lens are placed `macos` only and install nowhere until the macOS VM exists ([TODO](../../TODO.md)); Ollama stays on Macs for the GPU; Docker lives in the Linux VM.

An app asked for on a machine it does not live on is refused with exit 2, `dia lives on Macs only`, the hint the command with `--on`.

An app's state is one of OK (the manager of one of its methods here lists it, current: the method it carries by default, else another it has here, which its row then names and which `update` and `remove` reach), Update available (its manager says a newer version exists), Failed, Not installed, Not managed (found on PATH or in `/Applications` but installed by none of its methods: `ghostty.app in /Applications, not installed by cask`) or Not known (the manager did not answer). `Requires` orders an install and brings what is missing into it, the method's manager too (`apps install keynote` installs mas first, `● mas  required by keynote`); a requirement the catalogue does not know, or this machine cannot carry, refuses the install before anything runs, in the words of asking it by name (`no app named "y"`, `y lives on Macs only`). The catalogue's tests hold that a requirement, and a method's manager, live wherever their app does.

The method an app is installed by is the first of its methods here in the system's order ([Managers](catalog.md#managers)), the one a Machinefile's plan shows for it unless it asks another; `station apps move <app> --to <manager>` switches an installed app to another of its methods, installing by that one before it uninstalls the old copy ([Commands](commands.md#station-apps)).

## Methods

One registry, `Installers` in `src/apps/contract/`, holds an installer per method, each its own adapter under `src/apps/adapter/<method>/` with its commands and its parser; nothing else switches on a method. `wire` builds the registry; `src/apps/calls` asks each installer what it can do and what is behind (`Outdated`): brew reads `brew outdated --json=v2`, a pinned formula left out; a skill compares its lock with GitHub.

| Method            | Install                                                            | Remove                                           | Update                           | Kinds      |
| ----------------- | ------------------------------------------------------------------ | ------------------------------------------------ | -------------------------------- | ---------- |
| `brew`, `cask`    | `brew bundle` (Brewfile on stdin)                                  | `brew uninstall`                                 | `brew bundle --upgrade`          | Mac, macos |
| `mas`             | `mas install <id>`, silent 60 s: signed out                        | `mas uninstall`                                  | `mas update <id>`                | Mac, macos |
| `npm`, `uv`       | `brew bundle`                                                      | `npm uninstall -g`, `uv tool uninstall`          | `brew bundle --upgrade`          | Mac, macos |
| `bun`             | `bun install -g`                                                   | `bun remove -g`                                  | `bun update -g`                  | all        |
| `mise`            | `mise use --global <tool>@<version>`                               | `mise unuse --global --no-prune`                 | `mise upgrade`                   | all        |
| `apt`             | `apt-get update`, then `apt-get install`, as root                  | `apt-get remove`, never what it would widen to   | `apt-get install --only-upgrade` | Linux      |
| `github-release`  | station fetches the release's asset, checked, into `~/.local/bin`  | the commands it put there, while still its own   | the install again, at the latest | all        |
| `script`, `shell` | the vendor's installer                                             | not reversible                                   | rerun the script; none for shell | all        |
| `app`             | a `script` with a `hint` alone: the refusal names how              | not reversible                                   | none                             | Mac, macos |
| `skill`           | `skills add <repo> -g -y --skill <name> --agent claude-code codex` | `skills remove -g -y <name>`                     | the install again                | all        |
| `provided-by`     | the item that provides it, first; then its commands are looked up  | not reversible: remove the item that provides it | none                             | all        |

A `script` install is checked once it ran: its exit code is all the vendor's installer reports, so an install that leaves none of the commands the app provides is Failed, `installed, but brew is not found on the search path`, never OK. The search path is not frozen at the process's start: commands are looked up on disk at each call, in the folders installs land in as well (`/opt/homebrew/bin` and `/usr/local/bin`, read under the system root; `~/.local/bin`; mise's shims), and a child is given a PATH that carries them, so a Homebrew or a script installed by an earlier step of the run is the one the next step finds by name, and a later run finds it too.

Every command station runs for apps (a method's read, install, update and removal, and an app's version read) runs from your home, never from the folder you started station in, and in an environment station builds rather than the one your shell gave it. Its PATH, which is also where the command is looked up, is mise's shims first, then your PATH without a runtime's own folder: mise's installs, which `mise activate` puts first in a project, and nvm's and asdf's versions. mise's profile (`MISE_ENV`, `MISE_PROFILE`), a version named in the environment (`MISE_NODE_VERSION`), `mise activate`'s session and npm's `npm_config_prefix` are left out. mise is pinned to its global file (`MISE_CEILING_PATHS` set to your home) and its auto-install is off (`MISE_AUTO_INSTALL`, `MISE_EXEC_AUTO_INSTALL` and `MISE_NOT_FOUND_AUTO_INSTALL` false). npm's read and removal also name the folder of each node mise holds globally (`npm ls -g --prefix <folder>`, `npm uninstall -g --prefix <folder>`, once per installed version `mise ls --global` gives, since `mise unuse --global node` takes back all of them), so they address those nodes' packages whatever npm runs, and one folder whose list does not read leaves them all unread; with no node of mise's, npm's own prefix is read. A project that pins another node therefore never picks which npm answers, in a shell where mise is activated or under its shims alike, a shim never installs a runtime during a read, and what is read is what `mise use --global` and `mise unuse --global` act on. Removal passes `--no-prune`, so it only edits the global file and prunes no installed version. Whether an app is on this machine another way is no manager's answer: it is looked for on your own PATH, so a node only nvm or asdf holds is present another way (adopted by a plan, never installed again). A hit in mise's own folders (its shims, its installs) is mise's for an app mise carries on this machine, read by its listing, never a copy present another way: with no global node of mise's, a shim or a project's node leaves node absent, and a Machinefile's global node is installed. An app mise does not carry here, found there (one you put there with `mise use --global`), is present another way: adopted by a plan, never installed again. The version is read from the copy the lookup found, by its path, from your home, so it answers for that copy and never for another one a manager's PATH finds first (macOS's `/usr/bin/python3` stub against a shim).

An `apt` app is read by `dpkg-query`: a package dpkg says installed (`ii`) is held at its version, one removed with its configuration kept (`rc`) is not, and it is behind when `apt-cache policy` gives another candidate than the version installed. Each install and update refreshes apt's lists first. Every change runs as root and asks nothing: `sudo env DEBIAN_FRONTEND=noninteractive apt-get -y`, a configuration file the owner changed kept (`--force-confold`), and `sudo -n` off a terminal, so a sudo that would ask a password fails, `a password is required`, rather than wait; on a terminal it may ask. A method by apt says `needs: [sudo]`, so a Machinefile's plan asks the password once, where this machine's sudo asks one ([Machinefiles](machinefiles.md)). A removal takes back the packages it names and nothing more: apt removes what depends on a package with it, so `apt-get -s remove` is read first, and a removal it would widen to another package is refused before anything runs (`removing libfoo would remove foo-tool too: nothing removed`, in the app's log); what the package alone pulled in stays, apt's autoremove never runs. A version asked of an apt app is its first package's, `apt-get install <package>=<version>`.

A `github-release` app is fetched by station itself, never by a script and never as root: the release (the latest, or the tag a Machinefile pins, `1.4.4` finding `v1.4.4`) through the GitHub API (`STATION_GITHUB_API_URL`, `GITHUB_TOKEN` when set), its one asset whose name matches the method's pattern (`{os}` and `{arch}` the machine's, `linux` or `darwin`, `arm64` or `amd64`), downloaded whole and hashed. What it installs is what a checksum vouches for: every checksum the release gives the asset is checked, GitHub's digest of it and the line naming it in a checksums file (the one the method names, else the asset's own `.sha256` and any `checksums` or `sha256sum` file of the release), one that differs refuses it (`does not match checksums.txt ... nothing installed`), and none at all refuses it too unless the method says why it is `unverified`. Only then are the commands the app provides taken out of it (a `.tar.gz`, `.tar.bz2`, `.zip`, or the asset itself) and land in `~/.local/bin`, each by a rename over its old file, so an interruption leaves the old command or the new one. Station records each in `~/.local/share/station/releases/<owner>/<name>.json`, by the hash of the file it wrote, before the commands land and once they did: a command there that is not that file is the owner's, never overwritten (`~/.local/bin/fd is not station's: kept, nothing installed`) nor removed, and the inventory holds a release at the tag whose files are the ones on disk. `github-release` is the manager item of that method, ranked after a system's own managers; station is that manager, so the running binary is its own release, held (its version is `station --version`'s) and never installed, updated nor removed by it.

`station-menubar` is station's own menu bar app on a Mac: a `script` that installs the latest release's `Station.app.zip` (a running app keeps its old process until relaunched) to `~/Applications/Station.app`, refused unless the release's `checksums.txt` vouches for it, and read present by that bundle's version ([Release](../guides/release.md#how-a-machine-takes-it)). It is a `script` because what it installs is an app bundle, signed and moved whole, where `github-release` lands commands. Like every `script`, `station apps remove` cannot take it back: `make menubar-uninstall`'s two steps do, the bundle's `--unregister-login-item`, then deleting it.

A `provided-by` app comes with another item's install: `herdr-core`, the `hc` command station drives, comes with `herdr-app` on a Mac. Installing it installs the providing item first, reported as its own row, then looks its commands up. A command its provider brings at its first launch is a next step, never a failure: on a Mac where Herdr was never opened, `◐ herdr-core  open Herdr once, then station apps install herdr-core`, exit 0. On a Linux workspace no method installs `hc` yet: the `herdr-machine` setting of its Mac copies it in ([Machines](machines.md)).

Every method's program writes its words to the app's log, `~/.local/state/station/logs/<app>.log` (each app of a `brew bundle` gets the whole run), appended, and the terminal keeps only station's lines. A `script` installer reads one RETURN on stdin, the consent the install already is; a password it asks is sudo's, on the terminal. Homebrew's runs with `INTERACTIVE=1` where the run has a terminal, so on that stdin it still asks sudo rather than giving up, and with `NONINTERACTIVE=1` where it has none (its installer refuses `INTERACTIVE` beside `CI`); its catalogue line says so before it runs (`asks your password once`). An installer that offers a quiet mode is given it (plannotator's `--non-interactive`), and every `script` step is said as a line of its own before it runs, since a curl installer may still read the terminal.

An App Store app is installed by mas itself, outside the bundle: signed out, the App Store waits on a sign-in no terminal can give and mas says nothing, so a mas that says nothing for 60 s is stopped and the app fails with `the App Store did not answer in 60 s, as when it is signed out: open the App Store and sign in` (exit 1). `brew bundle` installs a whole request in one call and taps a trusted formula itself; the catalog's `trusted` taps are the owner's act of trust ([ADR-002](../decisions/002-brew-bundle-installs-native-commands-uninstall.md)). `station apps update` updates the catalog apps that have an update, and only those; `--manager <manager>` narrows it to one manager's apps, by any key its item installs by (`brew` and `cask` both reach Homebrew's).

## Superseded installs and App Store copies

A catalog app can name the formula it `supersedes`: with the formula alone the app is `? formula installed, app absent`, and once the app itself is OK `station apps install <app>` uninstalls the formula (`● tailscale formula  uninstalled: the app replaces it`). An app that `accepts: [app-store]` and whose bundle carries the App Store's receipt (`Contents/_MASReceipt`) is present through that copy, Not managed, detail `app-store`: `apps install` says it is already installed, `apps update` leaves it to the App Store (`○ the App Store updates it` when named), and `apps remove` refuses it (`Hint: remove it from /Applications`). Tailscale is both ([The tailnet](machines.md#tailscale)).

## Runtimes

mise owns the language runtimes on every machine ([ADR-003](../decisions/003-mise-owns-the-runtimes.md)). `station apps install node` writes mise's global file and installs; a machine that still has node from elsewhere shows it Not managed, "on PATH, not managed by mise", and the fix is that same command. `station shell zsh` activates mise in every shell ([Dotfiles](dotfiles.md)).

## Skills as apps

An agent skill is an app whose method is `skill`, in the category "Agent skills", installed through the `skills` CLI, which `station` requires like it requires Homebrew for a cask. There is no separate family:

```sh
station apps install frontend-design better-ui
station apps frontend-design        # its category is Agent skills
station apps remove shadcn
```

Only community skills are in the catalog. The `@jterrazz` skills are not curated here: `studio doctor` derives them from the repositories the estate declares. Two skills share a name with an app, so they carry a `-skill` suffix: `qmd-skill` and `agent-browser-skill`. A skill is installed for Claude Code (`~/.claude/skills`) and the shared `~/.agents/skills` that codex and the other universal agents read, never for every agent the skills CLI knows, which it does on a home where it detects none. Installed state is read from the `skills` lock file `~/.agents/.skill-lock.json`; currency is compared against GitHub with a 5 second budget: a folder changed upstream is Update available, and an unanswered comparison leaves the skill OK, `currency unknown`.

## The toolbelt

[`skills/studio-toolbelt/`](../../skills/studio-toolbelt/SKILL.md) tells an agent on a studio machine what is available. The sections between its `GENERATED` markers are compiled from the catalog by `make generate` (`src/cmd/generate/`), never edited by hand, and a test run by `make test` fails when the catalog and the skill drift. Change the catalog, run `make generate`, commit both together.

## Related

- [Commands](commands.md): `station apps`.
- [Architecture](../concepts/architecture.md): why an item is a file and the method table is one.
