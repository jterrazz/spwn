# ADR-031: The catalog defines, the Machinefile chooses

**Status:** Proposed
**Date:** 2026-11-11

## Context

The catalog was Go values in `src/catalog/`: a list of apps built with placement helpers (`Everywhere`, `OnMacOS`, `MacOnly`), a list of settings, and each app's disk row in a third file. Adding an app meant editing three places, and no one outside the binary could read, check or override an item. The Machinefile (`station.machinefile/v1`, [ADR-030](030-init-makes-a-machine-known-apply-carries-its-machinefile.md)) needs items it can name, from layers it does not own: the user's, a shared source, the built-in ones.

## Decision

Two file types, one responsibility each. A catalog item says how to install or set one thing on each system; a Machinefile says what one machine carries, and never defines an item.

- An item is one YAML file, `station.catalog/v1`: a `manager`, an `app` or a `setting` (what it is), and its `methods` (how it gets there): an ordered list, each method a selector in the Machinefile's words (`os`, `family`, `version`, `arch`, and `kind: computer | workspace | server`) and exactly one manager key (`brew`, `cask`, `mas`, `apt`, `npm`, `uv`, `bun`, `mise`, `github-release`, `script`, `skill`, `provided-by`; for a setting `file`, `builtin`, `macos-defaults` and its builtin effects). The file owns the whole item, its disk row included; its category is its folder.
- The built-in items are embedded, `src/catalog/items/<category>/<name>.yaml`, so the binary stays hermetic.
- The decoder is strict: one YAML 1.2 document, every key known, no anchor, alias or merge key, no scalar YAML 1.1 would read otherwise (`yes`, `0755`), every value in its closed list. A file naming a newer schema is refused with `update station`. Its JSON Schema is projected from the Go types by `make generate` into `schemas/station.catalog.v1.json`.
- `wire` loads the catalog once a run and hands the `catalog.Catalog` value to every reader; no reader reaches a package-level list.
- v1 is whole before any file outside the binary uses it: every key the design names (`after`, `phase`, `needs`, `after-apply`, `renamed-from`, `deprecated`, typed `options`, a method's `provides`, `upgrade`, `pin`, `path`) is declared and decoded, read or not yet, so a key added later never breaks an older station with `field not found` instead of `update station`. The format is [The catalog's item format](../reference/catalog.md).
- `provided-by: <item>` is a method: the command comes with another item's install, which is its requirement. `herdr-core` (`hc`) is the first, provided by `herdr-app` on a Mac.

## Consequences

- One file adds an item; the catalog's YAML is read by the word rules like the Go it replaced.
- The user and shared layers (L2.2) decode with the same strict decoder, so a user's typo is a refusal with its line, never a silent default.
- The lists stand by category, in the categories' tables, then by name: a file carries no order, so adding one never renumbers the others.
- A setting carries one method in v1, the one effect the settings family applies; images and dev-server profiles stay Go values until their own lots move them.
