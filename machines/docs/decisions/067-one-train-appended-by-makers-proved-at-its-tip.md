# ADR-067: One train, appended by makers, proved at its tip

**Status:** Accepted
**Date:** 2026-11-21

Accepted by the owner on 2026-11-20.

## Context

Proved lots reached `main` through three serial stages, each a third party cherry-picking 23 to 61 commits of other makers onto the last. Re-ordering other people's commits broke their recordings and forced a regeneration per commit: 36 of them, 266 min. Each restack moved the shas under every session that followed the stack, three times in one day.

## Decision

- **One integration branch, the train.** A lot is cut from the train's last proved tip. When it is proved, its own maker rebases it onto the train's current tip and appends it: the one who wrote the code resolves its conflicts, in arrival order, and nobody re-orders anybody's commits.
- **The train only grows.** A defect found after an append is fixed by a new commit at the tip, never by a fold into the middle; a fold belongs inside a lot, before its append.
- **The tip is proved by the full gate ([ADR-066](066-a-commit-owes-that-it-builds-and-lints-a-lot-owes-the-gate.md)) after each append**, or after each batch of appends when several wait, and green moves one forward-only ref, the proved tip. A red tip is bisected over the lots appended since the last green one, and the lot's maker fixes it at the tip.
- **One cold gate from a fresh clone before a release.**
- After a publication the train is local `main`. It is the estate's batch branch while a publication is pending ("A chunk is proved warm, a batch cold"), and nothing once the publication is out, so the estate's rule of no develop branch holds.

## Consequences

- No stacker role: makers carry the rebase recipe, and the train's tip is proved by one integrator.
- Other sessions follow one ref that only moves forward and never rebase on an interim tip.
- A lot that rebases over an engine change records the window's worlds once, at its tip.
