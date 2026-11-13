# ADR-047: A catalog entry is a package, and a machine file is `<name>.machine.yaml`

**Status:** Accepted
**Date:** 2026-11-13

## Context

A catalog entry has been called an item since the catalog existed ([ADR-031](031-the-catalog-defines-the-machinefile-chooses.md)): `item: station.catalog/v1` at the head of each file, `src/catalog/items/<category>/<name>.yaml`, "item" in the code, the glossary and the pages. The word says nothing a reader can use: a menu item, a login item and a status item are items too. The Machinefiles have no mark of their own in the name: `station-macos.yaml` sits beside a package file in nothing but its folder, and the owner's own machines have no home. The design round of 2026-11-12 (section B) settled the names.

## Decision

- **"Item" becomes "package", everywhere**: the code, the docs, the glossary and the schema. A package is one YAML file, `package: station.package/v1`, in `catalog/packages/<category>/<name>.yaml`, and the schema file is `station.package.v1.json`. Where "item" means a menu item, a login item or a status item, it stays.
- **A machine file is `<name>.machine.yaml`.** The built-in ones are `station-linux.machine.yaml` and `station-macos.machine.yaml`; the owner's overlay and lock keep their names beside it.
- **One catalog folder, in studio-station.** `catalog/packages/<category>/<name>.yaml` holds the packages and `catalog/machines/<name>.machine.yaml` the machines, the owner's own included. Extracting it to a repository of its own, a store, is a TODO for one day.
- **The cache and the state stay apart.** The cache of the shared source is a copy; `~/.local/state/station` is station's memory. Neither is the catalog.
- **No alias.** The old header key, the old schema name and the old file names are not read: nothing keeps a reader for them, and the owner's files are renamed once by hand.

## Consequences

- A lot renames the code, the schema and its generator, the catalog's 149 files and the two built-in machine files; the ledger's keys, which name an install by `item`, go with it.
- The pages that say "item" in this sense (the architecture, the glossary, the catalog reference, the develop guide's table) are rewritten in the lot that renames the code.
- ADR-031 and later records keep their words: a record's body is written once, and the rename is this one's.
- The format's name (`station.package/v1`) follows the owner's "schema" of section B; the header key is the one point the owner may still choose differently.
