# ADR-020: A world image is tagged by its tool set's hash

**Status:** Proposed
**Date:** 2026-11-24

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

## Context

Every world of a machine was built under one tag, `spwn/world:latest`, whose cache key was a label holding the hash of the generated Dockerfile and its files. A spawn of another tool set rebuilt the image and moved the tag, so a spawn that had built its image and not yet created its container could start from another set's image and fail its tool probe. Commit `baeb87a7d` closed that window with one flock in the temp folder, held from the image step through the container's creation: credentials, the gate's start and every other step in between ran under it, machine-wide.

One slow holder then stalled every parallel `spwn up`. The CLI specification brings up worlds in parallel workers, and several of them hit their 120 s timeout under load where a healthy run takes 3 to 4 s.

## Decision

A world image is tagged `spwn/world:<hash>`, the hash its build already computed, and a world is created from that tag.

- **Two tool sets never share a tag**, so no build moves the tag another spawn creates from, and the machine-wide lock over the spawn is gone.
- **One build of a tag runs at a time.** `packages/compile` holds a flock per tag in the temp folder around the cache check and the build only, never across the rest of a spawn. Its wait is bounded at 10 minutes, past which the build fails and names the tag. It keeps concurrent spawns of one tool set from building it twice, and a forced rebuild from untagging it under a sibling's build.
- **An explicit image is untouched**: `--image` and `SPWN_BASE_IMAGE` name a tag the user owns.

## Consequences

- A machine keeps one image per tool set it has spawned, where it kept one tag and dangling layers. Nothing prunes `spwn/world` yet.
- A forced rebuild still untags its own hash for the time of its build, so a spawn of the same tool set that already built and has not yet created its container can find the tag missing.
- `spwn/world:latest`, left by earlier versions, is no longer read and nothing removes it.
