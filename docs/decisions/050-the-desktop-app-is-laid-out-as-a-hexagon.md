# ADR-050: The desktop app is laid out as a hexagon

**Status:** Accepted
**Date:** 2026-11-15

## Context

The TypeScript member that draws the desktop window was named `apps/window`, after the one thing it shows, while its siblings are named for their surface: `apps/menubar` and `apps/engine`. Its `src/` grew by the kind of file (`engine`, `design`, `shell`, `pages`, `model`), a layout the dependency rules read only from the member's own folder.

## Decision

The owner decided on 2026-11-15 ("parfait ton idée", after seeing the tree).

1. **The member is named for its surface**: `apps/desktop`, beside `apps/menubar` and `apps/engine`. Its npm package is `station-desktop`.
2. **Its `src/` is laid out as a hexagon**: `domain`, `application/{use-cases,ports}`, `infrastructure/outbound`, `composition` and `presentation`. The engine's families are its domains.
3. **Imports point inward**, and `composition/compose.ts` is the only importer of infrastructure.
4. **The move is behaviour-neutral** and lands in several commits after the rename.
5. **The Cargo crate name `station-window` and the Tauri identifier stay**: they name the installed app, and changing them would change what is installed.
