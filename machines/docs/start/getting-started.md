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
station machinefile drift
```

`--machinefile` binds this Mac to a Machinefile, by its name or its file (a file is copied among yours, `~/.config/station/machinefiles/`), prints its plan, then applies it: what it asks of you first (`homebrew  asks your password once`), then each app installed and each setting set, one at a time. An app already there by another way is adopted, never installed again. Interrupted, it says `station machinefile apply resumes`, and `station machinefile apply` finishes it; `station machinefile drift` says, package by package, how the Mac differs from it since. The format is [Machinefiles](../reference/machinefiles.md).

## 4. Or the engine, the apps, then the settings, one at a time

```sh
station apps install tart tailscale
station engine start
station settings apply ghostty-dotfiles zed-dotfiles hushlogin
station overview
```

An app comes with what it requires: a cask on a Mac without Homebrew installs homebrew first, said before it runs (`▸ Installing homebrew · required by tailscale · asks your password once`), its installer's words kept in `~/.local/state/station/logs/homebrew.log`. The VMs need tart and the engine: a VM command without tart offers to install it, and the first one installs the engine at login ([Machines](../reference/machines.md#the-engine)). The engine reaches its VMs through the Mac's local network, which macOS opens to Machines only once that Mac has approved it: the first `station engine start` asks on the Mac's screen, and until it is approved the start says so, `◐ local network needs you · approve station on macbook-pro's screen, or in System Settings > Privacy & Security > Local Network`. Open the Tailscale app once and log in: Machines only reads it. A setting that needs values (a password, a token) opens a form on a terminal ([Settings](../reference/settings.md)). A Mac stays tidy: dev work lives in VMs ([Machines](../reference/machines.md)), and Claude Code reaches them with a shared token ([Accounts](../reference/accounts.md#the-vault)).

## 5. Remove what it put down

There is no `station uninstall` command yet ([the TODO](../../TODO.md#station-uninstall-undoes-its-own-writes)): `spwn uninstall <ref>` only detaches a dependency from an agent. Removal is these steps, in this order, by hand. Run them on the Mac that carries them.

First the apps and the data you want kept. `station apps remove <app>` takes back what station installed, and `station disk erase <app>` an app's data ([Disk](../reference/disk.md)): do both before the binary goes. VMs and images live in `~/.local/share/station/vms`: `station machines delete` and `station images delete` remove them, and the folder is your data until they have.

```sh
station engine stop --yes --remove
```

That one command takes the engine's one job from this Mac, whichever way it carries it ([ADR-073](../decisions/073-spwn-app-carries-the-engine-on-a-mac-with-a-screen.md)), and the ssh key agent with it, since the engine serves it ([ADR-076](../decisions/076-spwn-on-a-machine-one-active-app-one-background-job-the-engine-updates.md)). Without Spwn.app (mode B) it boots the job out of launchd and deletes `~/Library/LaunchAgents/com.jterrazz.station.engine.plist` (`catalog.EngineLabel`). With Spwn.app (mode A) the job is the bundle's own, `com.jterrazz.spwn.engine` (`catalog.BundledEngineLabel`), and it unregisters it from SMAppService, so Login Items no longer lists it; no plist exists in `~/Library/LaunchAgents` in that mode. A Mac without the app that ran an earlier release may also hold the key agent's own job, which no command of Machines takes: `launchctl bootout gui/$(id -u)/com.jterrazz.station.agent` and `rm ~/Library/LaunchAgents/com.jterrazz.station.agent.plist` remove it. Run it before anything below: in mode A `~/.local/bin/station` is a link into the bundle, and the command goes with the app. An engine kept running for a server keeps its job, and the command says so; stop the server first. Then Spwn.app itself, wherever you put it, which holds nothing else of the engine, and Station.app when the menu bar was installed:

```sh
rm -rf /Applications/Spwn.app ~/Applications/Spwn.app
~/Applications/Station.app/Contents/MacOS/StationMenuBar --unregister-login-item
rm -rf ~/Applications/Station.app
rm -rf ~/Library/Application\ Support/com.jterrazz.spwn ~/Library/Caches/com.jterrazz.spwn ~/Library/WebKit/com.jterrazz.spwn ~/Library/Preferences/com.jterrazz.spwn.plist
```

macOS keeps what it recorded of the app in its Background Task Management database, two records (Spwn and its engine) marked disabled, three on a Mac whose Spwn.app registered the key agent's own job `com.jterrazz.spwn.agent`, and no command of Machines reaches them: `sudo sfltool resetbtm` clears the whole database and is yours to run, after a restart. A Mac that moved from mode B to mode A keeps two more, the bare jobs `com.jterrazz.station.engine` and `com.jterrazz.station.agent`, which `sfltool dumpbtm` lists as legacy agents with no plist behind them: the move deleted their plists, and nothing but `sfltool resetbtm` takes the records, so it is your one-time cleanup there too, never run by Machines (decided 2026-11-22). The window's own folders in the last line are the Tauri shell's, named by its bundle identifier `com.jterrazz.spwn`; one that is absent is fine.

Then the binaries, the shell profile's blocks and the folders:

```sh
rm -f ~/.local/bin/spwn ~/.local/bin/spwn-agents ~/.local/bin/station ~/.local/bin/spwn-machines
rm -rf ~/.local/state/station ~/.config/station ~/.cache/station ~/.spwn
rm -rf ~/.agents/skills/spwn ~/.claude/skills/spwn
```

The second line takes the machine's `spwn` skill, which each apply and update wrote for the agents on it ([Apps and skills](../reference/apps-and-skills.md#the-spwn-skill)); the other skills of those folders are yours, or the `skills` CLI's.

`~/.local/share/station` is left out of that line on purpose: remove it last, once its VMs are deleted.

The installer wrote one block to the profile of the shell that ran it (`~/.zshrc` for zsh, `~/.profile` for bash): the line `# Added by spwn installer`, then `export PATH="$HOME/.local/bin:$PATH"` and the aliases `agents` and `machines`. Delete those four lines. `station init` wrote the block of `~/.zshrc` between `# >>> station >>>` and `# <<< station <<<` (`blockStart` and `blockEnd`, `apps/engine/settings/zshrc.go`), and the apps' packages each a block `# >>> station <id> >>>` to `# <<< station <id> <<<`: delete every such pair, markers included, and nothing between or around them.

The files `init` only edited are yours: delete none of them whole. Their paths are `catalog.AppFilePaths` (`apps/engine/catalog/disk.go`): `~/.npmrc`, `~/.bunfig.toml` (`$XDG_CONFIG_HOME/.bunfig.toml` when it is absolute), Go's env file (`~/Library/Application Support/go/env` on a Mac, `$GOENV` when absolute) and mise's `~/.config/mise/config.toml`. Remove only the keys [Disk's relocation table](../reference/disk.md#relocation) lists for each. In `~/.bunfig.toml` they sit between `# >>> station >>>` and `# <<< station <<<` (`bunfigStart`, `apps/engine/catalog/bunfig.go`), so remove that pair. On a Mac `init` found bare, each of the four holds nothing but station's keys once yours are not there, and goes whole; one with a line of yours stays, minus station's keys. `init` moved the apps' folders (`~/.bun`, `~/go`, `~/.npm`) to where those keys point, so once the keys are gone, an app writes them at the home root again.

Last, `exec zsh`. [Footprint](../reference/footprint.md) lists every path station and the catalogue's apps write, by who removes it.

Every command is in [Commands](../reference/commands.md); every file written is in [Footprint](../reference/footprint.md).
