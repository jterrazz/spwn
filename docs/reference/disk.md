# Disk

What an app keeps on a machine, by what an erase would lose, and the one family that sizes, frees, erases and moves it. Every path of every row is listed, generated from the table, in [Footprint](footprint.md).

| Section        | Answers                                                                      |
| -------------- | ---------------------------------------------------------------------------- |
| The groups     | What each Disk group means and who removes it                                |
| By kind        | A machine's disk by kind, what station holds on a Mac, and what nothing uses |
| The folders    | Where station's own files live                                               |
| Relocation     | How an app's home moves, and what never moves                                |
| Tart           | Where the engine and its VMs live                                            |
| `station disk` | The read, `free`, `erase` and `move`                                         |
| Misplaced      | The home-root folders an app left, and how they are seen                     |
| Backups        | What Time Machine skips                                                      |

## The groups

Every path an app or station writes has one catalogue level, declared in `apps/engine/catalog/disk.go`; the Disk group is the level in the owner's words (`disk.GroupOf`).

| Group           | Levels              | Means                                                                 | Who removes it                                 |
| --------------- | ------------------- | --------------------------------------------------------------------- | ---------------------------------------------- |
| `misplaced`     | a misplaced folder  | a folder at the home root its app's own settings move elsewhere       | `station disk move <folder>`, with no question |
| `can-rebuild`   | `cache`             | rebuilt or re-downloaded on demand: only time and bandwidth are lost  | `station disk free`, with no question          |
| `your-data`     | `data`              | state the owner may wipe to factory: histories, databases, simulators | `station disk erase <app>`, its name typed     |
| `never-touched` | `config` and `kept` | keys, tokens, settings, the VM records                                | nobody: a setting's `off`, where reversible    |

The files an install put there (an app bundle, a Cellar, mise's installs, bun's and uv's globals, browser builds) are no group: `station apps remove` owns them. station's own data are its VMs and images, which `station machines delete` and `station images delete` own: they are no Disk row.

## By kind

`disk.read` tells a machine's disk by kind as well as by row (`Kinds`, in this order), what its volume uses (`Used`, `Total` less `Free`), and what its caches can free (`CanFree`: "Caches hold 41.7 GB. Each app rebuilds its own when it needs it.", or "No cache to free. What is left is yours or the system's."). A `CanFree` names the call that frees it in `Call` (`disk.free` here), and no call when there is nothing to free, so a window offers the gesture by `Call` and never by comparing bytes. `CanFree` is absent while `caches` is, so no sentence understates a cache still walked. Each kind is read from what the survey already walks, and a kind whose measure has not answered is absent, never 0: one the machine cannot measure cheaply, one whose walk still pends (`Pending` holds) or did not answer within its budget, and `machines` while the hypervisor has not answered:

| Kind        | Measured                                                                                                                                                                                                         |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `system`    | macOS: one `statfs` of the data volume, its free blocks less the container's available ones, what the APFS container's other volumes hold. Linux: `/usr`, walked with the rows, less the apps installed under it |
| `apps`      | the files the catalog's apps installed (each row's `installed` paths)                                                                                                                                            |
| `your-data` | what the volume uses that no other kind holds, never below 0; absent whenever a kind it subtracts is unknown, since it would then hold that kind too                                                             |
| `caches`    | the Can rebuild rows, what `station disk free` frees, station's logs among them                                                                                                                                  |
| `machines`  | on a Mac whose hypervisor answered: its machines' and images' disks, and the leftovers                                                                                                                           |
| `free`      | the volume's free bytes                                                                                                                                                                                          |

The owner's home and `/Applications` are not walked whole: what no row and no kind names falls in `your-data`. On a Mac, `Station` adds what station holds outside any machine: `Holds` (`machine-disks`, each VM record's disk as the hypervisor lists it; `images`, each image version's disk; `leftovers`, below; `logs`, `~/.local/state/station/logs`; `this-mac`, what the volume uses beyond them; `free`), its sentence ("Outside any machine, station holds 107 GB: machine disks, images, leftovers and its logs"), the Mac's own (`Itself`: "This Mac itself holds 490 GB."), each machine's size with the Mac first, what nothing uses (`CanFree`: its sentence, "12.0 GB can be freed: 1 unused image and 1 leftover disk.", its `Frees` each a `Holder`, a `Kind` (`image`, `disk` or `folder`), a `Name` and its `Bytes`, its `Call` `disk.free-unused`), the old folders it keeps (`Kept`, below) and its verdict, which sums exactly what that `CanFree` frees: "12.0 GB can be freed." (warn), else "Nothing to free." (ok), with the line "on macbook-pro: no machine and no file of yours is touched". A VM's caches are its own `disk.read`'s `CanFree`.

### What nothing uses

Station's `CanFree` lists what it frees, one by one, under two holders:

- `images`: each image version nothing uses, by `<image>@<version>`: not the newest of its name, no machine made from it, no build making it, the rule `images.list` says as `Deletable` ([Images](images.md#listing)).
- `leftovers`: each disk the hypervisor lists, stopped, that station's hypervisor adapter cloned and has not deleted, that no machine record names and that is no image's or build's (`station-image-*`, `station-build-*`): the disk of a machine whose delete did not finish. The adapter keeps a ledger of what it cloned, one file per VM under `~/.local/state/station/cloned/` holding the identity of the VM's folder the clone made (`~/.local/share/station/vms/vms/<vm>`: its device, inode and birth time), written when a clone is made, moved by its own rename, which drops an entry a delete left under the new name, and removed by its delete, by a machine delete that finds the disk already gone, and by an image's discard. A name stands in it only while its folder has that identity, and an entry whose folder is gone or another is dropped when read: a disk it does not name (a VM made or renamed by hand in station's folder, even under a name station once cloned) is never a leftover. A disk that runs, or whose state the hypervisor says in a word station does not know, is in use, and a test VM of the machine tiers (`station-e2e-*`, `station-play`) is never a leftover: all stay in `this-mac`. Also the hypervisor's old folder: a misplaced folder of its package that moves whole (`~/.tart`), once the folder it moves to holds anything, since its move is then refused and nothing reads it again, and only while it holds no VM (nothing under `~/.tart/vms`, the catalog's `yours`): a cache and `tmp` alone. While the folder it moves to is empty, it is the misplaced folder `station disk move` carries.

An old folder that holds VMs is the owner's and never freed: `Station.Kept` lists it with the VMs it holds and its sentence (`"~/.tart holds VMs of yours (linux-dev), so station never frees it: move each into ~/.local/share/station/vms/vms first."`), and its bytes stay in `this-mac`. While a machine record cannot be read, nothing is unused, since its disk may be any of them: `CanFree` says which file (`"~/.local/state/station/vms/api.json cannot be read: nothing is offered to free until it can."`) with no `Call`, and the verdict is "A machine record cannot be read." (warn).

`disk.free-unused` frees what it is asked by `Names`, on a Mac only: at least one (`disk.names-needed` with none), each once, after the names typed, joined by a comma and a space (`confirm.needed` unanswered, `confirm.mismatch` typed otherwise), so the answer binds the set it was asked about. No name holds a comma or a space, so the joined subject reads one way: a machine (a disk station cloned) and an image are named in `[a-z0-9-]`, an image version in digits and dots, and the old folder is a fixed path. It is `at-target`: on another Mac it frees that Mac's own, and its question travels back to the Mac that asked, which shows the names and sends the answer. It reads what nothing uses again when it runs: a name that is not among it (a machine now made from that image, a record that names that disk, a disk that runs, one station did not make, an old folder that holds VMs, the newest image, a name gone) is refused `disk.not-freeable` before anything goes, and a record that cannot be read refuses the whole free, `disk.record-unreadable`. Just before each one goes it is judged again, once every machine record still reads (one that cannot be read leaves that one and every one after it `broken`, `disk.record-unreadable`'s reason): an image version goes through the images' own delete, whose in-use rule refuses it there; a leftover disk only while the records, the hypervisor's listing and its ledger still say it is one, then through the hypervisor's delete, which refuses a running one; the old folder only while its process does not run, and only once set aside: it is renamed beside itself (`~/.tart.station-freeing`) in one rename, its `yours` read again there, and a VM found there brings it back whole, never erased. One that fails is its outcome's `broken` with its reason, while the others still go. Nothing else is touched: no machine, no file of yours.

station's logs are a cache of its row, so they count in `caches` and, on a Mac, in what station holds outside any machine: `this-mac` is what the volume uses beyond station's parts, so the part's bar never counts them twice, and `Station` waits for their walk as it waits for the hypervisor. A VM record whose disk the hypervisor does not list has no machine disk, and a disk the hypervisor lists that no VM record or image version names is a leftover when it is one (above); any other (a test VM, a VM made or renamed by hand, a running one, an image or a build disk whose record is gone, an image whose template left every Machinefile source) is no part of station's: its bytes fall in `this-mac`.

Each `disk.read` asks the hypervisor itself, one listing of its disks within a command's 2 s, and never takes it from the survey a run shares: a machine or an image made or deleted since the last read is seen. `Station` is `null` until the hypervisor answers: while it is asked (`Pending` holds), when it fails or answers late, when a Machinefile source cannot be read, and on a Mac with no hypervisor, never zeros. While it is asked, fails or is late, `machines` is absent and `your-data` with it; on a Mac with no hypervisor `machines` is absent and `your-data` is the remainder of the other kinds.

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

## Misplaced

A misplaced folder is a home-root folder an app writes when its relocation is missing or was never applied: `~/go`, `~/.npm`, `~/.tart`. `station disk` lists it first, and `station overview` shows it as a `◐` row with its size and its fix, `station disk move <folder>`: what is still to move moves, the cache inside it is emptied (the app's own command, or by path with the app gone) and the emptied folders go. A file station does not own stays where it is.

## Backups

The setting `backup-exclusions` keeps `~/.cache` and `~/.local/share/station/vms` out of Time Machine with `tmutil addexclusion` (a sticky exclusion, no root); uninstalling it gives them back.

## Related

- [Footprint](footprint.md): every path, generated from the table.
- [Settings](settings.md): the settings; [Preferences](preferences.md): the environment variables.
