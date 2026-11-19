# ADR-062: A failed create stays on the record until its start or its delete

**Status:** Proposed
**Date:** 2026-11-19

## Context

A create keeps its VM once the first run is asked, whatever fails after it: a boot the
hypervisor refuses, no address, no SSH. Its end took the birth off the record all the same. So a
VM whose create failed read like any VM stopped by its owner: `machines.list` said `stopped`, no
birth, no last lines, the gesture `start`. Only the create's own error, printed once to whoever
ran it, said that anything had gone wrong. The window showed nothing, and a start ran the VM
without the set-up its create never finished (its first boot), so the VM then read running as if
it had been made.

## Decision

1. **A create that fails from its first run on keeps its birth, failed.** The record keeps the
   birth with `failed`, the step it failed at (an address or SSH failure moves it to `address` or
   `ssh`), the create's `cause` (its error's first line) and the console's `last_lines` (kept as
   `LastLines` keeps a down server's). Its sentence is `Failed at boot 2 of 4`. No process makes it
   any more, so it is deleted or started at once. An interrupted create keeps no failure: an
   interruption is the owner's word, not a fault.
2. **It is a standing of its own, `not-created`**, after `creating` in ADR-054's list, tone bad.
   Its head is `<name> was not created: it failed at <step>.`, its line the cause. It is a finding
   with the same words, broken, its fix its start.
3. **Its gesture is `try-again`, and its start is the create's retry.** On such a VM `machines.start` runs it,
   then does the create's set-up from the first boot on (its first boot, its server's SSH wait, its
   `vm_ready`). The birth goes once that set-up answers, or stays failed at the step that failed
   again. A Machinefile's boot steps are not run again: they are `station apply`'s, and the head's
   reading of the file tells what is missing. The call stays `machines.start`; the word is the
   engine's: the `Gesture` `try-again` (words.json `Try again`) in the header's `Gestures`, and the
   need's `Verb` `try-again`, never `Start`, which says a VM its owner stopped is resumed.
   A start that fails again keeps the new failure (its step, cause and last lines replace the old) and
   answers it as its error, so the window shows it at once.
   `delete` stays out of the gestures, because the page offers it for every VM.
4. **A delete takes the birth off with the record.** The birth lives in the record store, so it
   outlives the engine's restart.

## Consequences

- `machines.list`'s `Standing` gains `not-created` (words.json `not created`, bad), and the birth
  (`Birth`, and `birth` in `machines --json`) gains `failed`, `cause` and `last_lines`. The window
  draws the new standing through words.json.
- The roll-up counts a VM that was not created, by ADR-054's rule at each level: `RollUpState` gains
  `not-created`, bad, right after `down` (`1 machine not created`, N counting the VMs), so a header
  never reads `all running` while one VM shows not created, a VM stopped beside it included. A VM of a Mac that does not answer still
  counts through its Mac.
- A VM blocked by Local Network that later connects keeps its failure until a start, its first boot
  never having run. The create's hint says so, worded by the engine: `approve station on <mac>'s
screen, or in System Settings > Privacy & Security > Local Network; start <vm> once approved`.
- A create that fails after its first run now leaves a broken finding until it is started or
  deleted.
- The CLI's row of such a VM says `Failed at boot 2 of 4`, and the menu bar's Home row reads
  `not created`, `failed at boot`.
