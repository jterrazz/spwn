# Getting started

From a fresh Mac to a machine `station` knows: the one-line install, naming the machine, then the engine, the apps and the settings, one at a time.

## 1. Install `station`

```sh
xcode-select --install
curl -fsSL https://raw.githubusercontent.com/jterrazz/studio-station/main/scripts/install.sh | sh
exec zsh
```

The script reads the latest release tag, downloads `station-<os>-<arch>`, checks it against the release's `checksums.txt`, installs it to `~/.local/bin/station`, saying two lines of its own (`Installing station for darwin/arm64`, `● station v3.1.0 in ~/.local/bin`), and hands the terminal to `station init`, so init asks this Mac's name. Its own arguments go to init: `curl -fsSL … | sh -s -- --name macbook`. There is no clone: the dotfiles are in the binary. Running the script again is the upgrade.

## 2. Name this machine

```sh
station init --name macbook
```

`init` makes this Mac known to station: it names it, writes the managed block of `~/.zshrc`, moves an old `~/.studio/` to the XDG folders and brings station's stores to their version. It installs no app, runs no package manager and asks no password, and it ends on what the Mac carries: `Machinefile  none yet`, then the next steps. Where no terminal can be asked (a script, an ssh command without a terminal), init on a Mac with no name yet is refused, exit 2, before any step: `--name` is how such a run names it; the install stands. It is safe to run again ([Dotfiles](../reference/dotfiles.md)).

## 3. The engine, the apps, then the settings

```sh
station apps install tart tailscale
station engine start
station settings apply ghostty-dotfiles zed-dotfiles hushlogin
station overview
```

An app comes with what it requires: a cask on a Mac without Homebrew installs homebrew first, said before it runs (`▸ Installing homebrew · required by tailscale · asks your password once`), its installer's words kept in `~/.local/state/station/logs/homebrew.log`. The VMs need tart and the engine: a VM command without tart offers to install it, and the first one installs the engine at login ([Machines](../reference/machines.md#the-engine)). Open the Tailscale app once and log in: station only reads it. A setting that needs values (a password, a token) opens a form on a terminal ([Settings](../reference/settings.md)). A Mac stays tidy: dev work lives in VMs ([Machines](../reference/machines.md)), and Claude Code reaches them with a shared token ([Accounts](../reference/accounts.md#claude-in-a-vm)).

Every command is in [Commands](../reference/commands.md); every file written is in [Footprint](../reference/footprint.md).
