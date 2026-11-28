# ADR-110: A fragment is merged into a file the owner keeps, and taken back by its record

**Status:** Proposed
**Date:** 2026-11-28

A setting's `file` method could write a file whole or, declared but unread, as a block. A setting that must add a hook or a few values to a file another app owns (an editor's `settings.json`, an agent's hooks) had no way to do it without taking the whole file. This record decides how Machines writes a fragment into such a file, what it keeps of the owner's, and how `settings off` takes the fragment back. The rules themselves are [Dotfiles](../reference/dotfiles.md#fragments)'s.

## Context

ADR-052's failure was a hook written into an agent's settings whose command was not installed: every run of the agent then failed on it. A fragment lands in a file the owner also edits, so off cannot compare the file with Machines' copy, as a whole file's does: it has to know which values are Machines', and whether he changed them since.

## Decision

Two modes beside `file`: `block`, the package block's marker scheme in any text file, and `merge`, a JSON fragment merged into a `.json` file.

### How a merge merges

Objects are merged key by key; into an array an element is appended only when no deep-equal element is there; a value of the owner's is never replaced, only left and said. The target is read as strict JSON and nothing else: a comment, a trailing comma, a key twice, no object at its top refuse the file by name and it is never rewritten. JSONC is not accepted, since writing it back without its comments would destroy the owner's text. Bounds: a target of 1 MiB, a fragment of 256 KiB, a depth of 32.

### What the encoder keeps

The tree is held in the file's own order with each key's and scalar's raw bytes, and written back in `JSON.stringify` layout with the file's indent unit. A file already in that layout round-trips byte for byte; any other whitespace is not kept. An output equal to the input is not written.

### The record is the adapter's own

What a merge wrote (per setting and file: each JSON pointer and the digest of its canonical value, a block's digest, the file and folders Machines made) is kept in `~/.local/state/station/merged.json`, beside the ledger and not in it. The ledger is the Machinefile family's store, and one store has one owner; the authorized-keys record set that precedent. Off removes a value only while its digest is the recorded one, and a container only when it is empty and Machines made it.

### The guard is `requires`

No new key: the package's existing `requires` is the guard. A setting in either fragment mode is written only where an app it requires is installed; a named apply elsewhere is refused with a sentence naming the app. A Machinefile installs the required app first, and a bare apply skips the setting, as before.

## Consequences

- Deleting `merged.json` makes off take back nothing: Machines' values stay as the owner's.
- An element of an array the owner edited cannot be told from one he deleted, so off leaves it without a word.
- A block goes only into a file whose comments open with `#`.
- The guard checks the app is installed, not that the hook's command is on the PATH of the app that runs it.
