# ADR-082: A server keeps its data on a disk of its own

**Status:** Accepted
**Date:** 2026-11-23

Accepted by the owner on 2026-11-23, on the owner's word, with the answers recorded below.

**Amends:** [ADR-044](044-an-image-is-baked-from-its-machinefile.md): a VM's disks, until now its boot disk alone.

**Amends:** [ADR-033](033-a-machine-is-declared-by-its-machinefile.md): the `vm` family gains `vm.data`.

## Context

A server's boot disk is made from its image and can be rebuilt from its Machinefile (ADR-044, [ADR-079](079-a-server-runs-the-image-phase-then-a-first-start-phase.md)). A cluster's data cannot be rebuilt: the volumes, the k3s local-path storage and the backups written beside them. The plan of 2026-11-20 put that data on a share of the Mac's folder, `~/.studio-infrastructure` mounted at `/srv/studio-infrastructure` through `vm.mounts`. On 2026-11-21 the owner replaced it: the server gets a dedicated data disk, a second disk image, ext4, at `/srv/data` in the guest, with k3s's local-path at `/srv/data/k3s`. It is kept apart from the boot disk the Machinefile rebuilds (`TODO.md`, S6).

What the code has today:

- **One disk per VM.** tart runs a VM with `--disk-size` for its boot disk (`apps/engine/machines/adapter/tart/tart.go:265`), and each share is a `--dir` with its own tag (`apps/engine/machines/adapter/tart/run.go:77-82`).
- **`--disk` already has a meaning.** `machines edit --disk` sizes the boot disk (`apps/engine/surface/cli/machines/edit/command.go:36`).
- **The guest mounts with `nofail`.** ADR-044 writes each share as a `nofail` line of `/etc/fstab` at first start, before the boot phase.

The rehearsal's ext4 enforced ownership: five stateful workloads crash-looped on volumes made `root 0755`. So studio-infrastructure's ownership lot applies to a disk, where a share might have spared it.

## Decision

1. **`vm.data: { size }` declares a server's data disk.** It is one raw disk image of that size, attached by tart as a second disk. The key is refused on a workspace and on a computer.
2. **The image lives outside the VM's tart folder**, at `~/.local/state/station/data/<vm>.img`, so deleting or rebuilding the VM never deletes it. The footprint page lists it.
3. **Its first start formats it**: ext4, labelled `station-data`, and only when it holds no filesystem. A disk that already holds one is mounted as it is, never formatted.
4. **The guest mounts it at `/srv/data`**, by label, through a `nofail` line of `/etc/fstab`. The line is written at the first start before the boot phase, like `vm.mounts`, so `run.boot` and the cluster's units may require it. The path is fixed. What lives under it (`k3s`, the backups) is the server's tooling's.
5. **A rebuild keeps it.** A server created under a name whose data image exists attaches that image, and its create says so, naming the file. A new size larger than the image grows it (the image, then `resize2fs` at the next start). A smaller size is refused: a disk never shrinks.
6. **Deleting a server keeps its data disk by default.** The delete names the file it kept. `machines delete <server> --data` removes it too, after the server's typed name.

## Alternatives refused

- **A share of the Mac's folder (`vm.mounts`)**: the plan's first design, replaced by the owner. It also left two questions open, whether a share enforces ownership and what fsync costs through it, and the cluster's database would have sat on them.
- **One larger boot disk.** A rebuild from the Machinefile would erase the data. A server is rebuilt to change its machine, never its data.
- **A free list of disks with free guest paths.** One server needs one data disk. A list with paths would need rules for which guest folders a disk may take, as `vm.mounts` needed, for no need asked.
- **Format at every first start.** A rebuild would erase the data it was made to keep.

## Consequences

- S6 adds `vm.data` to the decoder and the JSON Schema, the attach in the tart adapter and the fake, the format and the mount in the first start, and the kept file at delete. Its proof is the stories and a tier-2 case: data written, the boot disk rebuilt, the data read back.
- The data image counts on the Mac's disk. The Disk section and `docs/reference/disk.md` show it in S6.
- The backups that stay beside the data (the owner, 2026-11-21) now sit inside the same image file. That file is lost with the Mac's disk, so the backups offer no safety beyond that disk. Offsite backup is studio-infrastructure's TODO. The owner accepted this for now on 2026-11-23.
- An image attached to a new server of the same name is that server's data. A name reused for an unrelated server would mount the old data: the create's line naming the file is the guard.
- HYPOTHESIS (2026-11-23): tart's `run --disk=<path>` attaches a raw image read-write as a second virtio disk, with a sync mode that honours fsync. To settle it, read `tart run --help` on development-macos. S5's `server-facts` case measures fsync on this disk and on the boot disk.
