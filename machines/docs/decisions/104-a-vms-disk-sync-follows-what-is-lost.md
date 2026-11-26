# ADR-104: A VM's disk sync follows what is lost

**Status:** Proposed
**Date:** 2026-11-26

Taken by the owner's standing delegation of 2026-11-26 (performance lot PF6), for him to overturn.

## Context

The performance audit of 2026-11-26 measured a 4 KiB write with `oflag=dsync` inside a fresh workspace at 7.1 to 7.8 ms, against 1.6 GB/s for the same writes buffered (finding F3). The adapter passed tart no `--root-disk-opts`, so every VM ran on tart's default, `VZDiskImageSynchronizationMode.full` (tart 2.42 `VM.swift:54`; the flag is parsed at `Run.swift:267`): each flush the guest asks for waits until the Mac's SSD holds the data, past the drive's own cache. The audit read that dpkg flushes each file it unpacks, so a package install, an image build and a Machinefile's apply would each wait on thousands of flushes; what a workload really waits is measured below.

Apple documents three modes. `full` "ensures the data moves from the disk's internal cache to permanent storage", so no synchronized data is lost to a panic or a power loss. `fsync` has "the same guarantees as the fsync system call": on macOS the data reaches the drive but may still sit in its cache. `none` "doesn't guarantee data integrity if any error condition occurs, such as disk full on the host, panic, power loss", and Apple names its use: "a VM is run only once to perform a task to completion or failure".

What a lost write costs is not the same for every VM. A build VM is thrown away on any failure and made again from its Machinefile. An ephemeral workspace is deleted when its work ends. A persistent workspace and a server hold the owner's data.

## Decision

Durability follows what is lost, and the engine chooses it per VM at each boot; nothing is asked of the owner.

| VM                   | Disk sync                                           | Why                                                                                                                                 |
| -------------------- | --------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| An image's build VM  | `none` (`--root-disk-opts=sync=none`)               | a failed build is thrown away whole; the image is recorded only after the cleanup's `sync` and a clean poweroff the build waits for |
| Ephemeral workspace  | `none`                                              | deleted when its work ends                                                                                                          |
| Persistent workspace | `full`, the hypervisor's default: no flag is passed | a Mac's power cut must not lose a write the guest was told is done                                                                  |
| Server               | `full`, the same                                    | the same                                                                                                                            |

- **One rule.** `machines.Lifetime.Sync` is the rule for a VM the engine starts; the build's boot (`wire/hypervisor.go`, `BootBuild`) asks `none`. `contract.BootOptions.Sync` carries it to the hypervisor's adapter, and tart's adapter alone turns it into a flag (`machines/adapter/tart/run.go`).
- **No `fsync` for persistent VMs.** Its guarantee is fsync(2)'s on APFS, which leaves the data in the drive's cache; a Mac mini has no battery behind that cache, so whether a power cut loses it is not documented. It is not taken until a measure settles it (below).
- **The proof.** The world's fake hypervisor records the mode of each boot (`booted.sync` in `world.json`); a story per kind holds it, the tart adapter's argv is pinned by its unit tests and its two stories that boot a build VM and an ephemeral one.

## Measured

The cost side only, on the new Mac mini (M5 Pro, tart 2.42) on 2026-11-26, in ephemeral workspaces made by its installed station, so on `full`: the host was never quiet (load1 11 to 20, a gate VM busy beside), 4 readings above load1 20 were skipped. The `none` side needs this branch's engine on a Mac and was not measured; the guest-side variant of each workload that issues no flush stands in for it, a proxy that leaves ext4's own journal flushes in place.

| Workload, inside the VM                                     | `full` (n, median, range)                   | No flush asked by the guest (n, median, range)      |
| ----------------------------------------------------------- | ------------------------------------------- | --------------------------------------------------- |
| 2000 writes of 4 KiB, `dd oflag=dsync`                      | 12, 14.59 s (7.3 ms each), 12.71 to 26.13 s | 6, 0.005 s buffered, 0.004 to 0.007 s               |
| `apt-get install` of 5 packages, 17 debs, from a warm cache | 6, 6.45 s, 6.15 to 7.48 s                   | 6, 6.20 s with `--force-unsafe-io`, 5.79 to 6.95 s  |
| `git clone --no-local` of git.git (326 MB), with checkout   | 6, 19.13 s, 17.26 to 21.77 s                | 6, 18.91 s with `core.fsync=none`, 18.49 to 20.76 s |
| `docker load` of `node:22-slim` (79 MB)                     | 6, 2.70 s, 2.14 to 3.97 s                   | not run                                             |

A flush costs 7.3 ms, but these workloads ask few: dpkg's own flushes are 0.25 to 0.47 s of a 6.45 s install (4 to 7 %: the gap of the medians, then the median of the six paired rounds), and a git clone waits on none that matters. The gain of `none` on an image build or a Machinefile's apply is therefore unproven and likely small; the audit's estimate of 2 to 5 times faster is not supported by the proxy. What `none` does to the Mac's SSD writes is unmeasured.

## Consequences

- An ephemeral workspace kept across a Mac power cut may come back with a damaged disk. Its stop and its reboot lose nothing: the guest's writes are in the Mac's cache when its run ends. The fix is the one a broken ephemeral VM already has: delete it and make it again.
- An image should be whole on disk once the hypervisor's process has ended, since every write it acknowledged is in the Mac's cache, which every later read sees; that is not yet proven on a real build (a build on `none`, then a workspace from it that boots, with `dpkg --audit` empty and the package database's checksum equal to the one the cleanup saw). A power cut in the seconds after a build, before macOS writes that cache out, could leave a recorded image whose disk is torn; flushing the image's disk on the Mac before its record is written closes that, and is left open.
- A persistent workspace and a server keep the 7 ms flush. Changing them needs a measure first: a persistent VM on `fsync`, power cut by hand on a throwaway Mac mid-write (a `dd oflag=dsync` loop that logs each acknowledged block), read back after the boot.
- Lifetime is fixed at creation, so a VM never changes mode while it runs.
