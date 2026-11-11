# Design system

Everything `station` shows, in a terminal, a pipe or the TUI, comes from one palette, one mark vocabulary and one document model.

| Section           | Answers                                              |
| ----------------- | ---------------------------------------------------- |
| The palette       | The colours, by meaning                              |
| The marks         | The glyphs and what each says                        |
| Typography        | Margins, rows, columns and empty states              |
| Header and footer | What frames every document                           |
| A run             | How an install, init and apply speak as they work    |
| Pipe and JSON     | The record format a script reads, written once       |
| The TUI           | The shell, its focus, its widths and the loading law |
| Graphs            | The sparklines and bars of Activity                  |

## The palette

Semantic tokens, each a light and a dark value chosen by the terminal's background (Catppuccin Latte and Macchiato, the flavour the dotfiles already use). `src/ui/token/palette.go` is the only file a colour is written in; the layer test refuses a hex or a colour constructor anywhere else.

| Token       | Used for                                                          |
| ----------- | ----------------------------------------------------------------- |
| `Text`      | body text                                                         |
| `Muted`     | hints, details, `○`, secondary text                               |
| `Subtle`    | rules and unfocused borders only; never text that carries meaning |
| `Surface`   | borders and the modal frame                                       |
| `Selection` | the cursor row's background                                       |
| `Accent`    | the focused border, the header name                               |
| `Good`      | `●`                                                               |
| `Warn`      | `◐`                                                               |
| `Bad`       | `✗` and `Error:`                                                  |

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

| Glyph            | Means                                                                                                    |
| ---------------- | -------------------------------------------------------------------------------------------------------- |
| `→`              | a next step, at the foot of a page                                                                       |
| `✓`              | a step passed, a row ticked for a group action, a toast's success                                        |
| `+` / `-`        | a folded / unfolded group in the TUI                                                                     |
| `▸`              | the current entry of the sidebar, the step that runs in the Progress strip, and a CLI step that asks you |
| `◇`              | All machines, in the TUI's tree                                                                          |
| `⏻`, `⚡`        | a Mac kept awake, a battery charging                                                                     |
| `›`              | between the levels of the breadcrumb, and before a text you type                                         |
| `├─`, `└─`       | a Mac's VMs under it                                                                                     |
| `▁▂▃▄▅▆▇█`, `█░` | a sparkline, a bar ([Graphs](#graphs))                                                                   |

The glyph is the meaning and survives a pipe and `NO_COLOR`; colour only reinforces it. `◌` (Muted) marks a row whose read has not answered yet, on a terminal and in the TUI only: a pipe, `NO_COLOR` and `--json` wait for every read, so they never carry it. `⊘` and `?` are Muted, `–` Subtle, each always followed by its word (`page.OffHere`, `page.NotForThisMachine`, `page.NotKnown`); the families draw them as they take their v3 words. Every glyph station prints is a token of `src/ui/token/glyphs.go` (`token.Glyphs`), and the layer test refuses one in any other Go literal of a migrated package. Folding in the TUI uses `+` and `-` before a group title; the cursor is a background, not a glyph.

## Typography

- On a terminal, a CLI page starts at the first column with the line that sums it up (`This Mac · 108 apps for this Mac · 2 updates`), in plain text, then one blank line; the line is never cut: a part it cannot hold goes on the next line, two columns in; its sections read on without one, rows two columns in, and a section with nothing to do is one line. The TUI keeps a one-space margin.
- A section title is `Text` and bold, bright with no hue (vision 07). A titled first line leads with its machine in bold (`This Mac`); its counts stay `Text`, as state colours are on marks only (K1). In a value, only a glyph takes its mark's colour, the words around it stay `Text` (`● 39 ok`). An open section's title counts nothing; a folded one is one line: its title, its count by mark (`○ 34`), then as many of its rows' names as fit (`yazi, dust, eza, …`). A brief section (`page.Brief`) is folded so on a terminal and open in the TUI; a calm one (`page.Calm`) is folded so on a terminal and, in the TUI, folded while every row is calm, as an `Auto` one is. A section that only says something is one line on a terminal too, its title then its words (`Not for this machine  7 apps live on other machines`), and so is a calm section of one row with nothing to do: its title, the row's mark and its words (`Always-on Mac   ○ 5 · for a Mac that hosts servers`), or the row alone when its label already says the title (`– 12 not for this machine · Mac only`); a table, whose columns are headed, keeps its heads.
- What needs you comes first. A page whose rows that need a look stay in their own groups lifts them first on a terminal, in a section of the role `page.Lifted` (Settings' `Needs a look`); the pipe, the TUI and the menu bar read those rows where they are, so a record is printed once.
- Another machine's output starts with `On <machine>:`, the machine `--on` named (tty and pipe).
- A row is a mark, a space, then cells separated by two spaces. Column widths are computed per document, so every section lines up, and the last cell is never padded.
- A cell is a name (strong), a value, a method (muted), then a detail (muted, cut with `…` to the width by its kind, a path in its middle, never cut in a pipe; beside the row's own cells when its last columns are blank and the aligned column would cut it; never cut below its kind's floor: a path or a URL then goes on its own line under the row, any other detail is lost on that width). A long path in a column is cut in the middle before it takes a detail's room. A row that is not `●` may end with the one command that repairs it. On a terminal a name is never cut, nor a command (a column of `page.KindCommand`, as Needs you's fixes): a row whose name is wider than the name column, or whose command does not fit beside the names, writes its name whole and its cells, detail and hint on the lines under it, at the name's column.
- Sizes are humanised by one function; numbers are right-aligned.
- An empty state is one muted row (`none`, `nothing to free`, `no machines registered`) plus a hint naming the command that fills it.

## Header and footer

A CLI page has no header: its first line says where it is (`This Mac · …`, `On mac-mini:`). The TUI draws its own header (UI-SPEC). The footer is a blank line, then the next steps, one per line, each prefixed `→ `, their words after the command at one column for the page.

## A run

An install, `init`, and later `apply` print one page, built by `pages.RunPage` from the steps of the run: the line that sums it up (`✓ Installed 2 apps on This Mac · 1m 12s`, `✗ 1 app failed to install on This Mac`, `✓ Known to station · 3 steps changed`), then each step with its mark, its cells and why it ran before what befell it (`● homebrew  5.0.1  script  required by gh`), then at most two next steps.

While it runs, the terminal's stderr says each step and why, as the spinner's label (`Installing gh · required by …`). A step whose program may ask you on the terminal (an install that needs sudo's password, and every curl installer) is a Muted `▸` line of its own instead, kept with `NO_COLOR` as the glyph and words alone, `▸ Installing homebrew · required by gh · asks your password once`, and no spinner draws over the prompt under it; a pipe hears neither.

A program's own words (a curl installer, `brew bundle`, Homebrew's installer) never reach the terminal: they are appended to the log of each app the step serves, `~/.local/state/station/logs/<app>.log`, and a failure prints only that path (`the install failed: its output is in ~/.local/state/station/logs/claude.log`). An interruption keeps its own words, `interrupted (Ctrl-C)`, and nothing is printed after it.

`install.sh` speaks in the same voice, two lines of its own (`Installing station for darwin/arm64`, `● station v3.1.0 in ~/.local/bin`) or one `✗` line on failure, then hands the terminal to `init`.

## Instant reads

`station overview`, `disk` and `apps` on a terminal draw their frame at once: every row the catalogue or a read table declares, each `◌` with its name only (a size shows its last-known value, Muted, from `~/.cache/station/sizes.json`). Each row fills in as its read answers, the region redrawn at most every 50 ms and clipped to the terminal's height with a Muted `… <n> rows pending` last line; a read past its budget answers `◐ ?` with `no answer in <budget>`. When the last read answers the region is erased and the final document printed once, so the scrollback holds what a pipe would. The reads run on two bounded lanes (16 commands, one walk of 8 workers), and the Overview owns no reading of its own: each of its lines is its section's read through the Router, the same read `apps` and `disk` draw.

## Pipe and JSON

When stdout is not a terminal, or `NO_COLOR` is set, a document prints as records, one per line, with no rule, footer or padding and no colour sequence. The separator is a tab everywhere:

- a section title is `# <title>`; the app and setting categories, and the status Apps, Settings and Background sections, follow it with their counts as `<n> <glyph>` cells (every other title is bare, as its records always were); Accounts' Share list and Kept, one section on a terminal and in the TUI, pipe as the two titled sections `# Share list` and `# Kept`;
- a column for the eye only (a peer's online state and tags, a Mac's role) is no field: the records keep the fields scripts already cut; nor is a column for a row's detail only (the folders a disk row holds, `page.DetailOnly`), which no line of a list shows either;
- a row is `<glyph>` then its cells; a row with no mark starts with an empty field;
- a key and its value are `<key><TAB><value>`;
- a blank cell stays (two tabs); trailing empty cells are trimmed.

So `cut -f2` is a row's name. `--json` prints the call's document, `station.<call>/v1`, two-space indented, with a trailing newline. Every document opens with the same four keys, in this order, filled by `output.Document` from what the leaf knows: `schema`, `generated_at` (RFC 3339 UTC, the machine's clock), `machine` (`{name, label, kind, on, this_mac}`, the machine the call ran on as the interface names it; what the call already said of it wins) and `findings` (its section's Needs-you lines, the ones the Overview ranks, `[]` when it has none); the call's body follows in its own order. A finding has one shape in every document, `core/finding.JSON`: `{severity, section, subject, text, fix: {call, args, command, path}}`, `call`, `command` and `path` null when unset, `args` only with a call, and `machine` added where a document spans machines (`overview`, `ports`, `machines`, `awake`, an `--all-machines` read). `station.status/v1` and `station.error/v1` are printed as they are. An `--all-machines` document holds `machines: [{machine, …, error}]`, `error` (`{type, code, message}`) set for a machine that did not answer. The key conventions are `name`, `ip`, `*_at`, `*_bytes`, `share_list` and `vms`, one key per concept across every schema.

## The TUI

`station` on a terminal opens one Bubble Tea app, the shell of SPEC §9: the header names the target machine with its mark, what it is, its engine and its Awake (a narrow terminal drops the hostname, then Awake, then the engine word); the breadcrumb under it (`All machines › mac-mini › infra › Overview`); the sidebar, MACHINES as a tree (All machines, each Mac with its Needs-you count, its servers then its workspaces) and then the selected machine's sections in the one order of `section.For`, those its own Overview has a line for (the machine gates each read by its capabilities, so a Linux workspace shows no Background; the kind's table stands in until that Overview answers), each with its badge (its worst mark, or Overview's count); the master pane, the section's page; the detail pane, which follows the cursor in a list section; the footer. Every screen is the page its command prints: the shell reads `station <words> --page` (`overview --all --on linux-1`, `ports --all-machines`), so a page has one function on every surface. All machines' Overview is the table of `pages/machines.Table`: the machines page joined, by each machine's name, with Activity and Ports across every machine and each Mac's Overview (KIND, CPU with a sparkline of the session's reads, MEMORY, DISK, PORTS public · private, AWAKE, NEED), then Needs you across every machine; `enter` opens the machine under the cursor.

The app holds exactly one focus: the sidebar, the master or the detail. `tab` and `shift+tab` move it pane to pane, `←` `→` between the sidebar and the list, `[` `]` the previous and next section, `enter` opens (a machine of the tree or of the All machines table, a section, a folded group, a row's detail), `esc` backs up one level (a filter, the detail, then the machine's parent), `/` filters a list and the title writes it (`Apps (/node) 3/41`), `ctrl+r` reads the section again, `?` lists every key of the focused pane then every key that works everywhere, `q` quits. The footer shows only the keys that work for the focused pane. Filter mode gives the text every key but `esc` and `enter`, so a typed `q` filters.

Widths: from 110 columns, three panes; under 110 the detail opens with `tab` over the list; a table too wide for its pane leaves out its least kept columns whole and says which (`85 columns: MEMORY, DISK and AWAKE left out`); under 80 the sidebar goes and the breadcrumb leads; under 60 × 16 one line says the size station needs. With `NO_COLOR` every glyph stays, the focused pane's border is heavy (`┏━┓`) where the others are rounded, and the cursor row is reverse video.

A section's keys are its page's Actions, and a screen of `surface/tui/screens/` maps each to what it does: a command run through the very binary (`apps install jq`), a page shown in the detail pane (a diff, a row's full page), lines shown there (a fix that is yours, a disk row's folders), a text copied, a URL or file opened, a line a new terminal runs, a choice put first (Update by method), another machine's section visited, or a view of the section turned on or off (Disk sorted by name, the matrix's only what needs a look). `n` opens the create form, from every screen: the kind (Workspace or Server), the name (the first free `linux-N`, `server-N`), the Mac (the one shown first), a workspace's lifetime or a server's health check, its size and its network; `←` `→` turn a choice, and a choice that changes the form (a server asks no lifetime) keeps what was typed. A form draws a secret masked and drops it once sent. `a` on a setting whose one question is its values (`claude-token`) opens its values form first, as its command does on a terminal; the answers go to that command on its stdin (`--values-on-stdin`, hidden, the TUI's own), never in its words, and the shell keeps them only while the question the command asks is put. A setting that runs its own dialog (`autologin`, `ssh-key`) is applied on a terminal. The mouse works and nothing needs it: a click selects a sidebar entry or a list's row, a double click opens it, the wheel moves the list's cursor. `ctrl+p` opens the palette (`surface/tui/screens/palette`): one entry per call of `calls.json`, offered on the machine shown when it applies there (a section's call where the machine shows that section, a VM's call on that VM or on each VM of a Mac), each with its section's key and the confirmation it asks announced (`Stop infra (server) on mac-mini · type infra`), searched by its words, the recent ones last; `tab` offers another machine's, `enter` runs it as its key would, a form first when it needs input. `m` opens Switch machine, a filterable list of the machines the session showed, the newest first, then every machine as the tree draws it; `enter` moves the whole TUI there and `-` goes back to the previous one. The TUI opens on the machine its terminal uses (`station use linux-1`), and moving it elsewhere leaves that choice alone: its reads and calls each name their machine. A command runs in the Progress strip, above the footer, while the owner keeps navigating: a spinner that turns once a second and what it does while it runs, with the seconds it has run (`Installing uv · 1/2 · 12 s`), then a `✓` toast that fades after 4 s and says what the command skipped (`infra skipped (server)`), or its failure, which stays with the keys that answer it (`i install mas`, `l log`, `x dismiss`) until dismissed; `tab` focuses it. The strip shows three calls; a fourth is counted (`+1 more · tab to open`) until `tab` focuses the strip, which then lists them all. `q` while a call of this session runs asks first (`1 call still runs here: quit anyway?`). A command that asks first gets its question as the one modal: y or n, or the name typed for a destructive one (`Type xcode to go on:`); the TUI only puts the question the command asked. An empty line names the key that fills it (`No VMs yet. Press n to create one.`).

All machines › Settings is the matrix (vision screen 4): one row per setting, one column per Mac and workspace, a glyph per cell, its OK count, and the machines not read now under it; the cursor is a cell, `[●]`. `↑` `↓` move across settings and `←` `→` across machines (`←` from the first reaches the sidebar), `enter` opens that setting on that machine's own Settings, `a` applies it on every machine it lives on, `F` keeps only what needs a look, `f` opens a checked setting's System Settings pane.

A section with keys of its own is a screen, under `surface/tui/screens/` (Activity, Awake, Ports, a VM's Overview and SSH): it adds to its page's words only what it keeps itself (Activity's graph window, `--graph 6m`, a flag no help shows), draws its lines or lets the list draw them, names its keys for the footer, keeps the list's cursor off the sections it draws no row of (a screen that draws its own lines draws the cursor on the row it is on, and the list's filter still keeps its rows), and says how often it is read again while shown (Activity every 2 s, Awake every second, so a guard's end redraws at once). A screen may read pages beside its own and draw fixed panes at the right of the master pane, under it on a terminal narrower than 110 columns: a server's Overview reads its Activity over the last hour, its Mac's Ports and its ssh command, and draws its state, what it was created with, its last hour, its door and what station leaves to its tooling, with its Health and its Ports beside; its SSH screen reads its Overview and its Mac's ssh command to say each hop that reaches it. All machines › Ports is one wide pane, as SPEC §9.2 draws it: a table per class (MACHINE, PORT, WHAT, then LISTENS ON and DECLARED, or ROUTE and VERIFIED, or the link), the fix of what needs you said once in full, the count of what listens on its machine only, which `a` unfolds (`ports --all`); `enter` opens a row's detail, its QR code with it, over the pane. All machines › Activity draws a row per machine, each VM one step in under its Mac, CPU with a sparkline of the session's reads, how its rates were read, what needs you from them, and each Mac's thermal state and battery, which its page carries only for the TUI (`activity --all-machines --graph`). Awake draws a block per Mac, this one first: its state, then the keys that keep it awake and the lid and battery toggles it reports (E1), its guards; a Mac that never sleeps says what set it and the servers that pause while it sleeps. While the TUI shows a machine's activity (its Activity or its Overview, or All machines for every machine of this Mac) it holds this Mac's engine's watch on it, as the menu bar open does, so the engine samples it every 2 s. A key's call is the CLI's command for its words, run as the same binary: a yes/no question first, or a typed name (`S` stops a server, `D` deletes a VM, `enter` off until its name is typed: `Type infra to stop it`), or a form whose answers make the words; it runs in the Progress strip as every command does. A copy (`c`) goes to the terminal's clipboard (OSC 52) and is toasted there too. The graphs are `ui/atom`'s: a sparkline from its label whose columns are the mean of their points, the newest at its right end, and a bar whose full and light blocks carry the share without colour.

The loading law: the shown section is read first, then All machines and each other Mac's Overview. Every read runs in a command off the frame, its answer arrives as a message, and the view only draws what has answered: until then a section is `◌` with its name. A view never calls a read or the system; the architecture rule `tui-loading-law` refuses a call of the TUI's read outside a command.

## Graphs

Activity is the one section with graphs, in the TUI and the menu bar only; the CLI prints values, a pipe and `--json` the numbers. A graph is a cell of the kind `page.KindSpark`: a sparkline of the eight levels `▁▂▃▄▅▆▇█` (`token.Spark`), one point per sample of the engine's history, each 0 to 100 of its measure's own scale (CPU, memory, swap, disk read and write, network in and out), and a bar of `█` and `░` for the disk's free space. A measure with no points has no graph, never a flat line of zeros. The window is the screen's (`--graph 6m`). Each graph sits after its line's value, so a screen reader and `NO_COLOR` keep the number.

## Related

- [Architecture](architecture.md): the output layer that decides the form.
- [Commands](../reference/commands.md): the commands these documents belong to.
