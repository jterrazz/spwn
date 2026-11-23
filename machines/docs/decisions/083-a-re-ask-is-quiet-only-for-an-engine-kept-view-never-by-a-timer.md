# ADR-083: A re-ask is quiet only for an engine-kept view, and never by a timer

**Status:** Accepted
**Date:** 2026-11-23

Accepted by the owner on 2026-11-23, on the owner's word.

Amends [ADR-071](071-the-window-mirrors-engine-kept-state-never-blank.md).

## Context

ADR-071 made every re-ask over a settled answer quiet: its outs were held until the watch ended, "or shown at the next tick". That tick was the read store's 30 s cadence, `FRESH_EVERY`, and the timer was what released a held out the watch never followed with an end. RM7 deletes the cadence: a read is now asked again on a notice of `engine.events`, a gesture's rereads, a relink or a restart, never by a clock. ADR-071's sentence then named a release that no longer exists.

The engine now pushes its kept views to the window on the `engine.state` stream (ADR-070), so a view has a second release beside its watch's end: the engine's next revision of it. A read the engine does not keep has no revision behind it. A first fix showed such a read's outs as they came. The non-view watches of a machine's Ports (`activity.read`, `apps.list`, `engine.watch`, `settings.list`) then made React commit 4 times on a notice that changed nothing, and a watch whose first out is pending could draw it over the settled answer. The owner's rule is that a page once drawn is never blank and that the assert proving it stays exact: zero commits on an equal re-ask.

## Decision

ADR-071's re-ask sentence is replaced by this one. Its other sentences stand.

- A re-ask is quiet only for a view the engine keeps. Its outs are held, the settled answer shown, until the engine's next revision of that view or the watch's end, whichever comes first.
- Any other re-ask is loud: over an answer still filling, its outs are shown as they come. A loud re-ask never shows pending over a settled answer. A watch's every out before its end is a fill in progress, so over a settled answer the loud re-ask's outs are held, and the settled answer stays until the watch ends with the new one.
- No timer releases a held out, and none asks a read again.

## Consequences

- An equal re-ask costs no render, whatever its read: its answer, merged by `replaceEqualDeep`, is the reading on screen.
- A loud re-ask over a settled answer shows nothing until its watch ends. A watch that never ends after a re-ask would keep the old answer shown, unmarked; every watch the window re-asks has ended once before.
- A watch reconnected by the client is held from its new first out only when it is a quiet view; a loud one over an answer still filling shows its outs as they come.
