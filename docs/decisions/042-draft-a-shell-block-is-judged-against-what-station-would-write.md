# ADR-042: A shell block is judged against what station would write now

**Status:** Proposed
**Date:** 2026-11-12

## Context

The owner decided that each catalog package may put one identified block into `~/.zshrc`, that a block goes with its package, and that one edited by hand reads `changed` and is left alone. The decision leaves open what `changed` is measured against: `~/.gitconfig`'s `[alias]` block and every other setting are measured against a fingerprint station recorded when it applied them, while a package block belongs to an app, which has no applied record, only a ledger line of the install.

## Decision

- **The measure is the text station would write now.** A block reads `present` when the file holds exactly the rendering of the method's `zsh` with this machine's variables and the app's options, `absent` when its markers are not there, and `changed` for any other text, a start with no end, or an id twice. No new store records a digest; an app's block is judged at each plan, from the file alone.
- **A block of a package the plan carries no longer is removed only while it is still such a rendering** (the options at their defaults), and only for an app the catalog knows; a block of an unknown item or of a setting is left, since a setting's own apply and `off` own its block.
- **A setting writes its own block** through its applier, so `settings apply`, `settings off`, `settings diff` and the recorded fingerprint work as for any setting; an app's blocks are written by `apply`, after the app is there, whichever way it got there.
- **`plan --diff` draws a block as `settings diff` draws every file**: station's lines first (`-`), then the file's (`+`).

## Consequences

- A later station that changes a block's text reads `changed` on every machine that wrote the old one, and apply leaves it: the owner deletes the block (its two marker lines and what is between them) and the next apply writes the new one. Accepted for the small, visible blocks of today; a recorded digest in the ledger is the way out if blocks grow or change often.
- A block written with an option's value reads `changed` once the option's value moves, and is rewritten the same way; a stale block whose app was installed with a non-default option stays, said as `changed`, until it is deleted.
- No second store, no migration of the ledger, and a block can never be overwritten without the owner having deleted it.
