# Images

An image is a versioned, read-only VM disk that VMs are cloned from. Each Mac holds its own: built there from a base with a recipe embedded in the binary, or fetched where a registry offers it. The catalogue has two images: `linux`, built or fetched, and `macos`, fetched only (station has no recipe for it).

| Section           | Answers                                                    |
| ----------------- | ---------------------------------------------------------- |
| Listing           | `station images`, every Mac's                              |
| Building          | `station images build`, the base and the version           |
| Updating          | `station images update`, a version a registry offers       |
| The recipe        | What the image contains, and the two sessions that make it |
| The binary inside | `--binary` and `make build-linux`                          |
| Cleaning          | `images delete`; Tart's pull cache is `disk free tart`     |

## Listing

```sh
station images                  # ●  linux  2026.11.01  4.0 GB  jterrazz-macbook-pro  api · linux-1
station images --on mac-mini    # that Mac's alone
station images --json           # station.images.list/v1: name, version, size_bytes, mac, vms, newest
```

One row per version per Mac, This Mac first, then every other Mac that answers; an image no Mac holds is one row, `not built yet`. Images live on a Mac: inside a VM the family is refused (exit 2).

## Building

```sh
station images build            # every image that is missing or whose recipe changed
station images build linux
```

The base of `linux` is `ghcr.io/cirruslabs/ubuntu`, pulled once (about 5 GB) and cloned for the build. The version is the UTC date, `2026.11.07`, with `.2` for a second build the same day. Station records the hash of the recipe files, so a changed recipe is rebuilt by a bare `build`. Images need Tart: `station apps install tart`.

## Updating

```sh
station images update           # every image this Mac holds
station images update linux     # ●  linux  2026.11.03  fetched
```

Station releases no image: each Mac builds its own. An update asks the registry `ghcr.io/jterrazz/station-<image>` (one tag per version) for the newest version, and an image it does not offer (the registry answers 403 or 404) exits 1 with `not released · station images build linux`, the gesture that makes it here. A version the registry does offer is fetched only when it is newer than every version held: `is the newest` otherwise. The fetched version lands whole as an image disk beside the others, or not at all: a registry that does not answer, or a fetch that fails, exits 1 and leaves the image as it was. Like a build, it changes only the next VMs.

## The recipe

The recipes are shell files embedded under `src/catalog/images/`. The build copies the `station` binary into the build VM, then runs two sessions inside it behind its Mac, each over station's own ssh client: the first by the base image's password, the second by the image key the first installs:

1. as the base's own user, `linux.sh` creates `vm_user` with passwordless sudo and the Mac's image key, installs the packages, Docker, mise, the `station` binary, lets sshd carry the engine's forwards (`/etc/ssh/sshd_config.d/60-station.conf`: `AllowTcpForwarding local`), runs `station setup` for the shell block, installs two systemd units: `station-intent` (tells the Mac about a poweroff, see [Machines](machines.md#lifetimes)) and `station-hostkeys` (generates ssh host keys at first boot, ordered before `ssh.socket` outside the default dependencies, which would make a cycle), masks the getty on `/dev/hvc0`, since tart reads the console file back into the guest as input, and makes `hvc0` the kernel's console (`/etc/default/grub.d/60-station-console.cfg`, checked in the generated `grub.cfg`, the build failing without it), so the kernel's lines reach the console file a server's down cause is read from;
2. as `vm_user`, `linux-cleanup.sh` stops the base's `tart-guest-agent` (it runs as `admin`), removes the base's user and fails the build if it cannot, empties the machine id and the host keys, and powers off.

A VM's first boot sets its own name and machine id and grows the disk. No tailscale is in the image: a VM is reached through its Mac ([Machines](machines.md#ssh-and-herdr)).

## The binary inside

The image carries `station-linux-arm64`. A release build downloads the asset of its own version into `~/.cache/station/` and checks it against the release's `checksums.txt`. A development build has no release, so `make build-linux` makes the binary and you pass it:

```sh
make build-linux
station images build linux --binary .artifacts/go/station-linux-arm64
```

## Cleaning

`station images delete <image>[@<version>]` never touches a VM: a VM is an APFS clone, independent of its image once made. Bare `station images` marks every version older than the newest. Deleting never prunes Tart's cache of the base: that cache is Tart's, emptied by `station disk free tart` ([Disk](disk.md)), and the next build pulls the base again after it.

## Related

- [Machines](machines.md): creating a VM from an image.
- [Footprint](footprint.md): the image records and the image key.
