# The wiki's format

A wiki is a corpus of Markdown pages whose facts are tags, read by one file at its root, `wiki.yaml`, in the schema `spwn.wiki/v1`. This page is that file's every key and the rules a page is written by; why tags compile to data and never run is [ADR-095](../decisions/095-the-wiki-is-an-engine-family.md).

| Section        | Answers                                                   |
| -------------- | --------------------------------------------------------- |
| The file       | How strictly `wiki.yaml` is read, and what a refusal says |
| Its keys       | Every key, what it declares, and what the engine refuses  |
| Writing a page | The rules the formatter and the grammar impose on a tag   |

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
