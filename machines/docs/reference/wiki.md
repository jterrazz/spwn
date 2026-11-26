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
| H1, H22, H38 to H41       | the compile: a kind that exists, a tag that reads, a braced value, a group's bounds, each attribute         |
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

## What is served

`wiki.read`, a call of this process applied on a Mac, answers one page of the corpus that Mac declares, read whole and checked at each ask: `Page`, its id, the first seat's home when empty, at `Budget`, `30s`, `2min` or `10min`, the corpus's own when empty. A Mac declares no wiki until a Machinefile can name one, so outside a world it answers `wiki.none`; a world seeds a Mac's corpus as its `wiki`, a repository's files at its head.

The answer carries every judgement the window draws, and the window decides none:

- the page's head, its paragraph Face, its floor, the seats bar (each seat, its home, the active one) and every page of the corpus with its counts of hard and soft findings;
- each section's items in reading order, the page's own `## Rests on` marked `Rests`: prose as paragraphs, list items, quotes and table lines, each its runs (strong, code, a link and where it points), none at `30s`, where the budget folds the page; a group with its intent, the seats that read it and why it opens folded (`author`, `seat` or `budget`); a block with its kind, Faces (a line Face as the words it reads, its marks dropped, as a page's is), tier, score and causes, its fields in its kind's order with each ref resolved (to a page's title, a block's word or a group's title), and `Shown`;
- `Shown`, the budget's table of S11 per block: at `10min` a lead is a `tile`, a normal block a `card`, a quiet one a `line`; at `2min` a lead is a `card`, a normal block a `line`, a quiet one `counted`; at `30s` a lead no group holds is a `tile` and no other block is shown, a group's one line carrying its lead. A table, a figure and a sub-page are a `tile` but at `30s`;
- a table's rows found and their cells by its columns, its `where` as the words it reads (`by Machines`), a sub-page's child's head, a figure's form, parts and caption, its parts drawn from what it shows when none is written, and the findings that sit on each block and on the page at the budget's signals: needs-you and hard ones at `30s`, remarks too at `2min`, ambient ones as well at `10min`;
- the head's two counts, `NeedsYou`, the needs-you findings served on the page and its blocks, and `Unshown`, the blocks this budget does not show, a folded group counting all of its own.

Its refusals are `wiki.none`, `wiki.unknown-page`, `wiki.unreadable` (`wiki.yaml` absent or refused, its finding said) and `wiki.bad-budget`, each told in `specs/usecases/wiki/read/`.

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
