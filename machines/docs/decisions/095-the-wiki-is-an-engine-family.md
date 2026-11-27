# ADR-095: The wiki is an engine family; its tags compile to data and are never executed

**Status:** Proposed
**Date:** 2026-11-25

## Context

The spwn OS wiki writes its facts as tags in Markdown pages (a Rule, a Term, a Decision, a Table of a kind), and a checker judges them: types, references, weights, seats and floors (SPEC-v1 of the OS wiki programme). Two forms were open. A page could be MDX, its tags components a JavaScript runtime renders and runs, or its tags could be a closed grammar the engine reads into records.

MDX runs the code a page carries: a check of a page would then execute it, a corpus could no longer be judged without a Node runtime, and a page could compute what it claims. The checker needs the opposite: a page is a statement, read the same way by the engine, the window and an agent.

Machines already reads its own declarative files this way. A Machinefile and a catalog package are strict YAML, decoded by the catalog into a model's values, and every refusal names its line ([Machinefiles](../reference/machinefiles.md)).

## Decision

The wiki is an engine family, `apps/engine/wiki/`, like `machinefile/`: a model, with the layers of every family. Its schema is `wiki.yaml`, `spwn.wiki/v1`, whose strict shape the catalog decodes and whose meaning the family judges. A page's tags are a closed grammar, compiled to typed records (a page, its sections, its blocks, their props and their findings), and nothing in a page is ever executed: an attribute is a literal, a `ref()` or a query, and the grammar has no expression.

The check is a pure function over the compiled pages and the schema. What it needs of git, of the file system past the pages it is given, or of the network is a later call over a contract, never the core.

The read is bounded so a hostile or overgrown corpus cannot hold the engine: the paths of the compile and the check its worst-input tests reach are linear in the corpus or held to a named bound (the needs cycles are found as strongly connected components, a table keeps 200 rows and every table over one kind and scope shares them, one answer is built to 4 MiB), every slice one page can grow is capped where its cost is incurred, at the compile (a table's columns, a figure's parts, a page's sections, a block's faces, a finding's words) or by the answer's meter consulted while the answer is built, `wiki.yaml` is held to caps before it is judged (its face templates, seats, folders, fields, enum values, icons, repositories, names and floor texts, each passed one hard H44), so a product of a schema's dimension and a page's is bounded by those caps, a finding past its rule's cap is counted and never formatted; what is left superlinear or unbounded over a whole corpus (the compiled density, the retired terms over the prose, a nearest name per unresolved ref, the corpus-wide lists walked per page or block, the index rebuilt per call) is the entry condition `TODO.md` names, "Before a Mac may name its wiki folder", and until it is met no Mac names one (the architecture rule no-wiki-folder); a page past 1 MiB is not compiled (H44), a corpus past 2,000 pages or 64 MiB is refused whole (`wiki.too-large`), a page shows 100 findings of a rule and counts the rest, and an abandoned ask stops between pages. The limits are constants of `apps/engine/wiki/limits.go`; the corpus last read is kept in memory by its stat signature.

Two calls serve the compiled corpus, and both reach it through that one cached read, so neither compiles what the other has: `wiki.read` serves a page at its four levels (a word, a line, a card, the full page), each item saying the first level that shows it, and `wiki.map` serves the corpus as one graph, its seats, pages, sections and blocks, open at the page asked. The three reading budgets of `wiki.yaml` are the engine's own measure, from which it weighs what each level holds; no call takes one and no answer carries one. A link of a page is served with its kind (a page, an anchor, or an `http` or `https` address); a link to any other scheme is served as its words, and said by a remark.

## Consequences

- A page is judged with no runtime of its own: the same records serve the CLI, the window and an agent.
- A tag can compute nothing: a table, a figure and a weight are derived by the engine from the facts, never written as code.
- The grammar is the engine's to grow, release by release; a corpus that needs a form the grammar lacks asks for it rather than writing it.
- A writing rule follows from the closed grammar and the formatter: an attribute value has no escape, so a quote in it is typographic, and the rules a page is written by are the family's reference page.
