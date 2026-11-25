# The window's icon

`spwn.icon` is the window's icon: an Icon Composer document, the one source of its drawing, from which macOS renders each of its six appearances (default, dark, and clear and tinted, each light and dark). `sizes/` holds the same icon drawn at each pixel size, for the `.icns` beside it.

## What the document holds

The overlapping dots: three dots one behind the other, shrinking away from the big one, the front one orange, flat, on a dark ground. The owner picked it on 2026-11-25: D, "Le verre", flat on a dark ground, in its setting 3, "Toi devant". Its parts follow Apple's layer rules: each layer is a square 1024 canvas with no tile, mask or shadow of its own, and the system draws those.

- **Three groups**, one layer each, an SVG of one filled shape: `front`, `middle` and `back`. Each dot behind is a crescent: its circle with the dot in front of it, grown by a ring of ground, already cut out. No group has glass, specular light, shadow or translucency, so the drawing stays flat.
- **In light**, a dark tile (`#26262A` to `#0C0C0E`) under an orange front dot (`#FF9F0A`), then `#E9E9EE` and `#8B8E98`.
- **In dark**, a darker tile (`#1C1C20` to `#050506`) under the same orange, then `#C9CAD1` and `#6D707A`.
- **In clear and tinted**, the dots are white, `#E2E3E8` and `#B9BBC4`, and the system turns them to clear or tinted glass over its own ground.

`icon.json` is edited by hand or in Icon Composer (inside Xcode); `fill` and `fill-specializations` exclude each other, so every colour is a specialization list whose entry with no `appearance` is the default.

## The drawn sizes

`sizes/spwn-icon-<px>.svg` is the whole icon, tile included, at 16, 32, 64, 128, 256, 512 and 1024 px. The 32 and 16 px ones are fitted by hand: their dots overlap less and land on whole pixels, so each crescent stays at least 2 px thick, where a scaled-down drawing turns to mush. The drawing and its numbers come from the brand's logo work of 2026-11-25.

## How it ships

`make window` compiles the document before it bundles: `scripts/app-icon-compile.sh` (at the repository's root, the same script Spwn Menu's build calls) runs Xcode's `actool` (26 or later) on it as `Spwn.icon`, into `.artifacts/window/icon/`, and fails unless `Assets.car`, `Spwn.icns` and the partial plist naming the icon `Spwn` are all there. Given `sizes/`, it then draws each picture of an iconset from the size of the same pixel count (`scripts/app-icon-iconset.swift`, never a larger one scaled down) and packs it with `iconutil` into `Spwn.icns`, in place of actool's renders.

`bundle.icon` in `../tauri.conf.json` lists those two outputs: Tauri copies the `Assets.car` into the bundle and writes `CFBundleIconName` from the icon it holds, and `Spwn.icns` becomes the fallback `CFBundleIconFile`. Tauri is never given the document itself: its own `actool` call fails inside its process with "Bad file descriptor", where the same call compiles from a shell. `make window` then ends with `scripts/app-icon-check.sh`, which fails unless the bundle names its icon and its `Assets.car` holds the light, dark and tintable renditions.

## The raster beside it

`icon.png` is the window icon Tauri's codegen embeds, which must be a PNG. It is never drawn by hand: it is the 256 px picture of the iconset, `icon_128x128@2x.png`, drawn from `sizes/spwn-icon-256.svg`, and is made again whenever the sizes change, on a Mac, in a VM: `swift scripts/app-icon-iconset.swift machines/apps/desktop/src-tauri/icons/sizes <out>.iconset`, then that picture replaces `icon.png`.

## How its look is proved

The Dock capture draws the built bundle where a person sees it, in every appearance, beside Apple's apps: `STATION_CAPTURE_SET=dock specs/machine/menubar/run.sh <label> --capture` ([Test](../../../../docs/guides/test.md)).
