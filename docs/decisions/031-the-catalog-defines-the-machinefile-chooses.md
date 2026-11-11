# ADR-031: The catalog defines, the Machinefile chooses

**Status:** Proposed
**Date:** 2026-11-11

## Context

The catalog was Go values in `src/catalog/`: a list of apps built with placement helpers (`Everywhere`, `OnMacOS`, `MacOnly`), a list of settings, and each app's disk row in a third file. Adding an app meant editing three places, and no one outside the binary could read, check or override an item. The Machinefile (`station.machinefile/v1`, [ADR-030](030-init-makes-a-machine-known-apply-carries-its-machinefile.md)) needs items it can name, from layers it does not own: the user's, a shared source, the built-in ones.

## Decision

Two file types, one responsibility each. A catalog item says how to install or set one thing on each system; a Machinefile says what one machine carries, and never defines an item.

- An item is one YAML file, `station.catalog/v1`: a `manager`, an `app` or a `setting` (what it is), and its `methods` (how it gets there): an ordered list, each method a selector in the Machinefile's words (`os`, `family`, `version`, `arch`, and `machine: computer | workspace | server`) and exactly one manager key (`brew`, `cask`, `mas`, `apt`, `npm`, `uv`, `bun`, `mise`, `github-release`, `script`, `skill`, `provided-by`; for a setting `file`, `builtin`, `macos-defaults` and its builtin effects). The file owns the whole item, its disk row included; its category is its folder.
- The built-in items are embedded, `src/catalog/items/<category>/<name>.yaml`, so the binary stays hermetic.
- The decoder is strict: one YAML 1.2 document, every key known, no anchor, alias or merge key, no scalar YAML 1.1 would read otherwise (`yes`, `0755`), every value in its closed list. A file naming a newer schema is refused with `update station`. Its JSON Schema is projected from the Go types by `make generate` into `schemas/station.catalog.v1.json`.
- `wire` loads the catalog once a run and hands the `catalog.Catalog` value to every reader; no reader reaches a package-level list.
- v1 is whole before any file outside the binary uses it: every key the design names (`after`, `phase`, `needs`, `after-apply`, `renamed-from`, `deprecated`, typed `options` read as `${option.<name>}`, `ports`, a manager's `manages` and `priority`, a method's `needs`, `provides`, `upgrade`, `pin`, `path`, a file's `mode`) is declared and decoded, read or not yet, a builtin this station does not carry decodes and is refused at apply with `update station`, so a key added later never breaks an older station with `field not found` instead of `update station`. The format is [The catalog's item format](../reference/catalog.md).
- `provided-by: <item>` is a method: the command comes with another item's install, which is its requirement. `herdr-core` (`hc`) is the first, provided by `herdr-app` on a Mac.

## Consequences

- One file adds an item; the catalog's YAML is read by the word rules like the Go it replaced.
- The user and shared layers (L2.2) decode with the same strict decoder, so a user's typo is a refusal with its line, never a silent default.
- The lists stand by category, in the categories' tables, then by name: a file carries no order, so adding one never renumbers the others.
- A setting carries a method per system like an app; until the settings family applies one effect per system, a setting lives where its first method does. Images and dev-server profiles stay Go values until their own lots move them.
- A port carries what sharing it sets, beside its label: `http`, and `env`, each value a template of `{url}` and `{host}` (known only when the port is shared), so a dev-server profile is an app's own `ports` once a lot reads them, never a table beside it.
- A `script` method has no manager and so no rank: it is always tried after every manager of the system; `github-release` is ranked by the item that manages it.
- Each system's manager priority is declared once, by the manager items' `priority` ranks, never as a list beside them, so a manager added in a file of its own takes its rank without editing another file.
- The words are unambiguous: a selector's machine kind is `machine:` (an item's `kind:` is what it is), an app's version probe is `version-read:` (`version:` is a selector's OS range), a disk action's command is `uses:` (`needs:` is what an item asks of the owner).
