# Dotfiles

The application configs `station` puts on a machine are embedded in the binary, one folder per application, so a machine converges on the same setup with no clone on disk.

| Section            | Answers                                        |
| ------------------ | ---------------------------------------------- |
| The files          | What is embedded and which setting installs it |
| The shell block    | What `station init` writes into `~/.zshrc`     |
| Removing the block | How to take it out by hand                     |

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

`station init` writes one managed block into `~/.zshrc`; `make provision` and `scripts/install.sh` call it. `make install` never touches the machine.

```zsh
# >>> station >>>
export PATH="$HOME/.local/bin:$PATH"
command -v station >/dev/null 2>&1 && eval "$(station shell zsh)"
# <<< station <<<
```

`station shell zsh` is hidden plumbing ([Commands](commands.md#plumbing)). It prints the embedded loader (its PATH carries `/opt/homebrew/bin` and `sbin` first when they exist and are missing, since only a login shell's `~/.zprofile` runs `brew shellenv` and an interactive non-login zsh would find no Brew-installed mise; then `~/.local/share/bun/bin` and `~/.local/share/go/bin`, where `bun add -g` and `go install` put binaries once [relocated](disk.md#relocation); station appends both to its own PATH, so what it installed there answers a non-interactive call too), the mise activation, station's own completion, and inside a VM the shared Claude token ([Accounts](accounts.md#claude-in-a-vm)). The loader also exports `STATION_TERMINAL`, a fresh id at each shell start, which [`station use`](machines.md#this-terminals-machine) keeps this terminal's machine under; the `starship.toml` prompt's `custom.station` segment runs `station shell prompt` only while that id has a choice, and shows `on linux-1`. Running `init` again leaves a correct block untouched and replaces an older form of the line, never adding a second; every other line of the file is left byte for byte.

## Removing the block

Nothing removes the block for you. Delete the lines from `# >>> station >>>` to `# <<< station <<<` in `~/.zshrc`, `make deprovision` from a checkout removes the binary and leaves the block alone.

## Related

- [Settings](settings.md): the items that install these files.
- [Footprint](footprint.md): where each one lands.
