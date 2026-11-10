# Apps and skills

An app is installed content: software or an agent skill (v2 had its own word for it). The catalogue in `src/catalog/` declares each one as a value, and `station apps`, `station overview` and the toolbelt skill all read it.

| Section        | Answers                                                |
| -------------- | ------------------------------------------------------ |
| The catalogue  | How an app is declared, and where it lives             |
| Methods        | How each method installs, removes, updates and is read |
| Runtimes       | Why mise owns node, go, python, bun, rust and java     |
| Skills as apps | Agent skills installed with `station apps install`     |
| The toolbelt   | The committed projection agents read                   |

## The catalogue

An app has a name, a description, a category, its install per kind of machine, the commands it provides and what it requires. Where it lives, the kinds of machine it may be on (`mac`, `linux`, `macos`), is derived from its installs, so it cannot disagree with them; `station apps --json` says it as `lives_on` (`every-machine`, `mac-only`, `vms-only`). Dev apps live in VMs: Xcode, CocoaPods, XcodeGen, Android Studio and Lens are placed `macos` only and install nowhere until the macOS VM exists ([TODO](../../TODO.md)); Ollama stays on Macs for the GPU; Docker lives in the Linux VM.

An app asked for on a machine it does not live on is refused with exit 2, `dia lives on Macs only`, the hint the command with `--on`.

An app's state is one of OK (the method's manager lists it, current), Update available (its manager says a newer version exists), Failed, Not installed, Not managed (found on PATH or in `/Applications` but not installed by its method: `ghostty.app in /Applications, not installed by cask`) or Not known (the manager did not answer). `Requires` orders an install and refuses one whose method's manager is missing (`keynote needs mas, which is not installed`).

## Methods

One registry, `Installers` in `src/apps/contract/`, holds an installer per method, each its own adapter under `src/apps/adapter/<method>/` with its commands and its parser; nothing else switches on a method. `wire` builds the registry; `src/apps/calls` asks each installer what it can do and what is behind (`Outdated`): brew reads `brew outdated --json=v2`, a pinned formula left out; a skill compares its lock with GitHub.

| Method            | Install                                                            | Remove                                  | Update                           | Kinds      |
| ----------------- | ------------------------------------------------------------------ | --------------------------------------- | -------------------------------- | ---------- |
| `brew`, `cask`    | `brew bundle` (Brewfile on stdin)                                  | `brew uninstall`                        | `brew bundle --upgrade`          | Mac, macos |
| `mas`             | `mas install <id>`, silent 60 s: signed out                        | `mas uninstall`                         | `mas update <id>`                | Mac, macos |
| `npm`, `uv`       | `brew bundle`                                                      | `npm uninstall -g`, `uv tool uninstall` | `brew bundle --upgrade`          | Mac, macos |
| `bun`             | `bun install -g`                                                   | `bun remove -g`                         | `bun update -g`                  | all        |
| `mise`            | `mise use --global <tool>@<version>`                               | `mise unuse --global`                   | `mise upgrade`                   | all        |
| `script`, `shell` | the vendor's installer                                             | not reversible                          | rerun the script; none for shell | all        |
| `app`             | manual: the refusal names how                                      | not reversible                          | none                             | Mac, macos |
| `skill`           | `skills add <repo> -g -y --skill <name> --agent claude-code codex` | `skills remove -g -y <name>`            | the install again                | all        |

A `script` install is checked once it ran: its exit code is all the vendor's installer reports, so an install that leaves none of the commands the app provides is Failed, `installed, but brew is not found on the search path`, never OK. The search path is not frozen at the process's start: commands are looked up on disk at each call, in the folders installs land in as well (`/opt/homebrew/bin` and `/usr/local/bin`, read under the system root; `~/.local/bin`; mise's shims), and a child is given a PATH that carries them, so a Homebrew or a script installed by an earlier step of the run is the one the next step finds by name, and a later run finds it too.

An App Store app is installed by mas itself, outside the bundle: signed out, the App Store waits on a sign-in no terminal can give and mas says nothing, so a mas that says nothing for 60 s is stopped and the app fails with `the App Store did not answer in 60 s, as when it is signed out: open the App Store and sign in` (exit 1). `brew bundle` installs a whole request in one call and taps a trusted formula itself; the catalogue's `trusted` taps are the owner's act of trust ([ADR-002](../decisions/002-brew-bundle-installs-native-commands-uninstall.md)). `station apps update` updates the catalogue apps that have an update, and only those; `--method <method>` narrows it to one method's (`brew` covers brew and cask).

## Superseded installs and App Store copies

A catalogue app can name the formula it `Supersedes`: with the formula alone the app is `? formula installed, app absent`, and once the app itself is OK `station setup` uninstalls the formula. An `AppStore` app whose bundle carries the App Store's receipt (`Contents/_MASReceipt`) is present through that copy, Not managed, detail `app-store`: `apps install` says it is already installed, `apps update` leaves it to the App Store (`○ the App Store updates it` when named), and `apps remove` refuses it (`Hint: remove it from /Applications`). Tailscale is both ([The tailnet](machines.md#tailscale)).

## Runtimes

mise owns the language runtimes on every machine ([ADR-003](../decisions/003-mise-owns-the-runtimes.md)). `station apps install node` writes mise's global file and installs; a machine that still has node from elsewhere shows it Not managed, "on PATH, not managed by mise", and the fix is that same command. `station shell zsh` activates mise in every shell ([Dotfiles](dotfiles.md)).

## Skills as apps

An agent skill is an app whose method is `skill`, in the category "Agent skills", installed through the `skills` CLI, which `station` requires like it requires Homebrew for a cask. There is no separate family:

```sh
station apps install frontend-design better-ui
station apps frontend-design        # its category is Agent skills
station apps remove shadcn
```

Only community skills are in the catalogue. The `@jterrazz` skills are not curated here: `studio doctor` derives them from the repositories the estate declares. Two skills share a name with an app, so they carry a `-skill` suffix: `qmd-skill` and `agent-browser-skill`. A skill is installed for Claude Code (`~/.claude/skills`) and the shared `~/.agents/skills` that codex and the other universal agents read, never for every agent the skills CLI knows, which it does on a home where it detects none. Installed state is read from the `skills` lock file `~/.agents/.skill-lock.json`; currency is compared against GitHub with a 5 second budget: a folder changed upstream is Update available, and an unanswered comparison leaves the skill OK, `currency unknown`.

## The toolbelt

[`skills/studio-toolbelt/`](../../skills/studio-toolbelt/SKILL.md) tells an agent on a studio machine what is available. The sections between its `GENERATED` markers are compiled from the catalogue by `make generate` (`src/cmd/generate/`), never edited by hand, and a test run by `make test` fails when the catalogue and the skill drift. Change the catalogue, run `make generate`, commit both together.

## Related

- [Commands](commands.md): `station apps`.
- [Architecture](../concepts/architecture.md): why the catalogue is values and the method table is one.
