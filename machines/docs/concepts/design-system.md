# Design system

Everything `station` shows, in a terminal or a pipe, comes from one palette, one mark vocabulary and one document model.

| Section           | Answers                                           |
| ----------------- | ------------------------------------------------- |
| The palette       | The colours, by meaning                           |
| The marks         | The glyphs and what each says                     |
| Typography        | Margins, rows, columns and empty states           |
| Header and footer | What frames every document                        |
| A run             | How an install, init and apply speak as they work |
| Pipe and JSON     | The record format a script reads, written once    |
| Graphs            | How Activity's graphs are drawn                   |

## The palette

Semantic tokens, each a light and a dark value chosen by the terminal's background (Catppuccin Latte and Macchiato, the flavour the dotfiles already use). `apps/engine/ui/token/palette.go` is the only file a colour is written in; the layer test refuses a hex or a colour constructor anywhere else.

| Token     | Used for                                                          |
| --------- | ----------------------------------------------------------------- |
| `Text`    | body text                                                         |
| `Muted`   | hints, details, `○`, secondary text                               |
| `Subtle`  | rules and unfocused borders only; never text that carries meaning |
| `Surface` | a button's text and an idle button's ground                       |
| `Accent`  | titles and buttons                                                |
| `Good`    | `●`                                                               |
| `Warn`    | `◐`                                                               |
| `Bad`     | `✗` and `Error:`                                                  |

The same tokens colour Fang's help and the forms that ask a setting's values.

## The marks

| Mark | Means                                       |
| ---- | ------------------------------------------- |
| `●`  | ok                                          |
| `◐`  | degraded, or unknown                        |
| `○`  | absent                                      |
| `✗`  | broken                                      |
| `◌`  | pending                                     |
| `⊘`  | off here, your choice                       |
| `–`  | not for this machine                        |
| `?`  | not managed by station, or not readable now |

Beside the marks, a few glyphs carry structure, never a state:

| Glyph      | Means                              |
| ---------- | ---------------------------------- |
| `→`        | a next step, at the foot of a page |
| `✓`        | a step passed                      |
| `▸`        | a CLI step that asks you           |
| `├─`, `└─` | a Mac's VMs under it               |
| `▁▂▃▄▅▆▇█` | a sparkline ([Graphs](#graphs))    |

The glyph is the meaning and survives a pipe and `NO_COLOR`; colour only reinforces it. `◌` (Muted) marks a row whose read has not answered yet, on a terminal only: a pipe, `NO_COLOR` and `--json` wait for every read, so they never carry it. `⊘` and `?` are Muted, `–` Subtle, each always followed by its word (`page.OffHere`, `page.NotForThisMachine`, `page.NotKnown`); the families draw them as they take their v3 words. Every glyph station prints is a token of `apps/engine/ui/token/glyphs.go` (`token.Glyphs`), and the layer test refuses one in any other Go literal of a migrated package.

## Typography

- On a terminal, a CLI page starts at the first column with the line that sums it up (`This Mac · 108 apps for this Mac · 2 updates`), in plain text, then one blank line; the line is never cut: a part it cannot hold goes on the next line, two columns in; its sections read on without one, rows two columns in, and a section with nothing to do is one line.
- A section title is `Text` and bold, bright with no hue (vision 07). A titled first line leads with its machine in bold (`This Mac`); its counts stay `Text`, as state colours are on marks only (K1). In a value, only a glyph takes its mark's colour, the words around it stay `Text` (`● 39 ok`). An open section's title counts nothing; a folded one is one line: its title, its count by mark (`○ 34`), then as many of its rows' names as fit (`yazi, dust, eza, …`). A brief section (`page.Brief`) is folded so on a terminal and open in the menu bar; a calm one (`page.Calm`) is folded so on a terminal, as an `Auto` one is while every row is calm. A section that only says something is one line on a terminal too, its title then its words (`Not for this machine  7 apps live on other machines`), and so is a calm section of one row with nothing to do: its title, the row's mark and its words (`Always-on Mac   ○ 5 · for a Mac that hosts servers`), or the row alone when its label already says the title (`– 12 not for this machine · Mac only`); a table, whose columns are headed, keeps its heads.
- What needs you comes first. A page whose rows that need a look stay in their own groups lifts them first on a terminal, in a section of the role `page.Lifted` (Settings' `Needs a look`); the pipe and the menu bar read those rows where they are, so a record is printed once.
- Another machine's output starts with `On <machine>:`, the machine `--on` named (tty and pipe).
- A row is a mark, a space, then cells separated by two spaces. Column widths are computed per document, so every section lines up, and the last cell is never padded.
- A cell is a name (strong), a value, a method (muted), then a detail (muted, cut with `…` to the width by its kind, a path in its middle, never cut in a pipe; beside the row's own cells when its last columns are blank and the aligned column would cut it; never cut below its kind's floor: a path or a URL then goes on its own line under the row, any other detail is lost on that width). A long path in a column is cut in the middle before it takes a detail's room. A row that is not `●` may end with the one command that repairs it. On a terminal a name is never cut, nor a command (a column of `page.KindCommand`, as Needs you's fixes): a row whose name is wider than the name column, or whose command does not fit beside the names, writes its name whole and its cells, detail and hint on the lines under it, at the name's column.
- Sizes are humanised by one function; numbers are right-aligned.
- An empty state is one muted row (`none`, `nothing to free`, `no machines registered`) plus a hint naming the command that fills it.

## Header and footer

A CLI page has no header: its first line says where it is (`This Mac · …`, `On mac-mini:`). The footer is a blank line, then the next steps, one per line, each prefixed `→ `, their words after the command at one column for the page.

## A run

An install, `init`, and later `apply` print one page, built by `pages.RunPage` from the steps of the run: the line that sums it up (`✓ Installed 2 apps on This Mac · 1m 12s`, `✗ 1 app failed to install on This Mac`, `✓ Known to station · 3 steps changed`), then each step with its mark, its cells and why it ran before what befell it (`● homebrew  5.0.1  script  required by gh`), then at most two next steps.

While it runs, the terminal's stderr says each step and why, as the spinner's label (`Installing gh · required by …`). A step whose program may ask you on the terminal (an install that needs sudo's password, and every curl installer) is a Muted `▸` line of its own instead, kept with `NO_COLOR` as the glyph and words alone, `▸ Installing homebrew · required by gh · asks your password once`, and no spinner draws over the prompt under it; a pipe hears neither.

A program's own words (a curl installer, `brew bundle`, Homebrew's installer) never reach the terminal: they are appended to the log of each app the step serves, `~/.local/state/station/logs/<app>.log`, and a failure prints only that path (`the install failed: its output is in ~/.local/state/station/logs/claude.log`). An interruption keeps its own words, `interrupted (Ctrl-C)`, and nothing is printed after it.

`install.sh` speaks in the same voice, two lines of its own (`Installing station for darwin/arm64`, `● station v3.1.0 in ~/.local/bin`) or one `✗` line on failure, then hands the terminal to `init`.

## Instant reads

`station overview`, `disk` and `apps` on a terminal draw their frame at once: every row the catalogue or a read table declares, each `◌` with its name only (a size shows its last-known value, Muted, from `~/.cache/station/sizes.json`). Each row fills in as its read answers, the region redrawn at most every 50 ms and clipped to the terminal's height with a Muted `… <n> rows pending` last line; a read past its budget answers `◐ ?` with `no answer in <budget>`. When the last read answers the region is erased and the final document printed once, so the scrollback holds what a pipe would. The reads run on two bounded lanes (16 commands, one walk of 8 workers), and the Overview owns no reading of its own: each of its lines is its section's read through the Router, the same read `apps` and `disk` draw.

## Pipe and JSON

When stdout is not a terminal, or `NO_COLOR` is set, a document prints as records, one per line, with no rule, footer or padding and no colour sequence. The separator is a tab everywhere:

- a section title is `# <title>`; the app and setting categories, and the status Apps, Settings and Background sections, follow it with their counts as `<n> <glyph>` cells (every other title is bare, as its records always were); Accounts' Share list and Kept, one section on a terminal, pipe as the two titled sections `# Share list` and `# Kept`;
- a column for the eye only (a peer's online state and tags, a Mac's role) is no field: the records keep the fields scripts already cut; nor is a column for a row's detail only (the folders a disk row holds, `page.DetailOnly`), which no line of a list shows either;
- a row is `<glyph>` then its cells; a row with no mark starts with an empty field;
- a key and its value are `<key><TAB><value>`;
- a blank cell stays (two tabs); trailing empty cells are trimmed.

So `cut -f2` is a row's name. `--json` prints the call's document, `station.<call>/v1`, two-space indented, with a trailing newline. Every document opens with the same four keys, in this order, filled by `output.Document` from what the leaf knows: `schema`, `generated_at` (RFC 3339 UTC, the machine's clock), `machine` (`{name, label, kind, on, this_mac}`, the machine the call ran on as the interface names it; what the call already said of it wins) and `findings` (its section's Needs-you lines, the ones the Overview ranks, `[]` when it has none); the call's body follows in its own order. A finding has one shape in every document, `core/finding.JSON`: `{severity, section, subject, text, fix: {call, args, command, path}}`, `call`, `command` and `path` null when unset, `args` only with a call, and `machine` added where a document spans machines (`overview`, `ports`, `machines`, `awake`, an `--all-machines` read). `station.status/v1` and `station.error/v1` are printed as they are. An `--all-machines` document holds `machines: [{machine, …, error}]`, `error` (`{type, code, message}`) set for a machine that did not answer. The key conventions are `name`, `ip`, `*_at`, `*_bytes`, `share_list` and `vms`, one key per concept across every schema.

## Graphs

Activity is the one section with graphs, and the menu bar draws them; the CLI prints values, a pipe and `--json` the numbers. A graph is a cell of the kind `page.KindSpark`: one point per sample of the engine's history, each 0 to 100 of its measure's own scale (CPU, memory, swap, disk read and write, network in and out). A terminal's renderer draws it as a sparkline of the eight levels `▁▂▃▄▅▆▇█` (`token.Spark`); the menu bar's machine page draws charts from the same points ([Activity](../reference/activity.md#graphs)). A measure with no points has no graph, never a flat line of zeros. Each graph sits after its line's value, so a screen reader and `NO_COLOR` keep the number.

## Related

- [Architecture](architecture.md): the output layer that decides the form.
- [Commands](../reference/commands.md): the commands these documents belong to.
