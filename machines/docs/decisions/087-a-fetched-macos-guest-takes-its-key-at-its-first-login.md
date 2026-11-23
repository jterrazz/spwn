# ADR-087: A fetched macOS guest takes its key at its first login

**Status:** Proposed
**Date:** 2026-11-23

Amends [ADR-084](084-base-macos-is-fetched-from-cirrus-labs-pinned-by-digest.md).

## Context

ADR-084 fetches `ghcr.io/cirruslabs/macos-tahoe-base` as Cirrus Labs made it. That image holds one account, `admin`, whose password is the default Cirrus Labs gives every image, `admin`, and no key of Machines. The engine reaches every VM as the VM user by the image key, and the owner's `ssh` by the device key the guest writes in its keys block (ADR-064, ADR-077, ADR-085). Neither is in a fetched macOS guest. On the new Mac mini on 2026-11-23 (acceptance pass 5), `station machines create macos-test --image base-macos` always ended `Failed at SSH 4 of 4`, the engine's log reading `ssh: unable to authenticate, attempted methods [none publickey]`. Its hint, `station machines start macos-test`, answered `is already running`, and the row stayed failed.

A Linux base is plumbed while it is built: `linux.sh` runs once over the base's `admin` password, adds the user and the image key, turns password login off, and the cleanup removes `admin` (ADR-044, ADR-051). A fetched image has no build to run that in. Three ways to put the key in were weighed:

- **A tart feature.** `tart run --dir` shares a folder with the guest, but a macOS guest mounts it only after a login, and something inside must still copy a key from it. tart has no cloud-init for macOS guests.
- **The guest agent's path.** The engine sends the keys in its hello to `station guest` inside the VM (ADR-077). That path needs the engine to be connected already, the very thing missing.
- **One login by the published default.** The same login `images build` already makes to a Linux base (`baseAdmin`, `admin`/`admin`), run once per VM, then retired.

## Decision

1. **A fetched image's VM is keyed at its first login.** When the engine's dial by the image key is refused (or this Mac holds no image key yet) and the VM's image carries a first login (`catalog.Image.FirstLogin`, `base-macos` alone), the engine makes the image key if it is missing, logs in once as `catalog.UpstreamLogin` (`admin`/`admin`, a public constant of every Cirrus Labs image, never a secret of the owner's), copies this Mac's own station binary to `/tmp/station`, runs `apps/engine/catalog/images/macos-firstlogin.sh` as root with its values on stdin, never in a command line, then dials by the image key again. The host key the first login meets is pinned in the VM's known hosts, as the engine's own dial pins it.
2. **The first login leaves no known password.** In the guest, the script makes the VM user, an admin whose password is made there and never kept. It gives that user a passwordless sudo, the image key in `~/.ssh/authorized_keys` and `/usr/local/bin/station`. It removes the auto-login and `/etc/kcpassword`, gives `admin` a password made there and never kept (`dscl . -passwd` by the old one, which a macOS guest requires of root for an account holding a secure token), and fails unless `admin`/`admin` no longer authenticates. Only then does it turn sshd's password and keyboard-interactive logins off in `/etc/ssh/sshd_config.d/000-station.conf`, checked with `sshd -T`: a step that fails past the password change leaves no known password, where a lock first would leave `admin`/`admin` alive and no password login to retry by. The keys block, the device key in it, is then the guest agent's, as on Linux.
3. **The engine tries it at every connection until it holds.** A first login that fails leaves the VM failing SSH answers; the engine's next connection, its own retry or a start, runs it again. A guest that took it refuses the password from then on, so it never runs twice.
4. **A start of a VM whose create failed is its retry even while it runs.** `machines.start` answers `is already running` only for a VM whose create did not fail; one that runs with a failed birth is run (a no-op), waited for and set up as ADR-062 §3 says, so the hint a failed create gives is a gesture that can succeed.

## Consequences

- Footprint in the guest: one admin account named as the Mac's VM user, `/etc/sudoers.d/60-station-user`, `~/.ssh/authorized_keys` and `~/.config/station/` under its home, `/usr/local/bin/station`, `/etc/ssh/sshd_config.d/000-station.conf`. `admin` stays, with its passwordless sudo from the base, a password nobody holds and no auto-login, so the guest has no logged-in desktop after its next reboot. On the Mac: the image key, made at the first macOS VM when no build made it before.
- Two random passwords, `admin`'s and the VM user's, appear for an instant in a root command line inside the guest, and are never written anywhere. Nobody holds them; a desktop login to the guest needs one to be set from an ssh session first.
- The guest's station is the binary of the Mac's engine when the VM was keyed. A newer engine does not replace it; the Linux images' baked binary is in the same case.
- What is proved over the fake world: the key dial refused, the first login as the image's account, its line run once, the image's login retired, the create made; a first login refused too keeps the VM failing SSH answers; a running VM whose create failed is retried by its start. Not proved by any suite: the script on a real macOS guest, and `ssh <vm>` by the device key answering `sw_vers`. A macOS guest nests no VM (`kern.hv_support` is 0), so that proof is one acceptance run on a Mac after the train.
- ADR-084's `admin` "with a known password" holds for the image, no longer for a VM made from it.
