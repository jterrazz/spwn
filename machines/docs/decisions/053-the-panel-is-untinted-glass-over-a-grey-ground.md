# ADR-053: The panel is the untinted system glass over a hue-free grey, with no white card

**Status:** Accepted
**Date:** 2026-11-17

## Context

The owner read the menu bar panel as washed white-bluish, and asked for a rendering as natural as macOS itself (2026-11-17). Two things of ours sat on the system's glass. The glass carried a tint of its own, white at 65 % in light and near-black at 65 % in dark ([ADR-036](036-the-menu-bar-owns-its-glass-panel.md), Decision 2), chosen so that second-tone text held 4.5:1 over any wallpaper. The boxes on it were white cards: a tile white at 58 % in light, a raised button at 80 %, a card at 30 %.

A capture of macOS 26.6.2 in a probe VM measured the Passwords menu bar extra and the Apple menu as one material, within 7 of 255 per channel in both appearances, and the Passwords panel at 322 pt wide with an 18 pt corner. The capture rig's wash measure (`STATION_CAPTURE_SET=native`, [the test guide](../guides/test.md), "The wash and the zoom") compares the panel's glyph-free median colour with the Apple menu's over the same desk, six desks in all: each appearance over a dark, a light and a colour wall.

No system material alone matched the menu. In one session over the six desks (sweep `mb1-sweep3`), the untinted glass, `NSVisualEffectView`'s `.menu` and `.popover` each passed one desk of six at best: they let far more of the desk through than the menu, which on Tahoe is near opaque.

The measure also found a conflict in its own bars. On the two light walls the Apple menu itself carries a blue cast, b\* -2.4 in dark and -3.0 in light (sweep `mb1-sweep4`). A panel with no hue of its own then reads 2.3 to 2.4 on b\* there, and fails a 2.0 bar whatever its alpha. The tinted glass came closest to the menu by summed distance, yet it is the one the owner disliked: over the dark wall in light it adds -4.6 on b\* to the menu's own cast, which is the bluish look.

So the numbers could not pick. Sweep `mb1-sweep4` put the candidates on one board, a crop of each beside the Apple menu over each desk with its numbers under it, and the owner chose by eye: "verre + fond 0,6", the glass over a ground at 0.6.

## Decision

1. **The panel's background is the system glass with no tint, over a hue-free grey at 0.6.** `StatusPanel`'s one background is an `NSGlassEffectView`, `.regular`, with no tint, at the 18 pt corner. Its `contentView` is a ground that paints `Token.Glass.ground`, the system's window colour (`NSColor.windowBackgroundColor`) at 0.6, which has no hue in either appearance, and the SwiftUI content lies on it. The window casts no shadow of its own, as in ADR-036. The glass sizes its `contentView` by constraints, so the content tells its size to the panel on the next turn of the run loop, never from inside its own layout pass: a size told from there dirtied the glass's constraints mid-update, and AppKit killed the app (sweeps `mb1-sweep` and `mb1-sweep2`). `Token.Glass.tint`, the trial's `STATION_TRIAL_TINT` and the rig's tint sweep are deleted. `AppGlassTests` holds the app to that one untinted glass and its ground, and the ground to a grey (r = g = b in sRGB) at 0.6 in each appearance.
2. **Boxes and labels take the system's quiet fills and label colours**, as the round-5 spec draws them. A box is one step off the material, white at 26 % in light and 5.5 % in dark, edged by a 6 or 7.5 % line. A button, a field and a small control take the system's dark fill in light, a hover the system's inset highlight. Labels are black or white at 85 %, the second tone at 58 % in light and 55 % in dark, the third at 25 %, hairlines at 10 %. `TokenTests` refuses a white card: a box past those bounds, or any other ground in light that is white at all.
3. **The panel is 322 pt wide with an 18 pt corner**, the Passwords extra's.
4. **The still stand-in follows the live panel.** A golden cannot sample the glass, so `PanelPane` draws a plate in its place: the system menu's grey, 244 at 64 % in light and 32 at 60 % in dark, with an even inner rim and no top light. The Catalog's flat desk is a neutral grey.
5. **The wash's bars are set on the owner's pick.** The panel passes within 4 CIEDE2000 and 3.5 on b\* of the Apple menu, on every desk: one step above the pick's worst desk, 3.69 and 3.28, both dark over the colour wall. The tinted glass fails them light over the dark wall. Passwords' panel against the Apple menu (2.7 and 2.8, b\* 0.7 and 0.3) and two openings of the menu (0) stay under both.

Sweep `mb1-sweep4`, one session, CIEDE2000 and b\* against the Apple menu, with the bars of Decision 5:

| desk (scheme, wall) | tinted glass     | the pick: glass over the grey at 0.6 |
| ------------------- | ---------------- | ------------------------------------ |
| dark, dark          | 1.68, -1.32      | 2.11, 0.67                           |
| dark, light         | 1.34, 0.48       | 3.06, 2.29                           |
| dark, colour        | 1.73, 2.87       | 3.69, 3.28                           |
| light, dark         | 7.26, -4.55 FAIL | 1.61, 0.84                           |
| light, light        | 1.29, 1.20       | 2.45, 2.38                           |
| light, colour       | 3.43, -2.88      | 1.73, 1.22                           |
| summed CIEDE2000    | 16.73            | 14.65                                |

The capture of the lot's code at `4f33ca0c1` (run `mb1-cap7`) passes the six desks: 2.04, 0.67; 3.15, 2.15; 3.30, 2.76; 1.61, 0.84; 2.36, 2.27; 1.18, 0.70, in the table's order. The tinted glass put back on the same code (run `mb1-mutcap`) fails light over the dark wall, 7.30 and -4.55.

It supersedes ADR-036's Decisions 1 and 2. The glass stays the panel's one glass, and its content stays inside its `contentView`, but the glass carries no tint, its corner is 18 pt, and the content lies on a grey ground there, the dimming fill ADR-036 named as its fallback. The rest of ADR-036 stands: the app owns its status item and its panel, and the behaviours the panel owns (Decision 3).

## Consequences

- Second-tone text, by M1's 10th-percentile method, reads 4.40 to 5.83 (run `mb1-cap7`), against 4.85 to 6.12 with the tint: 4.40 in dark over the light busy wallpaper is under ADR-036's 4.5:1, which the tint was set to hold. The board the owner chose on showed no contrast figure, so this cost is open with the pick. Reduce Transparency and Increase Contrast act on the glass as on any system glass, with no code of ours.
- The bars no longer ask the panel to match the menu's own blue cast on the light walls; they refuse a cast added on top of it, the owner's complaint.
- The panel's own colour moves between runs: the tinted build's dark wall read 4.23 in one run and 1.33 in the next. A margin of 0.31 and 0.22 above the pick is one run's evidence, and a red wash on the pick is read against a second run before the bars move.
- Every golden and `heights.json` were re-recorded at 322 pt; no story was added or removed.
- The opening's zoom, the other half of the owner's complaint, is not touched here: [ADR-036](036-the-menu-bar-owns-its-glass-panel.md) orders the panel in whole, nothing scaled or faded.
