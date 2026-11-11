# Release

This repository ships one artefact, the `station` binary, and it reaches a machine two ways: a tagged GitHub release, or a build from a checkout.

| Section                | Answers                               |
| ---------------------- | ------------------------------------- |
| The tagged release     | What a `v*` tag builds and releases   |
| How a machine takes it | The one-line install, and the upgrade |
| From a checkout        | `make provision`, `make deprovision`  |

## The tagged release

There is no deploy: the release IS the delivery. Pushing a version tag is the whole gesture.

```sh
git tag v3.0.0
git push --tags
```

`.github/workflows/publish.yaml` fires on `push: tags: v*` and calls the shared `studio-actions` `publish-go.yaml` workflow with the binary name `station`, the build path `./src/cmd/station`, the Go version of `go.mod`, a linker flag that stamps the tag as the version, and four targets: `darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`. Nothing is released on a merge to `main`: a merge only runs the validate workflow ([Test](test.md)).

Each target is released as an asset named `station-<os>-<arch>` plus a `checksums.txt`; the install script asks for that name and refuses a binary the checksums do not vouch for. The version is stamped from the tag ([ADR-005](../decisions/005-the-version-is-stamped-from-the-tag.md)); `v3.0.0` is released, and the next is `v3.1.0`, the first to write the new words of a machine's kind and lifetime ([ADR-032](../decisions/032-the-words-of-a-vms-lifetime-and-network.md)). A release that renames those assets breaks every machine's install path.

## How a machine takes it

A machine takes the release through the one-line install of [Getting started](../start/getting-started.md), which downloads the asset named for its platform; re-running it is the upgrade.

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
