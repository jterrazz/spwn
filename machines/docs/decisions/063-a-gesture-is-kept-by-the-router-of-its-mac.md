# ADR-063: A gesture is kept by the Router of its Mac, with who did it

**Status:** Proposed
**Date:** 2026-11-20

## Context

[ADR-061](061-the-history-is-kept-by-the-mac-that-holds-the-machine.md) keeps a journal on the Mac that holds a machine and reads it, and leaves two things open: what writes a line, and who it says did it. Forty-odd calls could each write their own line, but a store has one writer, its family's calls package, and a family never imports another. A request frame carried no author: a VM's record knew its creator, and nothing knew who restarted or deleted it.

A call reaches a Mac four ways: the CLI runs it in its own process, a frame reaches the stdio server, the window or the menu bar sends it to the engine's socket, or a Router hands it to a VM of this Mac or to another Mac.

## Decision

- **A gesture declares its lines**: `Call.Journal`, a function of its input, its output and the error it ended on, lifted into `Meta.Journal`. The kernel names a line by `core/journal`; `calls.json` marks such a call `journal`.
- **The last hand on this Mac keeps it, once.** The Router keeps a call it runs for this Mac or hands to one of its VMs; the engine's socket server keeps one it runs for this Mac. A Router that hands a call to this Mac's engine or to another Mac keeps nothing, and nothing inside a VM keeps a line. Both hand the lines to a keeper wire builds; `journal/calls.Keep` stamps them on this Mac's clock and writes them.
- **A gesture run inside another keeps nothing.** A gesture's run marks its context, so one it runs in turn (the `apps.update` a `machinefile.upgrade` runs) keeps no line of its own: the outer gesture's lines say what was done, one line per thing the owner did. The rule is one function, `call.Keeper.Keep`, which every keeping hand calls: a Computer only, a call that declares its lines, and no kept gesture around it.
- **A gesture refused before any effect keeps no line**; one that failed or was interrupted is kept failed, its error worded as the CLI words it, secrets withheld.
- **Who is `by`.** A frame gains an optional `by`, a session's name; the CLI sets it from `STATION_SESSION`; the engine's jobs name themselves (`the night`, `station`) in their own process, never on a frame. A name that is no session's is refused as `call.bad-input` before the gesture runs. Absent is `you`.

## Consequences

- A session's gesture says which session did it, wherever it ran, and a deleted VM's line says who deleted it.
- A far binary from before `by` decodes the frame and keeps a session's call as yours: the limit is stated, not hidden.
- The engine's own events (a server restarting, back) and the remaining confirmed gestures are kept by the same keeper as they declare their lines.
- A gesture invoked as a watch is not kept: every gesture today is a call.
- The mark lives in a process: a gesture that sends another to a different Mac's engine leaves that one kept there; `machinefile.apply` and `machinefile.upgrade` reach another Mac only to read.
