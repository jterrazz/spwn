# ADR-037: The TUI is retired; the CLI is the one terminal surface

**Status:** Proposed
**Date:** 2026-11-11

## Context

`station` on a terminal opened a Bubble Tea app: a shell of three panes, one screen per section, its own keys, palette, forms and progress strip. It read every page by running this same binary with `--page`, so a page and a gesture each had one function, and it cost about 6,000 lines of surface, 1,900 of renderer, a layer of the import table, four architecture rules, the `specs/tui/` documents and two more render targets for every page fixture. No ADR introduced it: [ADR-001](001-station-mirrors-the-studio-cli-layers.md) mentions it in passing, as a renderer of the commands' presenters.

The owner decided on 2026-11-11 to remove it. The command line carries every feature, and the menu bar is the graphical surface; a real GUI may come later on the same calls.

## Decision

The TUI is retired, and the CLI is the one terminal surface. Bare `station` prints `station overview`, with its `--all` and `--json`, on every stream: this machine at a glance, then the page's own next steps as commands, on the machine `--on` or `station use` names.

What only the TUI used goes with it: `--values-on-stdin` (a setting that asks values asks them on its terminal), activity's hidden `--graph` (the menu bar asks the engine's `activity.read` for its window), All machines' Activity section of thermal state and battery, `pages/machines.Table`, a hint's `key`, the TUI render targets and their goldens. What the menu bar and the specs still read stays: the hidden `--page`, a page's actions and their keys, `activity.read`'s graph window.

The Charm stack stays for the CLI's own forms and help: `huh` asks, and runs on Bubble Tea.

## Consequences

- One terminal surface, one function per page and per gesture, and a smaller tree for the architecture test to hold.
- A terminal no longer offers keys over the machines; what the TUI did on a key is a command, and a GUI that wants it calls the same call.
- The records of [ADR-001](001-station-mirrors-the-studio-cli-layers.md), [ADR-016](016-api-v2-and-remote-by-api.md) and [ADR-019](019-activity.md) still name the TUI as it was when they were taken; they are history and stay as written.
- A page's action keys (`a`, `i`, `enter`) were the TUI's bindings; the menu bar's shortcuts share their field, so they stay until that field is split.
