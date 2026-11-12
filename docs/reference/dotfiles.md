# Dotfiles

The application configs `station` puts on a machine are embedded in the binary, one folder per application, so a machine converges on the same setup with no clone on disk.

| Section            | Answers                                        |
| ------------------ | ---------------------------------------------- |
| The files          | What is embedded and which setting installs it |
| The shell block    | What `station init` writes into `~/.zshrc`     |
| Package blocks     | The block each package may add beside it       |
| Removing the block | How to take them out by hand                   |

## The files

They live under `src/catalog/dotfiles/` and are embedded with `go:embed`.

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

A catalog item whose method declares [`shell`](catalog.md#the-shell-key) puts one identified block into `~/.zshrc` per entry, named by its `id` (`station:postgres/path`), in plain zsh with station's variables filled in when it is written. Postgres uses it for the keg-only `psql`, which Homebrew links nowhere: its method puts the keg's `bin` on the shell's PATH, `[[ -d /opt/homebrew/opt/postgresql@18/bin ]] && path=(/opt/homebrew/opt/postgresql@18/bin $path)`.

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

- **Order.** Station's main block comes first, then the package blocks, each new one right after the last station block, in the plan's order, so what an app requires comes first and a later block's PATH wins. A block already there is rewritten where it stands.
- **Your lines** stay byte for byte, where they are, between and around the blocks.
- **Removal.** A block goes with its package: when `apply --remove` takes the app back, when the Machinefile lists the app no longer (the app stays yours, its block does not), and when a setting is turned off. It goes only while it is still what station would write.
- **A block you edited by hand** reads `changed` in `station drift` and in `station plan` (`edited by hand: station plan --diff shows how`), and apply leaves it as it is (hard case 15 of [Machinefiles](machinefiles.md)); delete it and the next apply writes it again. A block whose item the catalog no longer knows is left alone.

`station plan --diff` shows each block's diff, and each file a setting writes, as apply would change it: the file's lines first (`-`), then what apply writes (`+`). `~/.config/station/zsh/`, where station's block once read the functions of `docker-shortcuts` from, is retired: `station init` removes the file station wrote there and the folder once it is empty, never a file of yours.

## Removing the block

Nothing removes the main block for you. Delete the lines from `# >>> station >>>` to `# <<< station <<<` in `~/.zshrc`, `make deprovision` from a checkout removes the binary and leaves the block alone. A package block goes with its package, or by deleting its own two marker lines and what is between them.

## Related

- [Settings](settings.md): the items that install these files.
- [Footprint](footprint.md): where each one lands.
