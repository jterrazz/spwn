# The window's icon

`spwn.icon` is the window's icon: an Icon Composer document, the one source of its drawing, from which macOS renders each of its six appearances (default, dark, and clear and tinted, each light and dark).

## What the document holds

La pile: three floors growing with depth, the smallest on top with one ball resting on it, on a paper tile. Its parts follow Apple's layer rules: each layer is a square 1024 canvas with no tile, mask or shadow of its own, and the system draws those. The owner picked it on 2026-11-22: round 5's variant F, "Trois étages, ambre", in the colour appearances, and round 6's mix 2 in the clear and tinted ones.

- **Four glass groups**: `you`, the ball, then one group a floor from the top down, `f0`, `f1` and `f2`, each a top face and a side face, with specular light and no translucency; `f0` also holds the ball's contact shadow.
- **In colour**, light and dark: an amber ball on a white top floor, then a light amber floor and a deep amber one.
- **In clear and tinted**, light and dark: the top floor greyed under a solid white ball, so the ball stands apart from the floor it rests on, and the two floors below are white faces over empty walls. Each layer carries a default, a `-dark` and a `-tinted` picture, and the system turns the tinted one to clear or tinted glass over its own ground.
- **The ground** is a gradient: Papier in the light appearance (white to `#e4e6ec`), graphite in the dark one (`#38353d` to `#0a0a0c`).

`icon.json` is edited by hand or in Icon Composer (inside Xcode); `fill` and `fill-specializations` exclude each other, so every colour is a specialization list whose entry with no `appearance` is the default.

## How it ships

`make window` compiles the document before it bundles: `scripts/app-icon-compile.sh` (at the repository's root, the same script Spwn Menu's build calls) runs Xcode's `actool` (26 or later) on it as `Spwn.icon`, into `.artifacts/window/icon/`, and fails unless `Assets.car`, `Spwn.icns` and the partial plist naming the icon `Spwn` are all there.

`bundle.icon` in `../tauri.conf.json` lists those two outputs: Tauri copies the `Assets.car` into the bundle and writes `CFBundleIconName` from the icon it holds, and `Spwn.icns` becomes the fallback `CFBundleIconFile`. Tauri is never given the document itself: its own `actool` call fails inside its process with "Bad file descriptor", where the same call compiles from a shell. `make window` then ends with `scripts/app-icon-check.sh`, which fails unless the bundle names its icon and its `Assets.car` holds the light, dark and tintable renditions.

## The raster beside it

`icon.png` is the window icon Tauri's codegen embeds, which must be a PNG. It is `actool`'s output, never drawn by hand, and is made again whenever the document changes, on a Mac with Xcode, in a VM: `iconutil -c iconset .artifacts/window/icon/Spwn.icns` unpacks the compiled icon after `make window`, and its largest picture, `icon_128x128@2x.png` (256 px: actool packs no larger one), replaces `icon.png`.

## How its look is proved

The Dock capture draws the built bundle where a person sees it, in every appearance, beside Apple's apps: `STATION_CAPTURE_SET=dock specs/machine/menubar/run.sh <label> --capture` ([Test](../../../../docs/guides/test.md)).
