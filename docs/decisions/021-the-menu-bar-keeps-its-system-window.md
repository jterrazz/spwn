# ADR-021: The menu bar keeps its system window, its own focus walk, and a still status item

**Status:** Proposed
**Date:** 2026-11-07

## Context

The v4 panel (SPEC §A3, §A7, M16, M29) rests on three behaviours of macOS 26 that no golden can prove, named as risks in SPEC §D2 and probed before the Panel template is built on them (lot L06):

1. Whether a `MenuBarExtra(.window)` window follows its content's animated `.frame(height:)` frame by frame, or snaps. If it snaps, the app must own a borderless `NSPanel` under the status item.
2. Whether a programmatic `@FocusState` walk over parts marked `.focusable(interactions: .edit)` holds while the system's Keyboard navigation setting is off. If it does not, the highlight becomes plain `PanelState`.
3. What it costs to redraw the status item's label (an `NSImage` rendered offscreen at 1x, 2x and 3x) 30 times a second, for the badge's width (M29) and the busy mark (M16). If the cost is visible, the badge snaps and the busy mark turns at 10 fps.

The trial is `TrialPanel` and `TrialItemView` (`template/Trial.swift`). It runs in `Station.app` when `STATION_TRIAL=1` is set, and in the Catalog's canvas under `--trial`. The panel is 3 or 9 rows of `Token.Metric.Row.height`, measured with `onGeometryChange` and shown at a height animated on `Token.Motion.smooth` (SPEC §A3's height law). Tab and Back-tab walk the rows' focus, Return and Space toggle the height, and `STATION_TRIAL_CYCLE` toggles it on a timer. The app samples the panel window's frame at 240 Hz while it is open. It redraws the label at `STATION_TRIAL_FPS` and reports its own CPU from `getrusage`. The trial ran on 2026-11-07 in the `station-play` macOS 26.6.2 VM (play/RIG.md), driven over VNC, with no engine. The logs and captures are in `station-design/play/round-v4/probe/`.

## Decision

1. **Height: kept.** The system window follows the spring. In every toggle, the window's frame moved through 17 or 18 heights 16 to 17 ms apart, one per display frame, from 130 to 340 pt and back over the 340 ms curve (130, 155, 185, 223, 265, 292, 306, 316 … 339, 340). Its top edge stayed fixed under the menu bar (`y + height` = 736 throughout), so the panel grows downward. D2 builds on `MenuBarExtra(.window)` and adds no `NSPanel`. The prototype's radius 18, its whole-panel scale and its close to .98 stay the system's (SPEC §D2, second risk).
2. **Focus: kept, with one rule.** With Keyboard navigation on (`AppleKeyboardUIMode` 3) and off (0, `isFullKeyboardAccessEnabled` false), every programmatic move held: the `@FocusState` read back the row it was set to on the next turn of the run loop, every time, and the kit's ring followed it (captures `focus-on.png` and `focus-off.png`). In the Catalog's window, with the setting off, the ring landed on the row the walk named (`catalog.png`). The rule: Shift-Tab does not arrive at `.onKeyPress(.tab)` with Shift held. It arrives as the back-tab character (`U+0019`). Before the trial listened for it, Shift-Tab went to the system's own focus loop, which moved the focus without the walk knowing. S7 handles `.tab` and back-tab both, and does not rely on reading `.shift` from a Tab press.
3. **Status item motion: fallback.** Measured over 30 to 40 s at each rate, panel closed (the app's own `getrusage` share, and `top` averages for the menu bar's processes):

    | label redraw | StationMenuBar | WindowServer | ControlCenter |
    | ------------ | -------------- | ------------ | ------------- |
    | still        | 0 %            | 0 %          | 0 %           |
    | 10 fps       | 6 %            | 16 %         | 2 %           |
    | 30 fps       | 18 to 24 %     | 18 %         | 4 %           |

    At 30 fps the app also missed frames (about 28 drawn a second). The VM renders without a GPU, so a Mac will cost less, but a quarter of a core plus the WindowServer is visible on any Mac for an item that sits in the menu bar all day. The badge snaps to its new width with no animation (M29). The busy mark turns at 10 fps, and only while something is busy (M16). No label timer runs while the item is still.

## Consequences

- D2's Panel template keeps the system window, its glass, its corner (`Token.Glass.window`) and its instant close. The 4 h fallback of PLAN §4 is not spent.
- S7 builds the focus trap of SPEC §A7 as specified, plus the back-tab rule. The `PanelState` highlight fallback is not needed.
- The status item's stories draw the snapped badge (M29). The item's busy mark is drawn at the angle of a 10 fps frame (M16), and the `Model` runs its timer only while a call runs.
- The trial stays in the tree behind its flags until D2 and S7 replace it, so the record can be checked again on a new macOS.
