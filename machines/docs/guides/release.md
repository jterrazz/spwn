# Release

Machines ships two artefacts, the `station` binary and the window's `Spwn.app`, which carries the engine and the menu bar's `Station.app`, and they reach a machine two ways: a tagged GitHub release, or a build from a checkout. The pipeline that cuts the release (the tag, the jobs, the dry run, signing and notarization, the rollback) is spwn's, one release for both products: [Release](../../../docs/guides/release.md). This page keeps what is Machines'.

| Section                | Answers                                          |
| ---------------------- | ------------------------------------------------ |
| What a release carries | The assets, their names, what a darwin one needs |
| How a machine takes it | The one-line install, the upgrade, the window's  |
| From a checkout        | `make provision`, `make deprovision`             |

## What a release carries

Each binary is released as an asset named `station-<os>-<arch>`, for `linux` and `darwin` on `arm64` and `amd64`, all listed in the release's one `checksums.txt`; the install script asks for its binary's name and refuses a binary the checksums do not vouch for. A release that renames those assets breaks every machine's install path. A release binary is linked with `-s -w` (`RELEASE_STRIP`, the one line of `.github/workflows/publish.yaml` that says so, which the darwin gate reads too): no symbol table and no DWARF, a third smaller and half the download, while a panic's trace and `go version -m` stay readable; a binary built from a checkout (`make build`, the tests') keeps its symbols.

The window rides the same release: `Spwn_darwin_arm64.zip` (its first install), then `Spwn.app.tar.gz`, its `.sig` and `latest.json` (its update), listed in the same `checksums.txt`. A release that renames these, or a window whose `plugins.updater` endpoint moves, strands every installed window.

The version is stamped from the tag ([ADR-005](../decisions/005-the-version-is-stamped-from-the-tag.md)); `v0.21.0-alpha.1` is Machines' first release from jterrazz/spwn, continuing that repository's version line after v0.20.0, with no compatibility kept with the v3 stations before it; until 1.0 a station speaks only to stations of its own release ([ADR-072](../decisions/072-before-1-0-every-station-runs-the-same-release.md)), so every Mac, every image and both apps move to each release.

A darwin binary is built on a Mac with cgo, so it has IOKit (Awake's hold and the battery, lid and thermal reads) and LocalAuthentication (the presence a grant asks), and links the Secure Enclave's Swift half, `make enclave`'s universal static library over CryptoKit, which the release's darwin job builds before both binaries; a build without cgo has no presence to ask and refuses every grant with `grant.no-presence`. Both darwin binaries run on macOS 14.0 (Sonoma) and later, `MACOSX_DEPLOYMENT_TARGET=14.0`, and the release fails one whose build settings do not say `CGO_ENABLED=1` or whose `minos` is not 14.0. The menu bar keeps its own floor, macOS 26 (`apps/menubar/Package.swift`), since its panel rests on macOS 26 behaviours ([ADR-021](../decisions/021-the-menu-bar-keeps-its-system-window.md)); `make menubar` builds it, and `make window` builds it into `Spwn.app/Contents/Library/LoginItems/`.

A released `Spwn.app` is signed with the Developer ID, its menu bar inside it, notarized and stapled, so Gatekeeper accepts it even downloaded through a browser, which sets the quarantine attribute that `curl` does not; one built by `make window` stays signed ad hoc, and a browser's copy of it is refused at its first open.

## How a machine takes it

A machine takes the release through the one-line install of [Getting started](../start/getting-started.md), which downloads the asset named for its platform; re-running it is Machines' own upgrade. What its Machinefile carries is upgraded by `station machinefile update`, which leaves Machines itself alone ([Machinefiles](../reference/machinefiles.md#upgrade)).

A Mac takes the menu bar with Spwn.app, which carries it as its login item and updates it with itself: no catalog app installs it on its own, and the app's first start deletes the `~/Applications/Station.app` an earlier release installed, its login item unregistered first ([ADR-073](../decisions/073-spwn-app-carries-the-engine-on-a-mac-with-a-screen.md)).

The window updates itself: at each start it reads `latest.json` of the latest release and, when it is newer, offers it on the window's Settings; Install and relaunch swaps the bundle where it runs and relaunches it ([ADR-065](../decisions/065-the-window-updates-itself-through-the-tauri-updater.md)).

## From a checkout

```sh
git clone https://github.com/jterrazz/spwn.git
cd spwn/machines
make provision
exec zsh
```

`make provision` builds, copies the binary to `~/.local/bin/station`, and runs `station init`, which asks the machine's name on a terminal and writes the `~/.zshrc` block ([Dotfiles](../reference/dotfiles.md)).

`make deprovision` removes the binary and touches nothing else; the user data survives.

## Related

- [Footprint](../reference/footprint.md): everything the binary writes on the machine it reaches.
- [Machines](../reference/machines.md): All machines, and remote access.
- [Commands](../reference/commands.md): every verb the installed binary answers to.
