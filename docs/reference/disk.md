# Disk

What an app keeps on a machine, by what an erase would lose, and the one family that sizes, frees, erases and moves it. Every path of every row is listed, generated from the table, in [Footprint](footprint.md).

| Section        | Answers                                                  |
| -------------- | -------------------------------------------------------- |
| The groups     | What each Disk group means and who removes it            |
| The folders    | Where station's own files live                           |
| Relocation     | How an app's home moves, and what never moves            |
| Tart           | Where the engine and its VMs live                        |
| `station disk` | The read, `free`, `erase` and `move`                     |
| Misplaced      | The home-root folders an app left, and how they are seen |
| Backups        | What Time Machine skips                                  |

## The groups

Every path an app or station writes has one catalogue level, declared in `apps/engine/catalog/disk.go`; the Disk group is the level in the owner's words (`disk.GroupOf`).

| Group           | Levels              | Means                                                                 | Who removes it                                 |
| --------------- | ------------------- | --------------------------------------------------------------------- | ---------------------------------------------- |
| `misplaced`     | a misplaced folder  | a folder at the home root its app's own settings move elsewhere       | `station disk move <folder>`, with no question |
| `can-rebuild`   | `cache`             | rebuilt or re-downloaded on demand: only time and bandwidth are lost  | `station disk free`, with no question          |
| `your-data`     | `data`              | state the owner may wipe to factory: histories, databases, simulators | `station disk erase <app>`, its name typed     |
| `never-touched` | `config` and `kept` | keys, tokens, settings, the VM records                                | nobody: a setting's `off`, where reversible    |

The files an install put there (an app bundle, a Cellar, mise's installs, bun's and uv's globals, browser builds) are no group: `station apps remove` owns them. station's own data are its VMs and images, which `station machines delete` and `station images delete` own: they are no Disk row.

## The folders

station's own files follow the XDG rule: `~/.config/station` (settings), `~/.local/share/station` (Tart and its VMs), `~/.local/state/station` (records, logs, keys), `~/.cache/station` (downloads), `~/.local/bin` (the binary and the tart launcher). Each root honours its `XDG_*_HOME` when it is set and absolute. `~/.cache` stays apart from `~/.local`: everything under it may be deleted.

## Relocation

An app's path moves only through the app's own settings file, the one it reads whoever launches it (an IDE started from the Dock included), never through a shell variable alone. `station init` writes them, absolute, and never over a value the owner set by hand:

| App | File                                                                                                                   | Keys                                             | Old home |
| --- | ---------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ | -------- |
| go  | Go's env file: `$GOENV`, else `~/Library/Application Support/go/env` (Linux: `~/.config/go/env`)                       | `GOPATH`, `GOMODCACHE`                           | `~/go`   |
| npm | `~/.npmrc`                                                                                                             | `cache`                                          | `~/.npm` |
| bun | `~/.bunfig.toml`                                                                                                       | `globalDir`, `globalBinDir`, `install.cache.dir` | `~/.bun` |
| go  | mise's global file: `$MISE_GLOBAL_CONFIG_FILE`, else `$MISE_CONFIG_DIR/config.toml`, else `~/.config/mise/config.toml` | `settings.go_set_gobin = false`                  |          |

mise's go backend sets `GOBIN` to the running version's own install unless `go_set_gobin` is false, so `go install` would put its binaries where an upgrade or a prune loses them and `disk erase go` never sees them; with it off, they land in `~/.local/share/go/bin`. The key goes under the owner's own `[settings]` table when there is one.

bun's values sit in a managed block between `# >>> station >>>` and `# <<< station <<<`; a bunfig with its own `[install]` table is left alone, and init says which keys to set by hand. The macOS machine tier proved that `bun add -g`, `bun install -g`, `bunx` and `bun pm cache rm` honour the global bunfig and create no `~/.bun`. `station init` moves an existing `~/.bun`'s globals and bins into `~/.local/share/bun`, one rename each, so their relative bin links still resolve; `station disk free bun` deletes its old cache.

Nothing already under `~/Library/Caches` or another `~/Library` folder moves: go-build, Homebrew, mise's cache, the pnpm store and Playwright are where macOS keeps them. The folders an app keeps at the home root on purpose (`.ssh`, `.gnupg`, `.cargo` and `.rustup`, which rust-analyzer started by a GUI IDE needs, `.claude`, `.codex`, the dotfiles) are the declared exceptions, listed with their reasons in [Footprint](footprint.md#home-root-exceptions).

## Tart

Tart's app and its `LICENSE` live in `~/.local/share/station/tart/`, its home (`TART_HOME`) is `~/.local/share/station/vms`: the VMs and images, the OCI and IPSW caches and `tmp`, so a clone from the cache stays an APFS clone. The launcher `~/.local/bin/tart`, written by the setting `tart-launcher`, sets `TART_HOME` itself: a launchd agent, `--on` over ssh and a GUI all get it, with no shell variable. `station disk move ~/.tart` (and `station init`) moves an existing `~/.tart` and `~/.local/opt/tart.app` where they stand, one rename each, never a copy and never a pull; it refuses while `tart` runs, onto a destination that holds anything, and across volumes.

## `station disk`

- `station disk` tells the volume (`388 GB free of 994 GB`, `DiskUsage.Volume`), then Misplaced, Can rebuild, Your data and Never touched, each row an app and its bytes, the biggest first. `--json` is `station.disk.read/v1`: `free_bytes`, `total_bytes`, `findings` (each misplaced folder) and the rows, each with its `app`, `group`, `bytes`, `paths` and, misplaced, its `move_to`. A row that holds nothing is not shown. Every row is sized in parallel under 10 s; on a terminal a row shows its last size Muted until its walk answers, then fills, and a row past the budget shows `?` (`-1` in JSON). Docker's size is `docker system df` under 5 s. `--on <machine>` reads that Mac or workspace there; a server keeps nothing of station's and is refused.
- `station disk free [<app...>]` empties what an app can rebuild with its own command (`go clean -cache -modcache`, `brew cleanup --prune=all`, `tart prune --entries=caches --older-than=0`, `npm cache clean --force`...), or deletes a cache folder its docs call deletable; what the command leaves behind (Homebrew's `api` and `bootsnap`, go's `README`) is deleted after it, and a command that would create the cache it cleans (`pnpm store prune` on no store) is not run. Bare, it frees every Can rebuild row. It asks nothing; a failing app is its row's `✗` with its reason, the others still run, and the run ends `Error: 1 of 18 could not be freed`.
- An app that is not installed has no command to run: `free` and `erase` skip its commands and still delete by path. With nothing left, `free` says `○ <app> not installed`.
- `station disk erase <app>` erases an app's caches and data, back to factory: always one app, its name typed on a terminal, `--confirm=<app>` in a script; `--yes` answers nothing. It refuses before asking while the app runs (`Xcode is running: its data cannot be erased now`), and a Never touched row whatever it is asked (`~/.ssh is never touched: station erases no key, token or setting`). Erasing xcode empties DerivedData and erases every simulator device; the archives stay. A row with no erase names who owns its data instead: station's VMs and images are `station machines delete` and `station images delete`, Homebrew's formulae are `station apps remove`.
- `station disk move <folder>` moves a misplaced folder where its app expects it, one rename each, then applies the settings that make the app look there (`tart-launcher` for `~/.tart`); it asks nothing, refuses while the app runs, onto a destination that holds anything and across volumes, and a folder that is not misplaced (`~/Developer is not misplaced`).

The v2 words stay one release as hidden aliases: `station storage` is `disk`, `storage clean` is `free`, `storage reset` is `erase`, each printing `→ now: station disk …` on stderr.

## Misplaced

A misplaced folder is a home-root folder an app writes when its relocation is missing or was never applied: `~/go`, `~/.npm`, `~/.tart`. `station disk` lists it first, and `station overview` shows it as a `◐` row with its size and its fix, `station disk move <folder>`: what is still to move moves, the cache inside it is emptied (the app's own command, or by path with the app gone) and the emptied folders go. A file station does not own stays where it is.

## Backups

The setting `backup-exclusions` keeps `~/.cache` and `~/.local/share/station/vms` out of Time Machine with `tmutil addexclusion` (a sticky exclusion, no root); uninstalling it gives them back.

## Related

- [Footprint](footprint.md): every path, generated from the table.
- [Settings](settings.md): the settings; [Preferences](preferences.md): the environment variables.
