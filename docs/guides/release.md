# Release

This repository ships two artefacts, the `station` binary and the menu bar's `Station.app`, and they reach a machine two ways: a tagged GitHub release, or a build from a checkout.

| Section                | Answers                               |
| ---------------------- | ------------------------------------- |
| The tagged release     | What a `v*` tag builds and releases   |
| The dry run            | The darwin build, with no release     |
| How a machine takes it | The one-line install, and the upgrade |
| From a checkout        | `make provision`, `make deprovision`  |

## The tagged release

There is no deploy: the release IS the delivery. Pushing a version tag is the whole gesture.

```sh
git tag v3.0.0
git push --tags
```

`.github/workflows/publish.yaml` fires on `push: tags: v*` and runs three jobs, one run per ref at a time and never cancelled. Nothing is released on a merge to `main`: a merge only runs the validate workflow ([Test](test.md)).

- **Release** waits for Darwin, so a darwin build that fails releases nothing. It calls the shared `studio-actions` `publish-go.yaml` workflow with the binary name `station`, the build path `./src/cmd/station`, the Go version of `go.mod`, a linker flag that stamps the tag as the version, and two targets, `linux/arm64` and `linux/amd64`. It checks, then creates the release with those two binaries and their `checksums.txt`.
- **Darwin** runs on a GitHub macOS arm64 runner (`macos-26`, the newest Xcode 26 selected). It builds `darwin/arm64` and `darwin/amd64` with cgo, each with Xcode's clang for its arch, so the release's Mac binary has IOKit: Awake's hold and the battery, lid and thermal reads. Both are built for macOS 14.0 (Sonoma) and later, `MACOSX_DEPLOYMENT_TARGET=14.0`; the job fails a binary whose build settings do not say `CGO_ENABLED=1` or whose `minos` is not 14.0. `Station.app` keeps its own floor, macOS 26 (`apps/menubar/Package.swift`), since its panel rests on macOS 26 behaviours ([ADR-021](../decisions/021-the-menu-bar-keeps-its-system-window.md)). It then runs `make menubar`, which builds `Station.app` signed ad hoc, and zips it with `ditto` into `Station.app.zip`.
- **Attach** waits for both, then adds the three darwin assets to the release that Release created, and their lines to its `checksums.txt`, with `gh release upload --clobber`.

Each binary is released as an asset named `station-<os>-<arch>`, beside `Station.app.zip` and one `checksums.txt` for all five; the install script asks for its binary's name and refuses a binary the checksums do not vouch for. The version is stamped from the tag ([ADR-005](../decisions/005-the-version-is-stamped-from-the-tag.md)); `v3.0.0` is released, and the next is `v3.1.0`, the first to write the new words of a machine's kind and lifetime ([ADR-032](../decisions/032-the-words-of-a-vms-lifetime-and-network.md)). A release that renames those assets breaks every machine's install path.

The release is whole once Attach has ended: between Release and Attach, the time Attach runs, the latest release has no darwin asset yet, and an install on a Mac fails at its download.

`Station.app` is signed ad hoc, not notarised: it arrives by `curl`, which sets no quarantine attribute, so Gatekeeper never judges it. A copy downloaded through a browser carries the quarantine and is refused at its first open.

## The dry run

```sh
gh workflow run publish.yaml --ref <branch> -f dry_run=true
```

A `workflow_dispatch` run builds the Darwin job alone and uploads its five files as the run's artifact `station-darwin` (the three assets and `checksums-darwin.txt`); Release and Attach are skipped, so nothing is released. Its binaries are stamped `dry-run-<sha>`. `dry_run` defaults to true; set false on a `v*` tag, it releases that tag again, as its push did; from any other ref, it is a dry run still.

## How a machine takes it

A machine takes the release through the one-line install of [Getting started](../start/getting-started.md), which downloads the asset named for its platform; re-running it is the upgrade.

A Mac takes `Station.app` as an app of the catalog, `station apps install station-menubar` ([Apps and skills](../reference/apps-and-skills.md)), with no Xcode: it reads the latest release's tag once, downloads that release's `Station.app.zip`, refuses it unless its `checksums.txt` vouches for it, and puts it at `~/Applications/Station.app`, as `make menubar-install` does: the new bundle is copied beside the old, then renamed over it, so a failed install keeps the old app. A running `Station.app` keeps its old process until it is quit and opened again. Like that target, it registers no login item: the app's own "Open at login" switch does.

## From a checkout

```sh
git clone https://github.com/jterrazz/studio-station.git
cd studio-station
make provision
exec zsh
```

`make provision` builds, copies the binary to `~/.local/bin/station`, and runs `station init`, which asks the machine's name on a terminal and writes or migrates the `~/.zshrc` block ([Dotfiles](../reference/dotfiles.md)).

`make deprovision` removes the binary and touches nothing else; the user data survives.

## Related

- [Footprint](../reference/footprint.md): everything the binary writes on the machine it reaches.
- [Machines](../reference/machines.md): All machines, and remote access.
- [Commands](../reference/commands.md): every verb the installed binary answers to.
