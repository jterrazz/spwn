# The window's icon

`spwn.icon` is the window's icon: an Icon Composer document, the one source of its drawing, from which macOS renders each of its six appearances (default, dark, and clear and tinted, each light and dark).

## What the document holds

The Spwn mark, three dots rising on a diagonal, spans about 70 % of the tile, as Apple's own marks do. Its parts follow Apple's layer rules: each layer is a square 1024 canvas with no tile, mask or shadow of its own, and the system draws those.

- **Two glass groups**: `you`, the big dot (`Assets/big.svg`), and `spawned`, the two smaller ones (`Assets/small.svg`), each with specular light, a neutral shadow and 40 % translucency.
- **The ground** is a gradient: Papier in the light appearance (white to `#e9e9ec`), graphite in the dark one (`#232326` to `#040404`).
- **The dots** are ink (`#0a0a0a`) on Papier, and white in the dark appearance and in the clear and tinted ones, where the system turns them to glass over its own ground.

`icon.json` is edited by hand or in Icon Composer (inside Xcode); `fill` and `fill-specializations` exclude each other, so every colour is a specialization list whose entry with no `appearance` is the default.

## How it ships

`bundle.icon` in `../tauri.conf.json` lists the document, and Tauri's bundler compiles it with Xcode's `actool` (26 or later) into the bundle's `Assets.car`, then writes `CFBundleIconName`. When `actool` is missing or too old, Tauri logs an error and still builds, with a flat icon, so `make window` ends with `scripts/app-icon-check.sh` (at the repository's root), which fails unless the bundle names its icon and its `Assets.car` holds the light, dark and tintable renditions.

## The rasters beside it

`icon.icns` is the fallback `CFBundleIconFile` names, and `icon.png` is the window icon Tauri's codegen embeds, which must be a PNG. Both are `actool`'s output, never drawn by hand, and are made again whenever the document changes, on a Mac with Xcode, in a VM:

1. With `spwn.icon` copied to `<dir>/Spwn.icon` (actool names the icon for the document), `xcrun actool <dir>/Spwn.icon --compile <dir> --app-icon Spwn --include-all-app-icons --target-device mac --minimum-deployment-target 26.0 --platform macosx --output-partial-info-plist <dir>/partial.plist` writes `<dir>/Spwn.icns`, which replaces `icon.icns`.
2. `iconutil -c iconset <dir>/Spwn.icns` unpacks it, and its largest picture, `icon_128x128@2x.png` (256 px: actool packs no larger one), replaces `icon.png`.

## How its look is proved

The Dock capture draws the built bundle where a person sees it, in every appearance, beside Apple's apps: `STATION_CAPTURE_SET=dock specs/machine/menubar/run.sh <label> --capture` ([Test](../../../../docs/guides/test.md)).
