# ADR-043: An image is baked from its Machinefile; the recipe is plumbing alone

**Status:** Proposed
**Date:** 2026-11-12

## Context

The Linux image's recipe held a hand-written list of what a workspace carries: apt packages, Docker, mise, then the libraries headless Chromium needs (L5.0). That list was a second answer to the question a Machinefile answers ([ADR-033](033-a-machine-is-declared-by-its-machinefile.md)): what an owner's workspace wanted baked meant editing station's own shell file, a changed Machinefile never rebuilt an image, and an image kept no trace of the versions it carried. The design asked for `station images build <machinefile>`, the image named after its Machinefile and recording its hash and lock, and three phases (`image`, `boot`, `apply`), and left open how the build VM reads the Machinefile, what the image phase installs, and what becomes of the image called `linux`.

## Decision

- **The recipe is plumbing.** `linux.sh` keeps what a VM of station needs whatever it carries: the user, sudoers, the image key, the sshd drop-in, the units, the console, the docker group, the station binary and `station init`, and the few packages that plumbing itself uses. Everything an image carries is its Machinefile's.
- **The image phase is `vm.image.bake`.** `station apply --phase image` carries each item the bake list names and what a baked item requires (its requires, its provider, its manager), in the plan's order; a bare apply carries the baked items with every other, so on a VM from the image they read present and are kept. `--phase boot` carries the items whose `phase` is `boot`. Only apply's phase has extras, so `--remove` with another phase is refused.
- **The build VM reads the Machinefile as this Mac read it.** The build puts every file of the Machinefile's chain that is not station's built-in one (the binary embeds those), its overlay included, and this Mac's lock of it into the VM user's folder, its shared source left out: the Mac read the source at its pinned commit, and the VM has none of its own. It runs `station apply --machinefile <name> --phase image` as that user, its sudo passwordless, bounded at 30 minutes like the plumbing.
- **An image is named after its Machinefile and records it.** Its record holds the Machinefile, the hash of the recipe files, the hash of the chain and lock it was built from, and the lock the image phase wrote. A bare `images build` rebuilds an image whose recipe, chain or lock changed. Every Machinefile of a VM on Ubuntu that says `vm.image.from` is an image of this Mac.
- **`linux` becomes `station-linux`, for one release under both names.** The built-in Linux workspace's Machinefile is `station-linux`; `linux` names it in every images command and in `machines create --image`, its records are read as `station-linux`'s, and an update asks the registry under both names. A VM made from it is still `linux-<n>`: a VM's default name is its system's word, not its image's.
- **`station-linux` bakes what the recipe carried.** mise, git (now with an apt method), Docker, `build-essential` and `playwright-deps` (the 24.04 list of `playwright install-deps chromium`, an apt item), so a fresh workspace runs a browser gate and `docker run` as before.

## Consequences

- A workspace's content moves by editing a Machinefile, and the next bare `images build` sees it; station's own recipe changes only with the plumbing.
- An image carries its lock, which a VM's first start can read against the machine (hard case 17), a later lot's.
- The lock a build installs comes from this Mac's folder today; a lock committed beside its Machinefile in the shared source (open decision 3) is read by a later lot.
- `station-linux` keeps Docker and the browser libraries that decision 2 left to `studio-linux`, so no workspace loses them before `studio-linux` exists; the owner trims it once it does.
- The menu bar's New VM sheet still sends `linux`, read through the alias; it has to send `station-linux` before the alias goes.
- gnupg, which the recipe installed, is no longer in the image: the `gpg` item has no Linux method yet.
