# ADR-036: The menu bar owns its panel, and the panel's one background is a tinted glass

**Status:** Accepted
**Date:** 2026-11-11

## Context

The owner chose the M1 contrast direction: the system glass stays the panel's frame, dark enough that text holds 4.5:1 over any wallpaper. Commit `459cb381` reached that density with a fill, an 80 % plate under the content, laid over the window `MenuBarExtra(.window)` draws. The window's background is already the system glass, and the owner read the plate as a double background. He asked for a panel that feels native.

The capture of `459cb381` (run `old459`, dark over the dark wallpaper) shows where that reading comes from. On the straight edges the system glass's bright rim (values 95, 83) meets the plate (18) within a pixel or two. At the corners a band of bare glass, 5 to 8 px wide at 2x and lighter than the plate (22, 23), lies between the rim and the plate's corner. The plate's 24 pt corner sits inside the window's glass, so the glass shows as a second, outer surface.

Apple's guidance rules the fill out. Meet Liquid Glass (WWDC25 219) says to avoid glass on glass and to use fills, transparency and vibrancy for the elements on top of the glass. Build an AppKit app with the new design (WWDC25 310) says an `NSGlassEffectView` should hold the content as its `contentView`, never sit behind it as a sibling, and is customised by its corner radius and tint colour. The darkness M1 wants belongs to the glass, as its tint.

`MenuBarExtra` exposes neither its window nor its glass, so the tint cannot be set on it. Three ways were weighed:

- **A. Walk `MenuBarExtra`'s private view tree for its glass and set the tint.** It lost: the tree is undocumented and rebuilt by SwiftUI on any release, and the VM (macOS 26.6.2) and the owner's Mac (27.x) differ exactly there.
- **B. The app owns its status item and an `NSPanel` whose content view is an `NSGlassEffectView` holding the SwiftUI panel.** This is the documented pattern. [ADR-021](021-the-menu-bar-keeps-its-system-window.md) named it as its own fallback, and the app already owns one `NSPanel`, the HUD.
- **C. `.glassEffect(.regular.tint(_:))` on the panel's root, inside `MenuBarExtra`.** It lost: that is a second glass on the system glass, the same double background in a new material.

## Decision

B. It supersedes ADR-021's first decision: the panel no longer keeps `MenuBarExtra`'s window.

1. **One glass, the content inside it.** `StatusPanel` (`apps/menubar/sources/station-menu-bar/StatusPanel.swift`) owns the `NSStatusItem` and a borderless, non-activating `NSPanel`. The panel's content view is one `NSGlassEffectView`, `.regular` (the HIG's variant for text-heavy surfaces), at `Token.Glass.window`'s 24 pt corner, tinted by `Token.Glass.tint` per appearance. The SwiftUI `Popover` is the glass's `contentView`, hosted anew on each opening so `PanelOpening` grows it each time. The live `PanelPane` paints nothing, and the still renders keep their stand-in, so no golden moved. `AppGlassTests` holds the app to that one glass and to no second glass of any kind. The kit stays free of glass (`KitBoundaryTests`).
2. **The tint is 0.65, the plate's colour.** The capture rig's sweep (`run.sh <label> --capture`, run `cap3` on 2026-11-11) set the tint's alpha to 0.5, 0.65, 0.8 and 0.9 over M1's three wallpapers in both schemes. It measured every second-tone text of `home.exposed` over the real glass behind it, by M1's 10th-percentile WCAG method. The lowest alpha that holds 4.5:1 everywhere is 0.65, worst 4.90 in dark over the light busy wallpaper, where 0.5 read 4.42. The red pill reads 5.06 in every shot. Since the tint reaches the target, the fallback, a dimming fill inside the glass's `contentView`, is not built.
3. **The behaviours `MenuBarExtra` gave are the panel's own, and each was measured in the VM (run `cap3`).**
    - **Placement.** The panel hangs 6 pt under the item, its right edge under the item's right edge, clamped 8 pt inside the screen, as the old build's capture read `MenuBarExtra`'s window.
    - **Shadow.** The window casts none, and the glass's own shadow is the panel's only one. The glass draws that shadow past its edge, so a window of the glass's size clipped it to the frame's square corners, and a window shadow then followed that square alpha (run `cap4`, and the owner's capture on 27.x, 2026-11-11). The window is therefore the glass grown by a clear 40 pt margin, `StatusPanel.shade`, where the shadow fades out, and a click in that margin closes the panel like a click outside. Over the three wallpapers in both schemes, the window's outer ring changed by at most 2 of 255 between closed and open, so no square frame shows (run `k2`).
    - **Height.** The hosting view reports only the height its content's spring ends at, never the frames between: the window snapped from 130 to 340 pt in one frame (run `cap1`). So the window moves there itself, animated by AppKit on `Token.Motion.smooth`'s points and 340 ms, and snaps under Reduce Motion. The trial's 240 Hz sampler then read 16 or 17 heights 16 to 17 ms apart (130, 215, 258, 288 … 339, 340 and back). The top edge stayed at 736 pt throughout.
    - **Sizing.** The hosting view yields to the window on both axes. Before it did, Auto Layout grew the window back from its bottom edge while the content shrank. A size the content asks while the window is being placed is held and placed right after, never dropped. The content first asks the cap, its scroll view's height before the page is measured, then its own height from inside the `setFrame` that places the first. Dropped, that second size left the owner's Mac with 700 pt of glass around about 340 pt of content (2026-11-11). On a 3024 x 1964 screen, `home.empty` now logs content 700 then 360, and the glass ends at 360 (run `k2`).
    - **Focus.** The panel is key when shown (`opened key=true` in every shot). The trial's focus walk held every move with Keyboard navigation on and off.
    - **Closing.** A click on the item, a click outside and Escape each close it.
4. **Accessibility is the system's.** With Reduce Transparency on, the glass turns solid. With Increase Contrast on, it draws its contrasting border. Both were seen in the captures with no code of the app's. The owner's Liquid Glass slider acts on the same glass. HYPOTHESIS (2026-11-11): the slider was not exercised, since the VM has no scriptable setting for it; the owner's look on 27.x settles it.

## Consequences

- The app now carries what `MenuBarExtra` gave for free, about 200 lines in `StatusPanel.swift`. Spaces and full screen ride on the panel's collection behaviour, which no run exercised.
- The double background of `459cb381` cannot return: nothing paints over the glass, and `AppGlassTests` refuses a second glass or content beside it.
- The VM renders without a GPU on macOS 26.6.2, so the captures prove the structure and the numbers in range, not the owner's pixels on 27.x. His look after `make menubar-install` is the last check.
- `STATION_TRIAL_TINT` lives with the trial and goes with it.
