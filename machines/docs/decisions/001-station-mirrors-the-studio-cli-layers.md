# ADR-001: station mirrors the studio CLI's layers

**Status:** Superseded by [ADR-014](014-families-and-contracts.md)
**Date:** 2026-10-31

## Context

The Go CLI had grown as one registry package whose entries held functions, commands that printed directly, and TUIs that probed the machine in their render. The owner asked to isolate the layers well. The estate already has a TypeScript `studio` CLI whose layers (input, command, pure presenter, one output layer, one error block) are proven.

## Decision

Stay in Go and mirror the layers of the `studio` CLI: input, command, domain, pure presenter, one output layer (tty, pipe or json), one error block (`Error:` and one `Hint:`), exit codes from returned errors. One palette and one mark vocabulary serve the whole UI, and the TUI renders the same presenters as the commands. Charm v2 is the UI stack and Fang v2 renders help, version and completion behind the CLI's own error layer, so it can be removed without touching a command. A test, `specs/architecture/layers_test.go`, holds the import table.

Two divergences from the TypeScript reference are deliberate. The pipe record separates cells with a tab, not two spaces, because station's cells hold spaces (`local ip`) and its documents are sectioned. And each leaf is its own Go package named `<noun><verb>`, because Go has no folder-scoped module and the names must be unique.

## Consequences

- A change that crosses a layer fails the gate instead of drifting.
- A script reads `cut -f2` as the name; the format differs from the TypeScript CLI's and is written once in the design system page.
- Fang is an implementation detail with one importer.
