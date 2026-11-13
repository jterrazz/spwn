# Dotfiles

The application configs `station` puts on a machine are embedded in the binary, one folder per application, so a machine converges on the same setup with no clone on disk.

| Section            | Answers                                        |
| ------------------ | ---------------------------------------------- |
| The files          | What is embedded and which setting installs it |
| The shell block    | What `station init` writes into `~/.zshrc`     |
| Package blocks     | The block each package may add beside it       |
| Removing the block | How to take them out by hand                   |

## The files

They live under `apps/engine/catalog/dotfiles/` and are embedded with `go:embed`.

| Application | Contents                                                                              | Installed by        |
| ----------- | ------------------------------------------------------------------------------------- | ------------------- |
| `ghostty`   | terminal `config` and `themes/`                                                       | `ghostty-dotfiles`  |
| `starship`  | the `starship.toml` prompt                                                            | `starship-dotfiles` |
| `tmux`      | `tmux.conf`                                                                           | `tmux-dotfiles`     |
| `zed`       | `settings.json` and `.oxfmtrc.json`: Zed settings and the formatter values it applies | `zed-dotfiles`      |
| `zsh`       | `zshrc.sh`, the shell loader: prompt, colours, FZF, `jj` `jc` `jo` `jg`               | `station shell zsh` |

The `jj`, `jc`, `jo` and `jg` shortcuts attach tmux session `main` and open Claude, Codex and Gemini in tmux.

## The shell block

`station init` writes station's main block into `~/.zshrc`; `make provision` and `scripts/install.sh` call it. `make install` never touches the machine.

```zsh
# >>> station >>>
export PATH="$HOME/.local/bin:$PATH"
command -v station >/dev/null 2>&1 && eval "$(station shell zsh)"
# <<< station <<<
```

`station shell zsh` is hidden plumbing ([Commands](commands.md#plumbing)). It prints the embedded loader (its PATH carries `/opt/homebrew/bin` and `sbin` first when they exist and are missing, since only a login shell's `~/.zprofile` runs `brew shellenv` and an interactive non-login zsh would find no Brew-installed mise; then `~/.local/share/bun/bin` and `~/.local/share/go/bin`, where `bun add -g` and `go install` put binaries once [relocated](disk.md#relocation); station appends both to its own PATH, so what it installed there answers a non-interactive call too), the mise activation, station's own completion, and inside a VM the shared Claude token ([Accounts](accounts.md#claude-in-a-vm)). The loader also exports `STATION_TERMINAL`, a fresh id at each shell start, which [`station use`](machines.md#this-terminals-machine) keeps this terminal's machine under; the `starship.toml` prompt's `custom.station` segment runs `station shell prompt` only while that id has a choice, and shows `on linux-1`. Running `init` again leaves a correct block untouched and replaces an older form of the line, never adding a second; every other line of the file is left byte for byte.

## Package blocks

A catalog package whose method declares [`shell`](catalog.md#the-shell-key) puts one identified block into `~/.zshrc` per entry, named by its `id` (`station:postgres/path`), in plain zsh with station's variables filled in when it is written. Postgres uses it for the keg-only `psql`, which Homebrew links nowhere: its method puts the keg's `bin` on the shell's PATH, `[[ -d /opt/homebrew/opt/postgresql@18/bin ]] && path=(/opt/homebrew/opt/postgresql@18/bin $path)`.

```zsh
# >>> station >>>
export PATH="$HOME/.local/bin:$PATH"
command -v station >/dev/null 2>&1 && eval "$(station shell zsh)"
# <<< station <<<
# >>> station:postgres/path >>>
[[ -d /opt/homebrew/opt/postgresql@18/bin ]] && path=(/opt/homebrew/opt/postgresql@18/bin $path)
# <<< station:postgres/path <<<
```

An app's blocks are written by `station apply` once the app is there, whichever way it got there; a setting's, by `station settings apply`. Each block is one upsert, and one removal, by its id:

- **Order.** Station's main block comes first, then the package blocks, each new one right after the last station block, in the plan's order, so what an app requires comes first and a later block's PATH wins. With no station block yet, a new one goes at the end, before a last line of yours that has no newline, so a block's end marker always ends its line and an installer's `echo ... >> ~/.zshrc` lands on a line of its own. A block already there is rewritten where it stands.
- **Your lines** stay byte for byte, where they are, between and around the blocks.
- **One write at a time.** Every station process that changes `~/.zshrc` (`init`, `apply`, `settings apply`, the menu bar's) takes the lock `~/.local/state/station/zshrc.lock` first (Ctrl-C ends the wait for it: `interrupted`), and the file lands whole: the new text is written beside the file, through a link to the file it names, synced to the disk, given the file's own mode, owner and group, then renamed over it, and the folder synced. A crash or a power cut leaves the old file or the new one, never half of one, and two writers at once never drop each other's block. Where station may not write in the file's folder, or cannot give a new file its owner, it writes the file where it stands, still under the lock.
- **The record.** Each block apply writes is recorded in the ledger, `~/.local/state/station/ledger.json`: its id, its package, and the digest of the text written. A block is judged against that record, never against what station would write now alone. A block that holds exactly what apply would write, with no record of that text (the ledger deleted, say), gets its record from the next apply.
- **Station's earlier text** (a block still exactly as apply last wrote it, which station writes otherwise now: a new template, an option's value moved) reads `outdated`, and the next apply writes the new text where the block stands.
- **A block you edited by hand** (other text than apply last wrote, a start with no end, an id twice) reads `changed` in `station drift` and in `station plan` (`edited by hand: station plan --diff shows how`), and apply leaves it as it is (hard case 15 of [Machinefiles](machinefiles.md)); delete it and the next apply writes it again.
- **Removal.** A block goes with its package. A block apply wrote for an app this Machinefile installed and lists no longer is extra: a bare apply keeps it, as it keeps the app, and `apply --remove` takes it back only with the app's own removal, once the app is taken back or already gone, and only while it is still what apply last wrote. An app that stays keeps its block: one adopted or yours, one another Machinefile installed, one skipped on this system, one whose removal is judged again next run (required, held, unread, failed); one let go of as yours (station cannot uninstall it, or it is there another way since) leaves its block to you too, and so does a block you edited since station wrote it (`edited by hand since station wrote it: kept, and yours`), which `apply --remove` lets go of while it keeps it in the file; one you deleted already reads `deleted by hand: already gone`, and its record goes. A block the ledger does not hold is never station's to take. A setting's block goes when the setting is turned off.

`station plan --diff` shows each block's diff, and each file a setting writes, as apply would change it: the file's lines first (`-`), then what apply writes (`+`).

`~/.config/station/zsh/`, where station's block once read the functions of `docker-shortcuts` from, is retired. `station init` moves `docker-shortcuts.zsh` into `~/.zshrc` as the block `station:docker-shortcuts`, its bytes as they are, your edits included, so every shell keeps the functions and the setting reads as it did (applied, or changed since); a file whose block `~/.zshrc` holds already with exactly its bytes, which an earlier move could not delete, goes now, and one whose block holds other text stays where it is. The folder goes once it is empty, never a file of yours.

## Removing the block

Nothing removes the main block for you. Delete the lines from `# >>> station >>>` to `# <<< station <<<` in `~/.zshrc`, `make deprovision` from a checkout removes the binary and leaves the block alone. A package block goes with its package, or by deleting its own two marker lines and what is between them.

## Related

- [Settings](settings.md): the packages that install these files.
- [Footprint](footprint.md): where each one lands.
