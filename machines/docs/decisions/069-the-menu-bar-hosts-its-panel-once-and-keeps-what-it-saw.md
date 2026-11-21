# ADR-069: The menu bar hosts its panel once, and keeps what it saw

**Status:** Proposed
**Date:** 2026-11-21

## Context

[ADR-036](036-the-menu-bar-owns-its-glass-panel.md), decision 1, hosts the SwiftUI panel anew on each opening, so that no state of the last opening shows. The menu bar architecture audit of 2026-11-21 traced the flicker of an opening to that choice and to three that ride on it:

- A new hosting view starts every `@State` at zero, so the first size it asks is its scroll view's cap, and the panel has to wait for the second (ADR-036, "Sizing").
- The panel key carries the count of openings, and `.id(key)` tears the scroll tree down at each one.
- Each height the content reports moves the window on the 340 ms spring, the heights that data brought as much as the ones a click asked for (ADR-036 decision 3, "Height").
- Each close drops every page seen, so the next opening of a machine draws Home, then a skeleton, then the page: three trees of three heights.

The audit's decision 3 reverses ADR-036 decision 1. The owner approved it in the audit's review.

## Decision

1. **One hosting view for the app's life.** `StatusPanel` creates its `PanelHosting` once, beside the glass, and never removes it. While the panel is hidden, the view keeps laying out and the window follows its size, so an opening orders the panel front at a height that is already known. ADR-036 decision 1 is superseded. Nothing of one opening shows in the next, because the model resets it explicitly: `Navigation.home()` drops the stack, and each `Panel` scrolls back to its top when the count of openings moves (`stationPanelOpening`). The tree is no longer torn down for that.
2. **The panel key names the page, not the opening.** No `Panel` key carries `openings`. `.id(key)` still gives each page and machine its own scroll tree.
3. **Only a gesture moves the height.** A click or key in the panel marks a gesture (`PanelGesture`). Within the next 0.6 s, a new height moves on `Token.Motion.smooth`, both the window (`StatusPanel.fit`) and the content's own spring (`Panel`). Outside that window, a height that data brought is placed at once. The rest of ADR-036 decision 3 still holds: the top edge stays pinned, a size asked mid-placement is held, and Reduce Motion snaps. A host that marks no gesture, as the stories and the catalog do not, moves every height, so no golden moved.
4. **The pages seen are kept across closes.** `Navigation` keeps the last 8 pages its watches told (`Navigation.kept`), the least recent dropped first, across back and across the panel's closings. A page opened again is drawn at once as last seen, and its watch's first frame then replaces it.
5. **An opening asks the engine for nothing.** The app no longer sends `engine.refresh` for Home when it opens; the engine's own cadence keeps the pages fresh.

## Consequences

- The cold first size is gone: the hosting view has already measured its page when the panel opens. The bench of the opening (`specs/machine/menubar/run.sh <label> --capture` with `STATION_CAPTURE_SET=bench`) counts content generations and height changes per opening, before and after.
- A page kept as last seen can be up to its watch's first frame old when it is drawn. The page language has no mark for that age yet.
- One `Panel` container for every page, the audit's other half of this lot, is not built. Home, the machine pages and the menu bar's own pages are still separate `Panel` trees, so a push still changes trees. The cache makes it one change, not two.
- The observation split (`@Observable` link, home, pages, live, ui) comes after this decision, and it changes none of it.
