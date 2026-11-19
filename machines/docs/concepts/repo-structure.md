# Repo structure

A repository is a written corpus, some thin injection layers, and a compiler that projects source into machine-readable forms. Keep those three roles distinct and knowledge stays in exactly one home.

This is the doctrine every `@jterrazz` repo follows, whatever the language. It is the canonical home for it.

## The corpus belongs to the repository

A repository's own knowledge lives in its `docs/`. Anything a change in this repository would make false (its architecture, its layers, its conventions, its runbook, its quirks, the decisions it alone took) is authored here and nowhere else.

An outside corpus holds two things about a repository and no more: a pointer to it, and the contracts it has with other repositories. The OS wiki (`studio-os`, `home/<brand>/wiki/`) is the outside corpus in practice: it carries one row per repository and the agreements that span several of them. When a sentence there could be falsified by one repository alone, it is in the wrong home.

## The three roles

1. **The corpus**: `README.md` plus `docs/`, cut into the five sections below. This is where knowledge is _authored_: prose a human writes and maintains.
2. **The injection layers**: `CLAUDE.md` (the agent brief, and no `AGENTS.md` beside it) and the `skills/` entries. They **route**, they never author. A skill points at a page; it does not restate what the page says.
3. **The compiler**: a repo-specific command that projects a source layer into the corpus. In `package-typescript`, the rule rows project into the tables of its reference pages; in `studio-station`, `make generate` projects the install registry into the toolbelt skill rosters. A repo with nothing to project simply has no compiler.

## Three golden rules

- **A piece of knowledge is written once.** It lives on one page. Everything else links to it. No paragraph is duplicated between the README, a skill, and a page.
- **CLAUDE.md and skills route without retelling.** They are maps, not territory: a mental model plus a routing table into `docs/`. When they start explaining, the knowledge has leaked out of the corpus.
- **The compiler projects; it never authors.** Generated text is a projection of a source layer, held between `GENERATED` markers or stamped as generated. You change the source and regenerate; you never edit a projection.

## Committed projections vs presentations

The dividing line is which layers a derived file spans:

- **A projection crosses a layer boundary**: it compiles one layer into another (rule rows into a reference table, a registry into a skill roster). It is **committed**, so it is diffable, greppable, and agent-readable straight from the tree, and kept honest by a sync check.
- **A presentation re-packages within one layer**: it re-arranges docs into another docs shape (an `llms.txt` concatenation, a rendered HTML site). It authors no new knowledge, so it is never committed; it is **built in CI only if a delivery target exists** (a hosted site, an agent-ingestion endpoint). With no such target, no presentation is produced at all: the committed corpus is read directly.

## The manual's shape

A reader comes to a manual in one of four moods, and each mood wants a different page. The newcomer wants to be walked to a first success; the practitioner wants the steps of one task; the curious wants to understand why; the expert wants to look one fact up. A page that serves two moods serves both badly: a tutorial interrupted by an option table loses the newcomer, a reference padded with rationale buries the fact. So the manual is cut by mood (the [Diátaxis](https://diataxis.fr) pattern) into four sections, plus the record of decisions:

| Path              | Holds                                                                                                                                                              |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `README.md`       | The front door: what the thing is, why it exists, and a 30-second example that works when pasted. Then a pointer to `docs/README.md`. Not a manual.                |
| `docs/README.md`  | The map: one sentence, then every page of the four sections, grouped by section, each with what it answers.                                                        |
| `docs/start/`     | Tutorials: the first success, from nothing, step by step, every file complete as written. For the newcomer.                                                        |
| `docs/guides/`    | How-to guides: one task each, for a reader who knows what they want to do. A maintainer's tasks are guides like any other: `develop.md`, `test.md`, `release.md`.  |
| `docs/concepts/`  | Explanation: the ideas, the shape and why the lines are drawn where they are. `architecture.md` opens it.                                                          |
| `docs/reference/` | Lookup: every command, option, rule, finding and word, exhaustive and dry, in the order a reader searches them. A generated table lives here, between its markers. |
| `docs/decisions/` | The records of the decisions this repository alone took, `NNN-kebab.md`, beside the `_template.md` they are cut from.                                              |
| `docs/_assets/`   | Files a page embeds. Underscored, because it is ground rather than a section.                                                                                      |

There is no `contributing/` section: how to change, prove and ship the repository is what its maintainer does, and a maintainer is a reader with a task. Those pages are guides.

### The pages every manual holds

A reader landing on an unfamiliar repository knows where its shape is, how it is changed, what proves a change and how it ships, before opening a file, and so does an agent:

| Page                       | Owns                                                                                                    | When                                          |
| -------------------------- | ------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| `concepts/architecture.md` | The shape of the thing: its parts, their boundaries, and why the lines are drawn there.                 | always                                        |
| `guides/develop.md`        | How a change is made: the toolchain, the loop, which file a change opens, what it owes.                 | always                                        |
| `guides/test.md`           | What proves a change: the suites, the ground they stand on, how a golden is regenerated.                | always                                        |
| `guides/release.md`        | How it ships and how it runs: the release or the deploy, what leaves the tree, the footprint it leaves. | when the repository ships something that runs |

`architecture`, `develop` and `test` are required of every repository: everything has a shape, a way of being changed, and something that proves the change. A repository that cannot fill one of them has found a real hole, not an exemption. `release` is required when the repository ships something that runs: an image it deploys, a package it releases to a registry, a binary it releases, a platform it provisions. The rule only ever requires it; it never forbids it. A repository that ships in a way its tree does not show (a tagged release in `.github/`) writes its `release.md` unasked, and a library that writes a hollow one gets a review comment, not a red gate.

`start/` is what a repository with outside users opens on: a package, a CLI, a framework. An application nobody installs may leave it empty: its newcomer is a maintainer, and `guides/develop.md` is their first page.

### How a page is named and placed

- **A page is named for its subject**, in kebab case, with no number: `guides/adopt-a-stricter-release.md`, `reference/configuration.md`. The map gives the order; the name never does, so a page can move without renaming its neighbours. A name never names a moment: `exploration`, `review`, `notes`, `draft`, `wip` name a date rather than a thing, and what happened on a date is a decision record or git's history.
- **A guide is named for its task**: a verb first where the language allows it, and a concept or a reference page for its thing.
- **A section is flat.** A page sits directly in its section; a sub-folder means a section is hiding inside another.
- **Nothing else sits in `docs/`.** The map, the five sections and `_assets/`; a loose file or a sixth folder is a page that has not found its section.
- **The map routes to its own pages and nothing else.** It lists every page of the four sections, once, and links `decisions/`; it is the one file an outside corpus points at, so a page can be split or moved without breaking a link held somewhere else.

When a page starts serving a second mood (a guide growing an option table, a reference page arguing for itself), the second half moves to its section and the first links to it. That is the one-owner rule applied inside the corpus: the option is owned by the reference, the reason by the concept, and the guide links both.

This page owns the shape. Where a checker holds part of it mechanically, the rule ids and the sentence each one prints belong to that checker, so that the message and the rule cannot drift apart. This page never lists them.

## Canonical layout of the root

| Path        | Role                                                                                                           |
| ----------- | -------------------------------------------------------------------------------------------------------------- |
| `README.md` | The front door: what, why, a 30-second example, a pointer into `docs/`.                                        |
| `CLAUDE.md` | The agent brief: mental model + routing table into `docs/`. Routes, does not retell. No `AGENTS.md` beside it. |
| `TODO.md`   | The working backlog, when one is kept.                                                                         |
| `docs/`     | The corpus, in the shape above.                                                                                |
| `skills/`   | Injection layer for agents: one skill per capability, routes into `docs/`.                                     |

The list is exhaustive for the knowledge at the root: a file that is not on it does not belong there, and a repository carries no `CHANGELOG.md` and no `CONTRIBUTING.md`.

## Decision records

A decision this repository alone took is recorded in `docs/decisions/NNN-title.md`, numbered in the order the decisions were taken. A record carries a status and a date, then three sections: the context that forced the decision, what was decided, and the consequences that follow. The mold sits where the record is written: `docs/decisions/_template.md`, copied for each new record.

The status is `Proposed` until the repository's owner accepts it. A record that replaces an earlier one says so, and the record it replaces is marked superseded by it. The link closes both ways, so a reader who lands on the old one is told where the live decision is.

A decision that spans two or more repositories is not recorded here. It belongs to the corpus that spans them, and this repository links to it.

## Packages vs applications

A package's reference documents what its types cannot say: the commands, the options its settings take, the rules and findings it prints, the words it uses. The API itself is the types the package exports: the declarations an editor resolves are the one copy, so no reference page restates a signature and no API dump is generated into `docs/`. An **application** (an API server, a product CLI, a web app) adopts the doctrine in full (the corpus, the routing layers, the single-home rule), and its `reference/` holds what its operators look up. An application may still have its own compiler when it holds a source layer worth projecting, as this repo does with its install registry.

## Language specifics

The doctrine applies in full to every repo: a written corpus in the shape above, thin routing layers, and knowledge that lives in exactly one home. Only the _compiler_ half is language-specific. The projection-vs-presentation criterion is unchanged: a cross-layer compile is committed, a same-layer re-packaging is a presentation built only when a delivery target exists.

## Related

- [The stack](stack.md): how `@jterrazz` projects compose.
- [Apps and skills](../reference/apps-and-skills.md): this repo's own compiler, `make generate`.
