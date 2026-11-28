# Dotfiles

The application configs `station` puts on a machine are embedded in the binary, one folder per application, so a machine converges on the same setup with no clone on disk.

| Section            | Answers                                        |
| ------------------ | ---------------------------------------------- |
| The files          | What is embedded and which setting installs it |
| The shell block    | What `station init` writes into `~/.zshrc`     |
| Package blocks     | The block each package may add beside it       |
| Fragments          | A block or JSON values in a file you keep      |
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

`station init` writes Machines' main block into `~/.zshrc`; `make provision` and the public installer (`install.sh` at spwn.sh) call it. `make install` never touches the machine.

```zsh
# >>> station >>>
export PATH="$HOME/.local/bin:$PATH"
command -v station >/dev/null 2>&1 && eval "$(station shell zsh)"
# <<< station <<<
```

`station shell zsh` is hidden plumbing ([Commands](commands.md#plumbing)). It prints the embedded loader (its PATH carries `/opt/homebrew/bin` and `sbin` first when they exist and are missing, since only a login shell's `~/.zprofile` runs `brew shellenv` and an interactive non-login zsh would find no Brew-installed mise; then `~/.local/share/bun/bin` and `~/.local/share/go/bin`, where `bun add -g` and `go install` put binaries once [relocated](disk.md#relocation); Machines appends both to its own PATH, so what it installed there answers a non-interactive call too), the mise activation and Machines' own completion.. The loader also exports `STATION_TERMINAL`, a fresh id at each shell start, which [`station use`](machines.md#this-terminals-machine) keeps this terminal's machine under; the `starship.toml` prompt's `custom.station` segment runs `station shell prompt` only while that id has a choice, and shows `on linux-1`. Running `init` again leaves a correct block untouched and replaces an older form of the line, never adding a second; every other line of the file is left byte for byte.

## Package blocks

A catalog package whose method declares [`shell`](catalog.md#the-shell-key) puts one identified block into `~/.zshrc` per entry, named by its `id` (`station:postgres/path`), in plain zsh with Machines' variables filled in when it is written. Postgres uses it for the keg-only `psql`, which Homebrew links nowhere: its method puts the keg's `bin` on the shell's PATH, `[[ -d /opt/homebrew/opt/postgresql@18/bin ]] && path=(/opt/homebrew/opt/postgresql@18/bin $path)`.

```zsh
# >>> station >>>
export PATH="$HOME/.local/bin:$PATH"
command -v station >/dev/null 2>&1 && eval "$(station shell zsh)"
# <<< station <<<
# >>> station:postgres/path >>>
[[ -d /opt/homebrew/opt/postgresql@18/bin ]] && path=(/opt/homebrew/opt/postgresql@18/bin $path)
# <<< station:postgres/path <<<
```

An app's blocks are written by `station machinefile apply` once the app is there, whichever way it got there; a setting's, by `station settings apply`. Each block is one upsert, and one removal, by its id:

- **Order.** Machines' main block comes first, then the package blocks, each new one right after the last station block, in the plan's order, so what an app requires comes first and a later block's PATH wins. With no station block yet, a new one goes at the end, before a last line of yours that has no newline, so a block's end marker always ends its line and an installer's `echo ... >> ~/.zshrc` lands on a line of its own. A block already there is rewritten where it stands.
- **Your lines** stay byte for byte, where they are, between and around the blocks.
- **One write at a time.** Every station process that changes `~/.zshrc` (`init`, `apply`, `settings apply`, the menu bar's) takes the lock `~/.local/state/station/zshrc.lock` first, and deletes it when its write ends (Ctrl-C ends the wait for it: `interrupted`), and the file lands whole: the new text is written beside the file, through a link to the file it names, synced to the disk, given the file's own mode, owner and group, then renamed over it, and the folder synced. A crash or a power cut leaves the old file or the new one, never half of one, and two writers at once never drop each other's block. Where Machines may not write in the file's folder, or cannot give a new file its owner, it writes the file where it stands, still under the lock.
- **The record.** Each block apply writes is recorded in the ledger, `~/.local/state/station/ledger.json`: its id, its package, and the digest of the text written. A block is judged against that record, never against what Machines would write now alone. A block that holds exactly what apply would write, with no record of that text (the ledger deleted, say), gets its record from the next apply.
- **Machines' earlier text** (a block still exactly as apply last wrote it, which Machines writes otherwise now: a new template, an option's value moved) reads `outdated`, and the next apply writes the new text where the block stands.
- **A block you edited by hand** (other text than apply last wrote, a start with no end, an id twice) reads `changed` in `station machinefile drift` and in `station machinefile plan` (`edited by hand: station machinefile plan --diff shows how`), and apply leaves it as it is (hard case 15 of [Machinefiles](machinefiles.md)); delete it and the next apply writes it again.
- **Removal.** A block goes with its package. A block apply wrote for an app this Machinefile installed and lists no longer is extra: a bare apply keeps it, as it keeps the app, and `apply --remove` takes it back only with the app's own removal, once the app is taken back or already gone, and only while it is still what apply last wrote. An app that stays keeps its block: one adopted or yours, one another Machinefile installed, one skipped on this system, one whose removal is judged again next run (required, held, unread, failed); one let go of as yours (Machines cannot uninstall it, or it is there another way since) leaves its block to you too, and so does a block you edited since Machines wrote it (`edited by hand since station wrote it: kept, and yours`), which `apply --remove` lets go of while it keeps it in the file; one you deleted already reads `deleted by hand: already gone`, and its record goes. A block the ledger does not hold is never Machines' to take. A setting's block goes when the setting is turned off.

`station machinefile plan --diff` shows each block's diff, and each file a setting writes, as apply would change it: the file's lines first (`-`), then what apply writes (`+`).

## Fragments

A setting's [`file`](catalog.md#methods) method writes a whole file by default; with `mode: block` or `mode: merge` it writes only a fragment into a file that stays yours, and `station settings off` takes back only that fragment. The package is refused at decode when it names `permissions` (the file's mode stays yours), one file twice, a merge from or into a file that is not `.json`, or a block into a `.json` file.

### What is written

- **A block** (`mode: block`) is the text of the embedded file between the two marker lines `# >>> station:<setting> >>>` and `# <<< station:<setting> <<<`, placed as a [package block](#package-blocks) is: after the last station block, else at the end. The markers open with `#`, so a block goes into a file whose comments do; a fragment holding a marker line of its own is refused.
- **A merge** (`mode: merge`) reads the embedded file and yours as JSON objects and merges them. Into an object, a key you lack is added and a key both hold is merged again when both values are objects, or arrays; into an array, an element is appended only when no element deep-equal to it is there. A value of yours that is not Machines' (another scalar, another type) is never replaced: apply leaves it and says `yours, kept` for its JSON pointer, and the setting still reads applied, since apply has nothing left to write. A value Machines wrote that you change since reads `changed since`, and apply leaves it too.
- **The guard.** A fragment usually feeds another app (a hook into its settings), so a setting of either mode that `requires` an app is written only where that app is installed: a named `station settings apply` elsewhere refuses it, naming the app, and writes nothing. A bare apply skips it, as it skips every setting whose app is missing.

### What is kept of your file

- **Strict JSON only.** A file with a comment, a trailing comma, a key twice, two values or no object at its top is refused, naming it, and never rewritten; so is a file past 1 MiB, a fragment past 256 KiB and either nested deeper than 32. A block is refused in a file that is not UTF-8 text, and a damaged block (a start with no end, the id twice) is left as it is.
- **Preserved**: the order of your keys (Machines' new keys come after them, in the fragment's order), each key's and each value's own spelling (escapes, `1.0` against `1`), the indent of the file's first indented line (two spaces when there is none) and whether it ends with a newline. A file whose bytes would not change is not written at all.
- **Not preserved**: any other whitespace. A merge writes the file in `JSON.stringify` layout, one key or element per line, `{}` and `[]` when empty, so a compact or hand-aligned file is laid out again on its first write; a file already in that layout comes back byte for byte.
- **The write** is atomic, as for a package block: beside the file, through the link to the file it names, then renamed over it with the file's own mode and owner; a file Machines makes is 0600, with the folders it lacks. Two settings writing at once take the lock `~/.local/state/station/merged.json.lock` in turn.

### The record and off

Each fragment apply writes is recorded in `~/.local/state/station/merged.json`, per setting and file: each value's JSON pointer and the digest of its canonical form (an element's under its array's pointer), a block's digest, and the file and folders Machines made. Apply judges against that record: a value it wrote that the fragment now holds otherwise is written anew, and one the fragment dropped is taken back while it is unchanged.

`station settings off` removes a value only while its digest is still the recorded one, so a value you changed by hand is left and said (`yours, kept`); an object or array Machines made goes only once empty, and a file and the folders Machines made go only when nothing is left in them. With nothing edited, off gives back your file byte for byte. An element you edited cannot be told from one you deleted, so it is left without a word. With no record (`merged.json` deleted), off takes back nothing.

## Removing the block

Nothing removes the main block for you. Delete the lines from `# >>> station >>>` to `# <<< station <<<` in `~/.zshrc`, `make deprovision` from a checkout removes the binary and leaves the block alone. A package block goes with its package, or by deleting its own two marker lines and what is between them.

## Related

- [Settings](settings.md): the packages that install these files.
- [Footprint](footprint.md): where each one lands.
