---
name: studio-repo-structure
description: "Use when organizing or restructuring a repository's documentation, skills, CLAUDE.md, README, or generated docs: where knowledge lives, the Diátaxis shape every docs/ takes (start, guides, concepts, reference, decisions), what gets committed, corpus vs injection layers vs compiler, projections vs presentations."
---

# Repo structure

The doctrine every `@jterrazz` repo follows for where knowledge lives. Canonical home: `studio-station` `docs/concepts/repo-structure.md`.

## Mental model

A repo is three distinct roles, and conflating them is the failure mode this skill guards against:

1. **The corpus**: `README.md` + `docs/`. Where knowledge is _authored_.
2. **The injection layers**: `CLAUDE.md` (agent brief), `skills/` entries. They **route**, they never author.
3. **The compiler**: a repo-specific projector, when there is a source layer worth projecting (rule rows into the generated tables of `package-typescript`'s reference, `make generate` into the toolbelt rosters in studio-station). No source layer, no compiler.

The corpus belongs to the repository: anything a change in this repo would make false (its architecture, its conventions, its runbook, its quirks, the decisions it alone took) is authored in its `docs/`. An outside corpus (the OS wiki) keeps only a pointer to the repo and the contracts between repos. A decision this repo alone took goes to `docs/decisions/NNN-title.md`; one that spans several repos is recorded by the corpus that spans them and linked from here.

Three golden rules, compressed:

- **Written once**: a piece of knowledge lives on one page; everything else links to it. No duplication between README, a skill, and a page.
- **Route, don't retell**: CLAUDE.md and skills are maps: a mental model + a routing table into `docs/`. When they start explaining, knowledge has leaked out of the corpus.
- **The compiler projects, never authors**: generated text sits between `GENERATED` markers or is stamped as generated. Change the source and regenerate; never hand-edit a projection.

Projection vs presentation: the line is which layers a derived file spans. A **projection crosses a layer** (source into docs): it is **committed** and sync-checked. A **presentation re-packages within the docs layer** (an `llms.txt` concatenation, a rendered site): never committed, built in CI only if a delivery target exists.

## The shape, after Diátaxis

A page serves one reader's mood. The manual is cut by mood:

| Path              | Holds                                                                                             |
| ----------------- | ------------------------------------------------------------------------------------------------- |
| `README.md`       | The front door: what, why, a 30-second example that works pasted, a pointer to `docs/README.md`.  |
| `docs/README.md`  | The map: every page of the four sections, grouped by section, each with what it answers.          |
| `docs/start/`     | Tutorials: the first success from nothing, every file complete.                                   |
| `docs/guides/`    | How-to guides: one task each. Maintainer tasks are guides: `develop.md`, `test.md`, `release.md`. |
| `docs/concepts/`  | Explanation: ideas, shape, why. `architecture.md` opens it.                                       |
| `docs/reference/` | Lookup: commands, options, rules, findings, words; exhaustive, dry; generated tables live here.   |
| `docs/decisions/` | `NNN-kebab.md` records beside `_template.md`.                                                     |
| `docs/_assets/`   | Files a page embeds.                                                                              |

No `contributing/` section, no `CONTRIBUTING.md`, no `CHANGELOG.md`. Required pages: `concepts/architecture.md`, `guides/develop.md`, `guides/test.md` always; `guides/release.md` when the repo ships something that runs. Pages are kebab-case, unnumbered, named for a subject (a guide for its task), flat inside their section; the map orders them. Nothing else sits in `docs/`.

A page growing a second mood splits: the option table to `reference/`, the why to `concepts/`, the guide links both. The API is the types: no page restates a signature, no API dump is generated.

## Where to look

| Task                                                              | Read                                                                                                                                                                            |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| The doctrine itself (three roles, golden rules, root layout)      | `studio-station` `docs/concepts/repo-structure.md`                                                                                                                              |
| The manual's shape: the sections, the required pages, naming      | `studio-station` `docs/concepts/repo-structure.md`                                                                                                                              |
| The rule ids the gate prints for that shape                       | `package-typescript`'s [Quality checks](https://github.com/jterrazz/package-typescript/blob/main/docs/reference/quality-checks.md#the-docs-layout-pass), the Docs (layout) pass |
| The studio's own compiler (`make generate`, the toolbelt rosters) | `studio-station` `docs/reference/apps-and-skills.md`                                                                                                                            |

How `@jterrazz` projects compose (packages, naming, CI) is a separate capability: see the `studio-stack` skill.

## Always

- Never restate a page's content in a skill or in CLAUDE.md; link to it instead.
- Never hand-edit a projection; regenerate it (`TEST_UPDATE=1`, `make generate`).
- Regenerate a projection in the same change that touches its source, so the sync check stays green.
- One skill per capability, split by genuinely distinct trigger conditions, not one skill restating two pages.
- Move or rename a page together with every inbound link in the same change: the map, the cross-links, `CLAUDE.md`, the skills, the `read:` of every finding.
