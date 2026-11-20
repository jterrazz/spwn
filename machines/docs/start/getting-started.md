# Getting started

From a fresh Mac to a machine `station` knows: the one-line install, naming the machine, then what it carries: its Machinefile applied, or the apps and the settings one at a time.

## 1. Install `station`

```sh
xcode-select --install
curl -fsSL https://spwn.sh/install.sh | bash
exec zsh
```

The script reads the latest release tag, downloads `station-<os>-<arch>`, checks it against the release's `checksums.txt`, installs it to `~/.local/bin/station`, saying two lines of its own (`Installing station for darwin/arm64`, `● station v3.1.0 in ~/.local/bin`), and hands the terminal to `station init`, so init asks this Mac's name. Its own arguments go to init: `curl -fsSL … | sh -s -- --name macbook`, or `curl -fsSL … | sh -s -- --machinefile <name>` to bind this Mac to a Machinefile and apply it at once (step 3). There is no clone: the dotfiles are in the binary. Running the script again is the upgrade.

## 2. Name this machine

```sh
station init --name macbook
```

`init` makes this Mac known to station: it names it, writes the managed block of `~/.zshrc`, and relocates the apps' files: it writes each app's own settings file so the app keeps its files in the XDG folders, then moves what the app already left at the home root (`~/.bun`, `~/go`, `~/.npm`, `~/.tart`) where those settings point. Before it moves anything it lists each move, from where to where and its size, under `# To move`, and asks `Move them?` on a terminal. It installs no app, runs no package manager and asks no password, and it ends on what the Mac carries: `Machinefile  none bound by init`, then the next steps. Where no terminal can be asked (a script, an ssh command without a terminal), init is refused, exit 2, before any step, on a Mac with no name yet (`--name` names it) and when it has something to move (`--yes` answers its question); the install stands. It is safe to run again ([Dotfiles](../reference/dotfiles.md)).

## 3. What it carries: its Machinefile

```sh
station init --machinefile macbook-pro
station drift
```

`--machinefile` binds this Mac to a Machinefile, by its name or its file (a file is copied among yours, `~/.config/station/machinefiles/`), prints its plan, then applies it: what it asks of you first (`homebrew  asks your password once`), then each app installed and each setting set, one at a time. An app already there by another way is adopted, never installed again. Interrupted, it says `station apply resumes`, and `station apply` finishes it; `station drift` says, package by package, how the Mac differs from it since. The format is [Machinefiles](../reference/machinefiles.md).

## 4. Or the engine, the apps, then the settings, one at a time

```sh
station apps install tart tailscale
station engine start
station settings apply ghostty-dotfiles zed-dotfiles hushlogin
station overview
```

An app comes with what it requires: a cask on a Mac without Homebrew installs homebrew first, said before it runs (`▸ Installing homebrew · required by tailscale · asks your password once`), its installer's words kept in `~/.local/state/station/logs/homebrew.log`. The VMs need tart and the engine: a VM command without tart offers to install it, and the first one installs the engine at login ([Machines](../reference/machines.md#the-engine)). The engine reaches its VMs through the Mac's local network, which macOS opens to Machines only once that Mac has approved it: the first `station engine start` asks on the Mac's screen, and until it is approved the start says so, `◐ local network needs you · approve station on macbook-pro's screen, or in System Settings > Privacy & Security > Local Network`. Open the Tailscale app once and log in: Machines only reads it. A setting that needs values (a password, a token) opens a form on a terminal ([Settings](../reference/settings.md)). A Mac stays tidy: dev work lives in VMs ([Machines](../reference/machines.md)), and Claude Code reaches them with a shared token ([Accounts](../reference/accounts.md#the-vault)).

Every command is in [Commands](../reference/commands.md); every file written is in [Footprint](../reference/footprint.md).
