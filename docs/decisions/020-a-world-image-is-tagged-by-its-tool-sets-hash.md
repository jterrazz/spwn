# ADR-020: A world image is tagged by its tool set's hash

**Status:** Accepted
**Date:** 2026-11-24

Accepted by the owner on 2026-11-24, on the owner's word.

## Context

Every world of a machine was built under one tag, `spwn/world:latest`, whose cache key was a label holding the hash of the generated Dockerfile and its files. A spawn of another tool set rebuilt the image and moved the tag, so a spawn that had built its image and not yet created its container could start from another set's image and fail its tool probe. Commit `baeb87a7d` closed that window with one flock in the temp folder, held from the image step through the container's creation: credentials, the gate's start and every other step in between ran under it, machine-wide.

One slow holder then stalled every parallel `spwn up`. The CLI specification brings up worlds in parallel workers, and several of them hit their 120 s timeout under load where a healthy run takes 3 to 4 s.

## Decision

A world image is tagged `spwn/world:<hash>`, the hash its build already computed, and a world is created from that tag.

- **Two tool sets never share a tag**, so no build moves the tag another spawn creates from, and the machine-wide lock over the spawn is gone.
- **One build of a tag runs at a time.** `packages/compile` holds an exclusive flock per tag in the temp folder around the cache check and the build only, never across the rest of a spawn. Its wait is bounded at 10 minutes, past which the build fails and names the tag. It keeps concurrent spawns of one tool set from building it twice.
- **A forced rebuild never untags.** `--force-rebuild` builds with no cache and lets the new image take the tag; the old image stays, untagged, for any container still using it, and a sibling spawn about to create from the tag finds the new one.
- **A spawn holds its tag in use** from its build to its container's creation: a shared flock per tag, which any number of spawns take at once, waiting only through the instant a prune removes that very tag.
- **A new tag prunes the others.** After a build that wrote a new tag (not a cache hit), every other `spwn/world:<hash>` is removed unless a spawn holds it in use, which the pruner learns by taking that tag's in-use flock exclusive without waiting, or a container, running or stopped, was created from it, which Docker refuses to remove without force. A prune is best-effort: each removal is written to the build log, and no failure fails the spawn.
- **The locks belong to the user**: their files carry the uid in their name and mode 0600. They are never removed, since unlinking a flock file another process has open would let a third lock a new file beside it.
- **An explicit image is untouched**: `--image` and `SPWN_BASE_IMAGE` name a tag the user owns.

## Consequences

- A machine keeps the world image of each tool set a container was created from, and drops the others at the next new build. A tool set spawned again after its image was pruned builds again.
- The temp folder keeps two empty lock files per tag ever built.
- `spwn/world:latest`, left by earlier versions, is no longer read, and the first new build prunes it like any other tag once no container uses it.
