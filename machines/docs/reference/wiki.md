# The wiki's format

A wiki is a corpus of Markdown pages whose facts are tags, read by one file at its root, `wiki.yaml`, in the schema `spwn.wiki/v1`. This page is that file's every key, the rules a page is written by, and what the engine judges and weighs of a corpus; why tags compile to data and never run is [ADR-095](../decisions/095-the-wiki-is-an-engine-family.md).

| Section         | Answers                                                              |
| --------------- | -------------------------------------------------------------------- |
| The file        | How strictly `wiki.yaml` is read, and what a refusal says            |
| Its keys        | Every key, what it declares, and what the engine refuses             |
| Writing a page  | The rules the formatter and the grammar impose on a tag              |
| What is judged  | The rules the engine judges a corpus by, and the ones left for later |
| What is weighed | A block's weight, its tier, and what a page shows at the budget      |
| What is served  | The call that answers a page with its reading, and where it is found |
| Beyond the spec | Every reading of SPEC-v1 the engine made on its own                  |

## The file

`wiki.yaml` is read as strictly as a Machinefile: its first key is `wiki: spwn.wiki/v1`, it is one document, it has no anchor, alias or YAML 1.1 scalar, and every key is known at every depth. A refusal is `H42` and names its line: `wiki.yaml:16: H42 kind rule is the engine's: a corpus declares kinds of its own`.

The strict shape is the catalog's (`apps/engine/catalog/wiki.go`); what each value means, and every refusal of a meaning, is the wiki family's (`apps/engine/wiki/schema.go`).

## Its keys

| Key        | Declares                                                                               | Refused when                                                                                   |
| ---------- | -------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `floors`   | the five floors, each an `id`, a `name` and the `question` it answers                  | there are not five, an id comes twice, a name or a question is missing                         |
| `floor_of` | a folder of the corpus (ending in `/`) and its floor, 0 to 4                           | the folder has no `/`, the floor is out of range                                               |
| `seats`    | each seat: `id`, `name`, its folders `path`, the intents it `reads`, `home`, `plan`    | no seat, an id twice, two seats' folders overlap, an intent that does not exist, no home       |
| `repos`    | each repository a page's anchor may name, and its path                                 | a repository with no name                                                                      |
| `mounts`   | a folder another repository's `docs/` is read at: `at`, `repo`, `path`                 | the repository is not in `repos`, two mounts at one folder                                     |
| `budget`   | how long judging the corpus takes at most: `30s`, `2min` or `10min`                    | any other value                                                                                |
| `kinds`    | the corpus's own kinds: `title`, `icon`, `weighs` (0 to 2), `state`, `fields`, `faces` | an engine kind redeclared, more than 16 content kinds with the engine's, a field below refused |
| `icons`    | each icon by name, as its rows                                                         | a name that is no id                                                                           |

A field of a kind is `type` (`sentence`, `text`, `enum`, `ref`, `anchor`, `date`, `bool`, `int`, `list<T>`, or a union `a|b`), `required`, `values` for an enum and only for it, `max` for a sentence and only for it, and `to` for a ref, naming a kind. A field every kind takes (`id`, `floor`, `rests_on`, `weight`, `icon`) is never declared. A kind's `state` names one of its enum fields, and a face template's `{x}` names a field or `id`.

A question that holds a comma is quoted, since a comma ends an entry of a flow mapping, and the file is written in block style so the formatter keeps its lines.

## Writing a page

- An attribute is written at column 0: the formatter strips an indented one.
- A tag a prose link targets (`[rights](#right)`) is written on one line: `#id` resolves only to a single-line tag.
- A tag spans 12 lines at most, the line rule's limit.
- A value holds no em or en dash: the prose rule reads tag lines too.
- A value holds no `"`, since the grammar has no escape: write typographic quotes, “ and ”.

## What is judged

The engine's `Check` (`apps/engine/wiki/check.go`) judges the compiled pages of a corpus and its schema, and nothing else: no git, no file past the pages it is given, no network, no engine state. Each finding names its rule, its page, its line and column, and its level: a hard rule has none, and a soft one is `ambient`, `remark` or `needs-you`. Every rule is once in the registry, `apps/engine/wiki/rules.go`.

| Rules                     | Judge                                                                                                       |
| ------------------------- | ----------------------------------------------------------------------------------------------------------- |
| H1, H22, H38 to H41, H44  | the compile: a kind that exists, a tag that reads, a braced value, a group's bounds, each attribute, size   |
| H2, H3, H4, H5, H6        | each field's type, enum and presence, a ref's `to`; ids; refs; an anchor's repository; a rule's proof       |
| H8, H9, H10, H12, H14     | a term defined once and a retired one written nowhere; floors; `needs`; an authored Face; a component       |
| H15 to H17, H35           | `weight`, one written lead per set, `intent`, `fold`                                                        |
| H19, H20, H36, H23 to H25 | a table's kind, columns and `where`; a sub-page that is a page, a child, and set once                       |
| H26, H28 to H33           | a figure's form and caption; icons and their grids; every page under one seat, every seat owning a page     |
| S2 to S4, S6, S8, S29     | a soft rule broken twice, a refused link or a scenario with no proof, a reading, prose only, a waiver       |
| S11 to S18                | the budget, a written weight against its score, a lead, a group's size, a table's scope and width, children |
| S21, S22, S26, S28        | an icon missing, an intent read by no seat, a tag past 12 lines, a markdown table shaped like a kind        |

Left for later lots, since each reads what the pure core may not:

- H5's other half (the file and the symbol in the clone), H7, H11 and H13 read another repository's files.
- H43 and S25 read the bucket; S1 and S19 read `wiki.lock` and git.
- S5, S7 and S27 read an anchor's file, an ADR's status line, or whether a clone is there.
- S9 and S20 wait for prose links compiled to Links; S10 reads the briefs; S23 waits for the map.
- A tombstone reused (H3's last clause) and the floor of a child page under its parent (H9's page half) read git and the parent's `skip`, which SPEC-v1 does not define.

## What is weighed

A block of a content kind has a weight: its score is the sum of its causes, SPEC-v1 §7.

| Cause           | Points        | When                                                                        |
| --------------- | ------------- | --------------------------------------------------------------------------- |
| `prose-leans`   | 2             | the page's prose links to it, `(#id)`                                       |
| `illustrated`   | 3             | a figure of the page shows it; that figure's ref is not counted as `linked` |
| `linked`        | 1 each, 2 max | other blocks of the page carry a ref to it                                  |
| `guard-hard`    | 2             | a rule whose force is hard or hardened                                      |
| `guard-soft`    | 1             | a soft rule                                                                 |
| `decided-claim` | 2             | a claim of certainty `decided`                                              |
| `decision`      | 1             | a decision                                                                  |
| `scenario`      | 1             | a scenario                                                                  |
| `kind`          | `weighs`      | a block of a corpus kind                                                    |
| `anchored`      | 1             | it carries `code`, `proof`, `carried_by`, `adr`, `path` or `source`         |
| `needs-you`     | 3             | it raises a needs-you finding, S6 or S29                                    |
| `broken`        | 3             | it has a hard finding                                                       |

The blocks of one group are weighed together, and outside a group those of one section on one floor. The lead is the written `lead`, else the highest score at 3 or more, the first in reading order on a tie; a block is quiet when written so or scored 1 or less, and normal otherwise. A group, a table, a lens, a figure and a sub-page have no weight. A page's seat is the seat whose folders hold its id, and its floor its frontmatter `floor`, else `floor_of`; a block is on its page's floor unless it writes its own.

At the corpus budget a page shows at most seven lines (S11): at `10min` a quiet block is a line, at `2min` a normal one, at `30s` none, a group its page's seat does not read is one line and hides its blocks, and at `30s` every group is one line.

The three budgets are the engine's own measure: no call takes one and no answer carries one, and a page is served at its four levels instead, each block's ranks derived from what the three budgets show of it (below).

## What is served

`wiki.read`, a call of this process applied on a Mac, answers one page of the corpus that Mac declares, read whole and checked whenever it changed since the last ask (the limits below): `Page`, its id, the first seat's home when empty. A Mac declares no wiki until a Machinefile can name one, so outside a world it answers `wiki.none`; a world seeds a Mac's corpus as its `wiki`, a repository's files at its head.

One answer serves the page at its four levels of detail, `word`, `line`, `card` and `full`: a level changes how much a block, a group or prose shows and never hides one. Each of them carries `Ranks`, the face it shows at each of the four in that order, and `Max`, the most it opens. The ranks never fall from one level to the next nor pass `Max`, which `levels_test.go` holds over the corpus and `FuzzServe` over any page.

- **A block** takes its ranks from the three budgets' table of S11. It shows its line at `word` when `30s` shows it, and its card at `line` when `2min` shows it a card or a tile. At `card` it shows what `10min` shows, its full face if that is a tile. At `full` it shows its card at least. A block's `Max` is `full`, a figure's `card`.
- **Prose and a group** open no further than a card, their ranks `word`, `line`, `card`, `card`. Prose shows its `First` sentence at `word` and `line`, and all of itself from `card`; `More` says it holds more than that sentence. The sentence ends at the first `.`, `!`, `?` or `:` that a space or the end of the text follows, outside code, unless the `.` closes `e.g.`, `i.e.`, `cf.`, `vs.` or an initial. A group shows its blocks' words at `word`, its blocks from `line`, and its lead too from `card`.
- **The lede** is the page's paragraph: its first sentence at `word`, all of it after. On a record's page the lede is its decision instead, a line at `word` and `line` and full from `card`.

The answer carries every judgement the window draws, and the window decides none:

- the page's head, its paragraph Face, its floor; on a record's page its `Number`, `Title` and `Decision`, the record's head and what the first block of the corpus that names it `decides`;
- on a block, or a group, whose floor is not its page's, that floor as `OwnFloor`;
- each section in reading order, with its `Label` and `Icon`: the opening section is `Top` and `home`, `## Rests on` is `rests` and marked `Rests`, a section that holds a `guard` group is `guard`, and any other `define`;
- each section's items in reading order. Prose is served as paragraphs, list items, quotes and table lines, each its runs (strong, code, a link and where it points), with an id, `p<section>.<n>`. A group carries its intent, the seats that read it and why it opens folded (`author` or `seat`). A block carries its kind, Faces (a line Face as the words it reads, its marks dropped, as a page's is), tier, score and causes, and its fields in its kind's order with each ref resolved (to a page's title, a block's word or a group's title);
- the mirror: the first table outside any group whose rows are exactly a group's blocks says so, the table's `Mirrors` naming the group and the group's `MirroredBy` the table, with `MirroredFrom`, the first level that draws the table as a card at least, from which the group shows its blocks as words;
- a decision as its record says it, `Decision`: the number, title, status, date and the records it amends or is amended by, read from the head of the page its `adr` names, and what the block says it `decides`; `Resolves` is false, and nothing of the record is served, when the `adr` names no record of the corpus;
- a table's rows found, `Count`, and their cells by its columns, its `where` as the words it reads (`by Machines`), a sub-page's child's head, a figure's form, parts and caption, its parts drawn from what it shows when none is written (a `stack` that shows a table counts, for each page or block its rows name, how many rows name it, most first);
- every finding that sits on a block or on the page, each with `From`, the first level that shows it (SPEC §15): `word` for a needs-you or hard one, `line` for a remark, `card` for an ambient one;
- what asks for the reader's eye, in one order wherever it is listed (needs-you first, then hard, then remarks, then ambient): a block's findings in that order, `NeedsYou` on a block or group one of whose findings needs the owner, and the page's `Attention`, every finding on the page and its blocks, each with the word of what it is about and `At`, the address of its block, or of the folded group that holds it, empty for the page's head;
- the head's counts, `Blocks` and `NeedsYou`, the needs-you findings of `Attention`;
- `Reading`, one per level: the `Words` the level shows, the `Seconds` they take at 230 words a minute, its `Label`, and `Same` when the level shows exactly what the one before it shows.

A label is always round: `Seconds` is the exact value, and the label rounds it to the nearest 5 s under a minute (5 s at least, and 57.5 s or more is `1 min`), to the nearest 15 s from a minute to five minutes, and to the nearest minute past them. The labels never fall from one level to the next, and two levels that show different things may share one.

Its refusals are `wiki.none`, `wiki.unknown-page`, `wiki.unreadable` (`wiki.yaml` absent or refused, its finding said; or a file of the folder its read refuses, a link out of it or a FIFO, named by its path inside the wiki only) and `wiki.too-large`, each told in `specs/usecases/wiki/read/`.

### The map

`wiki.map`, beside it, answers the graph the window's canvas navigates the corpus by, in one answer, so the window derives none of it. It reads the same compiled corpus as `wiki.read`, through the same cache: an unchanged corpus is compiled once for both calls.

- **Its nodes** are the seats, `S:<seat>`, in the schema's order and listed as `Roots`; each seat's pages, `P:<page>`, lowest floor first, the records after the other pages of their floor, then by word; and under the page `Open` names, its sections, `C:<page>:<i>` with `i` the section's index in `wiki.read`'s `Sections` (a section with nothing in it is left out), and what each holds, its blocks, groups and paragraphs, `B:<page>:<id>`. A page not open is served with no kids. `Ref` is the id `wiki.read` serves a node by: the seat's, the page's, the section's anchor (empty for the opening) or the block's, the group's or the prose's, `p<section>.<n>`.
- **Each node** carries its `Depth` (1 for a seat to 4 for a block, a group or a paragraph), `Parent` (null for a seat) and `Kids` in reading order; `Needs`, the needs-you findings under it, its kids' and its own (a page owns those on its head, a block its own, a group its blocks'); `Marks`, one per kid, true where that kid holds one (a group marks its blocks), eleven at most, `More` counting the rest; `Band`, the canvas row of its seat, the lowest floor of the seat's pages; and its words: `Label`, then `TitleParts`, `KickerParts` and `LineParts` when they say more than the label, its `Icon` (the one `wiki.read` serves the block with, so the canvas and the sheet draw the same) and `Count`, what it holds, said with its kind's noun (pages, sections, a section's blocks, a group's blocks). A seat names its `Home`, the node of the page it opens at.
- **The head** is `Home`, the root's word and line, the pages and seats it holds, the needs-you findings and the pages they sit on, and the corpus's hard and soft findings; `Floors` are the ladder, each with its band's `LabelParts`, `["E1", "Produit"]`; `Icons` the dot icons its nodes are drawn with.
- **Its words carry no glyph.** A field the canvas draws as several words in one line is served as its parts in order, and the window joins them with the kit's separator: a record's title is `["ADR-002", <title>]`, a group's kicker `["Group", <intent>]`, a section's line and a group's its blocks' words. A paragraph's label and title are its first three and four words, with `LabelCut` and `TitleCut` true when it holds more; the window draws the kit's ellipsis after them. The glyphs are the kit's alone (`specs/architecture/kit_test.go`, rule 16).

Its size is the words it carries. Over the seven pages of `apps/engine/wiki/testdata/corpus/` (measured 2026-11-26 on `development-linux-mini`), the map with none open is 4,993 bytes for 11 nodes, and with a page open 6,839 to 13,425 bytes for 15 to 36; a node averages 264 bytes for a seat, 423 to 454 for a page, 347 to 518 for a section, 187 to 331 for a block, 323 to 356 for a group and 328 to 444 for a paragraph. Over 500 pages, the map with none open is 240,937 bytes for 504 nodes, built in 79 ms once the corpus is read, and the largest with a page open 249,673.

A map open at no page of its own is refused `wiki.unknown-page`; its other refusals are `wiki.read`'s, each told in `specs/usecases/wiki/map/`.

### The limits

The read both calls share is bounded, each limit a constant of `apps/engine/wiki/limits.go`:

- a page file past 1 MiB is sized and never read: it is served with no content and its one hard finding, H44, naming its size; a file that grows after it was sized is read only to one byte past the limit, and is served the same way;
- a corpus past 2,000 pages or 64 MiB of files read is refused whole, `wiki.too-large`; the folder's walk stops at the limit;
- a page shows at most 100 findings of one rule, then one more of that rule that counts the rest;
- an abandoned ask stops the read between two pages, compiled or checked.

The corpus last read is kept in memory by its signature (each file's path, size and modification time; in the fake world, its bytes' hash), so an unchanged corpus is served without a file read; any change reads and compiles it anew. One corpus is kept, the last read, so the memory it holds is bounded by the limits above, and asks made at once each serve a corpus read whole.

The signature has one known limit: a page rewritten at its size whose modification time does not move is served as it was until the next change. That happens when a program restores the time, or when two writes fall in one tick of the file system's clock, a few milliseconds on Linux. The fake world has no times, so its signature hashes every byte and has no such limit.

## Beyond the spec

Each reading the engine made where SPEC-v1 is silent:

- A block's `floor` and a page's frontmatter `floor` are a number, 0 to 4, like `floor_of`.
- A value `null` is a value not written.
- A union `ref|anchor|sentence` reads a string of the anchor form (`repo:path[#symbol]`) as an anchor, any other string as a sentence.
- `rests_on` is `"reading"` or a ref; frontmatter `needs` is one page or a list, `[a, b]`.
- A frontmatter key outside `kind`, `floor`, `needs`, `line`, `word`, `icon`, `rests_on` and `retired` is H41.
- A term is defined once across pages (H8); twice in one page is H3, an id twice.
- A retired term's words are its id, its dashes read as spaces, and they are found in prose only.
- `where` takes `has` on a list field only and `is` on a single value only (H36).
- `weight` written on a group, a table, a lens, a figure or a sub-page is H15; a second written lead is H16 outside a group too, in the section and floor its blocks are weighed in.
- The built-in icons are names: each engine kind, each intent, each rule force, link right and component level; their grids are the window's. A kind's icon is its `icon` key, a seat's the icon named its id.
- A sub-page's child is a page under its folder, or one whose `needs` names it.
- A markdown table asks for blocks (S28) when it has two columns or more, each a field of one content kind.
- S13 and S18 count a page's weighed blocks.
- At `30s` a lead outside any group is served as a tile: §15 says only that a group's one line carries its lead.
- A figure with no `parts` is drawn from what its `shows` names, a group's blocks or the one block, each by its word and placed nowhere.
- Weight causes are listed with their points, and a page's own `## Rests on` section or `rests_on` frontmatter raises S6 on the page.
- A finding about `wiki.yaml` names that file and its line, column 1.
- An anchor names a page of the corpus when a mount reads its file, or when its repository holds the corpus: a repository whose path climbs n folders (`studio-os: ../../..`) holds the corpus n folders down, so the anchor's path past its first n folders is the page.
- A record is a page under a `decisions/` folder titled `ADR-n: title`; its status, date, and the records it amends or is amended by are read above its first section, as `**Key:** value` or as a sentence opening with the key (`Amended by [ADR-086](...)`). A status is `Accepted` only as written so.
