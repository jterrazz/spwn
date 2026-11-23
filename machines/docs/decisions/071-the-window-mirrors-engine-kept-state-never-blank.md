# ADR-071: The window mirrors engine-kept state, never blank

**Status:** Accepted
**Date:** 2026-11-21

Amended by [ADR-083](083-a-re-ask-is-quiet-only-for-an-engine-kept-view-never-by-a-timer.md): a re-ask is quiet only for a view the engine keeps, released by its next revision or the watch's end; any other is loud, never shows pending over a settled answer, and no timer releases either.

This record decides the client half. The engine half stands on ADR-070, "The engine keeps one read model"; the `state.watch` stream that would carry its kept snapshots to the window is decided later, by its own record.

## Context

The window reads the engine by pull: every read is computed when it is asked, and the read store (`apps/desktop/src/application/use-cases/store.ts`) asks again on a cadence, on focus, on a notice, after a gesture and after a relink. An audit of 2026-11-21 ranked the window's flicker by cause. A revisit drew `Reading` again, because an entry was forgotten 1.5 s after its last subscriber left. A gesture's rereads and a relink drew every partial out over a settled answer. A far Mac that missed one ask replaced its page with the error. An answer equal to the last arrived as a new object and re-rendered every component that held it. A new input drew nothing until it was answered.

The README's design rule already said "a read asked again draws nothing", but only for the cadence's re-asks, and the store had no means to keep an answer past its grace or to tell an equal answer from a new one. The owner rejected replacing the store by TanStack Query: its promise-per-query model fights the engine's progressive watch and the store's hold, and two caches would live side by side during the move.

## Decision

Once a request has an answer, the window never draws less than that answer again. This is a law of the read store, not of each page:

- A new answer is merged into the last by deep-equal reuse (`replaceEqualDeep`): a part that did not move keeps its identity, an answer equal to the last is the last itself, and no listener hears it.
- An error on a re-ask keeps the answer when it says the answer could not be read again (no engine word, `unreachable`, `interrupted`): the reading carries the error as `stale`. A watch whose connection drops keeps its answer unmarked, since the client asks it again on its own. Any other refusal replaces the answer, since the engine said something about the subject itself.
- Every re-ask over a settled answer is quiet, the cadence's, a gesture's rereads and a relink's alike: its outs are held until it ends, or shown at the next tick.
- An entry outlives its last subscriber: its request closes after the grace, its answer is kept for 10 minutes among the 50 kept last, and a subscriber that comes back sees it at once while it is asked again quietly.
- A hook that asks for it shows the previous input's answer, marked `placeholder`, until its own input is answered.

What the stale and placeholder marks look like, and the layout a first read keeps, are the pages' to draw.

## Consequences

- An equal answer costs no render, so the cadence costs the engine its reads and the window nothing.
- A refusal of the cadence's kind is no longer drawn over an answer: a page must draw `stale` for its reader to know the answer is old, and until it does the answer is shown as if fresh.
- A gesture's rereads no longer show a watch's progress over a settled answer; a gesture that wants its rows' progress shows the gesture store's pending state.
- The window holds up to 50 answers it no longer draws, and their requests closed.
- The store keeps its hold logic and its cadence until the engine keeps its state and pushes it; then the store mirrors the stream, and the cadence and the hold are deleted, not flagged.
