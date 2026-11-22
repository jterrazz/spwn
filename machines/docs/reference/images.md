# Images

An image is a versioned, read-only VM disk that VMs are cloned from. Each Mac holds its own: built there from a Machinefile, the image named after it, or fetched where a registry offers it ([ADR-044](../decisions/044-an-image-is-baked-from-its-machinefile.md)). Machines knows one base per distro ([ADR-051](../decisions/051-a-machine-is-a-base-and-its-machinefiles.md)): `base-ubuntu`, `base-debian` and `base-macos`, each named after its built-in Machinefile and sized by its `vm.resources`, `base-macos` fetched only (Machines has no recipe for it yet); every Machinefile of yours or of the shared source that says a VM's `vm.image.from` on Ubuntu or Debian is an image too (`studio-linux`). The ADR's other half, `base-macos` built here and `clt` and `xcode` as Machinefiles, is not built yet.

| Section           | Answers                                                 |
| ----------------- | ------------------------------------------------------- |
| Listing           | `station images`, every Mac's                           |
| Building          | `station images build`, the Machinefile and the version |
| Updating          | `station images update`, a version a registry offers    |
| The recipe        | The plumbing, the image phase, and the cleanup          |
| The binary inside | `--binary` and `make build-linux`                       |
| Cleaning          | `images delete`; Tart's pull cache is `disk free tart`  |

## Listing

```sh
station images                  # ●  base-ubuntu  2026.11.01  4.0 GB  macbook-pro  api · linux-1
station images --on mac-mini    # that Mac's alone
station images --json           # station.images.list/v1: name, version, distro, size_bytes, mac, vms, newest, deletable, building
```

One row per version per Mac, This Mac first, then every other Mac that answers; an image no Mac holds is one row, `not built yet`. Images live on a Mac: inside a VM the family is refused (exit 2).

A version is deletable when it is not the newest of its image, no machine was made from it and no build is making it; `Report.Unused` in `apps/engine/images/record.go` is that one rule, and `images.list` says it as `Deletable` (`deletable` in `--json`), the same rule that lists the versions [Disk](disk.md#what-nothing-uses) frees with the leftovers, through this delete.

An image being built is one more row, the version it will be, at its step: `Step 2 of 4` (`building` in `--json`). The four steps are the build's: Base system (the upstream pulled, cloned and booted), Packages (the station binary and the plumbing recipe), Settings (the Machinefile's image phase) and Seal (the cleanup, the power off and the record). A build marks its step in a file of `~/.local/state/station/builds/`, with the process it runs in, and clears it when it ends; `images.list` runs at its target, so the process that answers it reads those marks (on This Mac, the CLI's own), not the engine, and a mark whose process no longer runs is no row. The building row also lists the four steps in order (`Walk` in `images.list`), each with its `State`, `done`, `now` or `next` (a closed list: `ImagesBuildState` in `packages/api/words.json`, `done` ok, `now` new, `next` plain), and the word the engine says it by, so a client draws the walk and never works out which step has passed; a build at Seal reads `done`, `done`, `done`, `now`. The version a build is making is not deletable.

## Building

```sh
station images build                  # every image missing, or whose recipe, Machinefile or lock changed
station images build base-ubuntu    # the built-in Linux workspace
station images build studio-linux     # a Machinefile of yours or of the shared source
```

An image is built from a Machinefile of a VM that says `vm.image.from`, the upstream its disk starts from (`ghcr.io/cirruslabs/ubuntu:24.04`, pulled once, about 5 GB, and cloned for the build), read from every source as `station plan` reads it: a name no source holds is `no image "<name>"`, and a Machinefile that makes no image (a mixin, a computer's, a VM's with no `vm.image.from`) is refused naming why, both exit 2. The version is the UTC date, `2026.11.07`, with `.2` for a second build the same day. The record keeps the upstream, the Machinefile, the hash of the recipe files and the hash of the Machinefile's chain (every file it extends, its overlay) and lock, so a bare `build` rebuilds an image whose recipe, Machinefile or lock changed; it also keeps the lock the build VM wrote once its image phase ran, the versions the image carries. Images need Tart: `station apps install tart`.

`base-ubuntu`, `base-debian` and `base-macos` are the built-in images' only names: `linux`, `station-linux` and `macos` name no image, in any images command or in `machines create --image` (`unknown image "macos"`), and an image built under an old name is no longer read ([ADR-051](../decisions/051-a-machine-is-a-base-and-its-machinefiles.md)).

Every record names the image's distro, `ubuntu`, `debian` or `macos` (`catalog.Distro`, worded Ubuntu, Debian and macOS), and its version, the built-in image's Machinefile's `system.version` (`24.04`, `13`; none yet for `base-macos`); the image of a Machinefile of yours is the distro of the recipe it is built with, the one its `system.os` names (Ubuntu when it names none). A record that names no distro, one an older station wrote, is read no more: every images command that reads its image exits 2 with `image.old-record`, naming the file and `station images delete <image>@<version>`, the one gesture it lets through while no machine was made from that version and no build is making it. That delete removes the record and the version's disk; the next create that needs the image builds it again.

`base-macos` has no recipe: a macOS workspace starts from `ghcr.io/cirruslabs/macos-tahoe-base` (the `base-macos` Machinefile's `vm.image.from`), as Cirrus Labs made it. Its `admin` has a passwordless sudo because that base ships `/etc/sudoers.d/admin-nopasswd`, not because Machines set it, so `station apply` there asks no password only while that file stays ([ADR-034](../decisions/034-apply-carries-a-machinefile-the-ledger-bounds-removal.md)). Machines building it, with its sshd password login off, is decided by [ADR-051](../decisions/051-a-machine-is-a-base-and-its-machinefiles.md) and not built yet.

## Updating

```sh
station images update           # every image this Mac holds
station images update base-ubuntu     # ●  base-ubuntu  2026.11.03  fetched
```

Machines releases no image: each Mac builds its own. An update asks the registry `ghcr.io/jterrazz/station-<image>` (one tag per version) for the newest version, and an image it does not offer (the registry answers 403 or 404) exits 1 with `image base-ubuntu is not released` and the hint `station images build base-ubuntu`, the gesture that makes it here. A version the registry does offer is fetched only when it is newer than every version held: `is the newest` otherwise. The fetched version lands whole as an image disk beside the others, or not at all: a registry that does not answer, or a fetch that fails, exits 1 and leaves the image as it was. Like a build, it changes only the next VMs.

## The recipe

The recipe is the base plumbing of a system, shell files embedded under `apps/engine/catalog/images/`; what the image carries is its Machinefile's. Ubuntu and Debian run the same files: `base-debian` over `ghcr.io/cirruslabs/debian`, Debian 13 pinned by digest in its Machinefile, `base-ubuntu` over `ghcr.io/cirruslabs/ubuntu:24.04`. The build copies the `station` binary into the build VM, then runs three sessions inside it behind its Mac, each over Machines' own ssh client: the first by the base image's password, the others by the image key the first installs:

1. as the upstream's own user, `linux.sh` creates `vm_user` with passwordless sudo (the rule is `/etc/sudoers.d/60-station-user`, a name with no dot, since sudo skips a file whose name has one and a login may; the build checks it with `visudo -cf` and fails with the cause if `sudo -n` still asks `vm_user` a password) and the Mac's image key, its `~/.config/station/machinefiles` folder, and the `docker` group with `vm_user` in it; installs only what the plumbing itself uses (zsh, the user's shell; `curl` and the certificates; `cloud-guest-utils`, which grows the disk at first boot; `libatomic1`, which node's Linux builds link); the `station` binary; lets sshd carry the engine's forwards (`/etc/ssh/sshd_config.d/60-station.conf`: `AllowTcpForwarding local`); runs `station init` for the shell block; installs two systemd units: `station-intent` (tells the Mac about a poweroff, see [Machines](machines.md#lifetimes)) and `station-hostkeys` (generates ssh host keys at first boot, ordered before `ssh.socket` outside the default dependencies, which would make a cycle); masks the getty on `/dev/hvc0`, since tart reads the console file back into the guest as input; and makes `hvc0` the kernel's console (`/etc/default/grub.d/60-station-console.cfg`, checked in the generated `grub.cfg`, the build failing without it), so the kernel's lines reach the console file a server's down cause is read from;
2. as `vm_user`, the image phase: the build puts every file of the Machinefile's chain that is not Machines' built-in one, and this Mac's lock of it (its shared source left out), into `vm_user`'s folder, and `linux-bake.sh` runs `station apply --machinefile <name> --phase image` there, which installs what `vm.image.bake` lists and what it requires ([Machinefiles](machinefiles.md#phases)), then prints the lock that run wrote, which the image record keeps. `base-ubuntu` bakes mise, git, Docker, `build-essential` and `playwright-deps` (the libraries and fonts headless Chromium needs on Ubuntu 24.04), so a fresh workspace runs a browser gate and `docker run`. `base-debian` bakes nothing: a Debian machine carries the plumbing and what its own Machinefile adds;
3. as `vm_user`, `linux-cleanup.sh` stops the upstream's `tart-guest-agent` (it runs as `admin`), removes the upstream's user and fails the build if it cannot, empties the machine id and the host keys, and powers off.

The plumbing is bounded at 30 minutes, and the image phase at 35: 15 for its packages, then 20 for its `run.image` lines ([Phases](machinefiles.md#phases)). A VM's first boot sets its own name and machine id and grows the disk. No tailscale is in the image: a VM is reached through its Mac ([Machines](machines.md#ssh)).

## The binary inside

The image carries `station-linux-arm64`. A release build downloads the asset of its own version into `~/.cache/station/` and checks it against the release's `checksums.txt`. A development build has no release, so `make build-linux` makes the binary and you pass it:

```sh
make build-linux
station images build base-ubuntu --binary .artifacts/go/station-linux-arm64
```

## Cleaning

`station images delete <image>@<version>` deletes only a deletable version: the newest, one a machine was made from and one a build is making are refused (exit 2, `image.in-use`), naming why, whichever client asks; a bare `<image>` deletes every version, the newest included, but is refused (`image.in-use`, "being built") while a build of that image is making one. A delete never touches a VM: a VM is an APFS clone, independent of its image once made. Bare `station images` marks every version older than the newest. Deleting never prunes Tart's cache of the upstream: that cache is Tart's, emptied by `station disk free tart` ([Disk](disk.md)), and the next build pulls the upstream again after it.

## Related

- [Machines](machines.md): creating a VM from an image.
- [Footprint](footprint.md): the image records and the image key.
