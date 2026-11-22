# ADR-065: The window updates itself through the Tauri updater

**Status:** Superseded by [ADR-076](076-spwn-on-a-machine-one-active-app-one-background-job-the-engine-updates.md)
**Date:** 2026-11-20

## Context

A release ships the window as `Spwn_darwin_arm64.zip`, signed, notarized and stapled, and nothing brings an installed
window to the next one. The owner decided that it updates itself with the Tauri 2 updater, signed by the key the
desktop app of `apps/web` already carries. That app shipped with the same key and polls the same
`releases/latest/download/latest.json`: every spwn desktop app up to v0.15.5 would read the window's manifest. The
updater compares the version compiled into the binary, and `make window` compiled `0.1.0` into every build.

## Decision

- **The binary carries its release's version.** `make window` passes the version git describes to `tauri build`, so
  the bundle and the updater read the tag without its `v`; newer is semver precedence, an alpha included, and no
  downgrade is offered.
- **The update is made after the staple.** `scripts/window-update.sh` packs the stapled `Spwn.app` into
  `Spwn.app.tar.gz`, signs it with the private half of that key, verifies the signature against the public half the
  window carries, and writes `latest.json`; the release uploads the three files beside the zip. A run that does not
  sign makes none of them.
- **The manifest names one platform, `darwin-aarch64-window`**, the target the window's updater looks up. A spwn
  desktop app of `apps/web` looks up `darwin-aarch64`, finds nothing, and offers nothing.
- **Rust owns the updater.** Two commands of the window's own check and install; the webview passes no URL, version
  nor file, and is granted no permission of the plugin. A debug build, and any build off macOS, answers that it does
  not update, and reaches no network.
- **The window checks once, at its start, and offers.** No timer and no notification, as for the engine
  ([ADR-060](060-the-window-offers-the-engines-release-and-updates-on-a-press.md)); the press is the consent to swap
  the bundle and relaunch.
- **The manifest's version is signed.** `requireSignedVersion` is on, and the signature binds the version, so a
  manifest cannot pair a new version with an old archive.

## Alternatives refused

- **The plugin's JavaScript API.** It hands the webview a check with any endpoint and an install of whatever it
  downloaded.
- **`createUpdaterArtifacts`.** Tauri would pack the ad hoc build, before its signature and its ticket, and every
  `make window` would need the private key.
- **A fixed-tag endpoint, or a second moving release.** A binary cannot learn a tag it does not know, and a second
  release is one more thing to keep for what `releases/latest` already gives.
- **The default target `darwin-aarch64`.** An old spwn desktop app would offer the window and swap it into its own
  bundle.

## Consequences

- The public key lives in two configs, `apps/web`'s and the window's; a rotation touches both and strands every
  installed window on its version.
- A release that renames the three files, or a window whose endpoint moves, strands every installed window.
- A window run from a read-only or translocated place cannot swap itself; the install says so.
- The swap and the relaunch are proved outside `make test`: by a rehearsal in the macOS VM, then by the first real
  update.
