# ADR-042: A shell block is judged against what station last wrote

**Status:** Proposed
**Date:** 2026-11-12

## Context

The owner decided that each catalog package may put one identified block into `~/.zshrc`, that a block goes with its package, and that one edited by hand reads `changed` and is left alone. The decision leaves open what `changed` is measured against. Measured against the text station would write now, a new template or an option's moved value reads as a hand edit on every machine, and apply then leaves station's own stale text in place. `~/.gitconfig`'s `[alias]` block and every other setting are measured against a fingerprint station recorded when it applied them; an app's block had no such record.

## Decision

- **The ledger records each block apply writes.** `~/.local/state/station/ledger.json` holds, beside the installs, one record per block: its id, its item, the digest of the text written between its markers, and when. Writing a block replaces its record; removing it deletes it.
- **A block is judged against the file and that record.** `absent` when its markers are not there (apply writes it); `present` when the file holds exactly what apply would write now (kept); `outdated` when it holds exactly what apply last wrote and apply would write otherwise now (written anew where it stands); `changed` for any other text, a start with no end, or an id twice (left as it is). Apply judges the file again just before it writes, so a block edited between the plan and the run is left too.
- **A block goes with its package, under ADR-034's bound.** A block the ledger holds for an extra (an app this Machinefile installed, recorded by station, and lists no longer) is extra too: a bare apply keeps it, as it keeps the app; `apply --remove` takes it back only with the app's own removal (taken back, or already gone), and only while it is still what apply last wrote. An app that stays keeps its block: adopted or the owner's, another Machinefile's, skipped on this system, or kept for the next run to judge again (required, held, unread, failed). An app let go of as the owner's lets go of its block too, and a block edited by hand since station wrote it is the owner's whatever becomes of its app: `apply --remove` drops its record and keeps it in the file. A block the ledger does not hold is never station's to take. An extra whose block could not follow it keeps its own record, so the next run tries again.
- **A setting writes its own block** through its applier, so `settings apply`, `settings off`, `settings diff` and the setting's recorded fingerprint work as for any setting; an app's blocks are written by `apply`, after the app is there, whichever way it got there.
- **`plan --diff` draws what apply would change**: the file's lines first (`-`), then what apply writes (`+`), for a block and for a setting's file alike (`station settings diff` keeps its own way, station's first).

## Consequences

- A later station that changes a block's text, or a Machinefile that moves an option a block names, rewrites the block on the next apply wherever the owner left it untouched, and leaves it wherever he edited it.
- Deleting the ledger makes station take back no block, as it takes back no app; a block then left with no record reads `changed` once station writes it otherwise.
- A block whose package stays for good (adopted, then listed no longer) stays in `~/.zshrc` for good, as the package does; the owner deletes its marker lines to drop it.
- The ledger keeps version 1: `blocks` is a key an older station does not read, and an older station that writes the ledger drops it, which only leaves those blocks unrecorded.
