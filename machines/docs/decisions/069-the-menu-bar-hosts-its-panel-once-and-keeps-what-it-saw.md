# ADR-069: The menu bar hosts its panel once, and keeps what it saw

**Status:** Accepted
**Date:** 2026-11-21
**Amended:** 2026-11-21, decisions 4 and 6, accepted under the owner's standing go of 2026-11-21

Amended by [ADR-096](096-spwn-is-one-app-its-menu-bar-notifications-and-engine-live-while-it-runs.md) (Proposed): the panel is hosted in Spwn's own process.

## Context

[ADR-036](036-the-menu-bar-owns-its-glass-panel.md), decision 1, hosts the SwiftUI panel anew on each opening, so that no state of the last opening shows. The menu bar architecture audit of 2026-11-21 traced the flicker of an opening to that choice and to three that ride on it:

- A new hosting view starts every `@State` at zero, so the first size it asks is its scroll view's cap, and the panel has to wait for the second (ADR-036, "Sizing").
- The panel key carries the count of openings, and `.id(key)` tears the scroll tree down at each one.
- Each height the content reports moves the window on the 340 ms spring, the heights that data brought as much as the ones a click asked for (ADR-036 decision 3, "Height").
- Each close drops every page seen, so the next opening of a machine draws Home, then a skeleton, then the page: three trees of three heights.

The audit's decision 3 reverses ADR-036 decision 1. The owner approved it on 2026-11-21, decision 3 of the audit's review.

## Decision

1. **One hosting view for the app's life.** `StatusPanel` creates its `PanelHosting` once, beside the glass, and never removes it. While the panel is hidden, the view keeps laying out and the window follows its size, so an opening orders the panel front at a height that is already known. ADR-036 decision 1 is superseded. Nothing of one opening shows in the next, because the model resets it explicitly: `Navigation.home()` drops the stack, and each `Panel` scrolls back to its top when the count of openings moves (`stationPanelOpening`). The tree is no longer torn down for that.
2. **The panel key names the page, not the opening.** No `Panel` key carries `openings`. `.id(key)` still gives each page and machine its own scroll tree.
3. **Only a gesture moves the height.** A click or key in the panel marks a gesture (`PanelGesture`). Within the next 0.6 s, a new height moves on `Token.Motion.smooth`, both the window (`StatusPanel.fit`) and the content's own spring (`Panel`). Outside that window, a height that data brought is placed at once. The rest of ADR-036 decision 3 still holds: the top edge stays pinned, a size asked mid-placement is held, and Reduce Motion snaps. A host that marks no gesture, as the stories and the catalog do not, moves every height, so no golden moved.
4. **The pages seen are kept across closes, and give way only to a whole page.** `Navigation` keeps the last 8 pages it drew (`Navigation.kept`), the least recent dropped first, across back and across the panel's closings. A page opened again is drawn at once as last seen, in the turn of the gesture that opened it, before the panel's next frame. It gives way only to the first whole page its watch tells (`Page.whole`: a header, no loading state, no value or mark pending, the engine's own rule of a full page in `engine/process/opening.go`). The pages the watch tells while it is still reading are held, and the newest of them is drawn once `Navigation.patience` (0.75 s) passed without a whole one, so a Mac that does not answer still shows what was read. An opening thus draws the page seen, then at most the page read, never a page half read between the two.
5. **An opening asks the engine for nothing.** The app no longer sends `engine.refresh` for Home when it opens; the engine's own cadence keeps the pages fresh.
6. **One panel frame for every page.** What a `Panel` measured (its header, content and footer heights), the height it draws, its scroll position and its edges live in one `PanelFrame`, held by the app's `StorePanel` for its life and shared by Home, every page pushed and Find. A page drawn in another's place starts at the height the panel has and moves from there under decision 3, never from no height; it starts at its top, since the frame remembers the key of the page it last scrolled. A `Panel` with no frame in its environment, as in a story or a test, holds its own.

## Consequences

- The cold first size is gone: the hosting view has already measured its page when the panel opens. The bench of the opening (`specs/machine/menubar/run.sh <label> --capture` with `STATION_CAPTURE_SET=bench`) counts content generations and height changes per opening, before and after.
- A page kept as last seen can be up to its watch's first frame old when it is drawn. The page language has no mark for that age yet.
- Home, the machine pages and the menu bar's own pages each still build their own `Panel` view, since each page view owns the state its header and content share (a Mac's segment). What they share is the frame of decision 6, which holds every state a `Panel` kept, so a push changes the content and not the panel.
- A page drawn as last seen can stay up to `Navigation.patience` past its watch's first frame when the watch reads slowly.
- A page is whole by the engine's marks alone. A part the engine draws as read before it is passes: a Mac's Accounts is drawn `none` until its sign-ins answer (`pages/menubar/accounts.go`, `macAccounts`), so opening a Mac with sign-ins draws the page seen, that page without them, then the page read: three content generations where the rule aims at one.
- The observation split (`PanelStore`: link, home, pages and ui, the live values LiveStore's, each view reading only its slices) comes after this decision, and it changes none of it.
