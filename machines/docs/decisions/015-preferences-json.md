# ADR-015: station's own choices are preferences.json

**Status:** Proposed
**Date:** 2026-11-06

Written 2026-11-06 from the v3 vocabulary (SPEC §2.1) and the code that carries it.

## Context

station v2 kept its own choices in `~/.config/station/settings.json`. v3 gives "Settings" to the section of what station sets up on a machine (the macOS word since Ventura), so one word would have named two things. The file was also one shape every family read and wrote, so any call could change any key.

## Decision

The file is `~/.config/station/preferences.json`, the contract `core/contract.Preferences`. It is a set of groups, each owned by one family and written only by that family's calls (`name`, `user_email` and `user_name` by setup, `macs` and `vm_user` by machines, `share_list` by accounts, `ports` by ports, `off_here` by settings, `never_sleeps` by awake); the rule `one-writer-per-store` holds it. Every group station knows is listed once in `wire`, in the order the file writes them; a key no group names is dropped at the next write. A group's retired name (`hosts`, `here`, `services`) is read under its new one and written under it at the next write. `station setup` moves a v2 `settings.json` to `preferences.json` once, byte for byte, only while no `preferences.json` is there.

## Consequences

- "Setting" means one thing; "preference" another.
- A family's choices change only through its calls, so a hand edit and a call read the same.
- A group forgotten in `wire`'s list is dropped at its own write: the list is the one place a new group is declared, and `src/wire/stores_test.go` proves a written group survives.
