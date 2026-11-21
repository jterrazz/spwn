# ADR-072: Before 1.0, every station runs the same release, and every Mac pushes its state

**Status:** Accepted
**Date:** 2026-11-21

Accepted by the owner on 2026-11-21, in his words: "toutes les machines envoient leur etat en continue, pas de legacy supporté bro tant qu'on a pas release une vraie version majeure (faut les forcer à upgrade)".

## Context

[ADR-070](070-the-engine-keeps-one-read-model.md) left three doors open for stations of other releases. A far Mac pushed its slices only when its record in `preferences.json` said so, and was polled every 30 s otherwise. The compatibility verdict was a range, so an app drew every engine from the oldest revision it accepted to the newest it knew. And a peer station was refused only below `api.Floor`, v0.1.0, so every release since then, and every v3 station of the numbering before the line restarted at v0.1.0, was served, with the calls and abilities its hello listed as the only guard.

Each door is code that keeps an older release working. Spwn has not shipped a major release yet, and the owner wants every machine on the latest one rather than a fleet the code has to carry across releases ([ADR-055](055-station-reads-only-its-current-state.md) took the same line for station's own state).

## Decision

Until 1.0, the rule is one release:

- **Every Mac pushes its slices continuously.** A far Mac's `engine.state` is held open by every other Mac. No record chooses push or poll, and no poll path exists. A Mac that stops pushing goes stale: its last values are kept and drawn stale, as a silent Mac's are.
- **The compatibility verdict has no range.** An app knows one page revision. An engine at that revision fits; below it, or naming none, the engine is older; above it, the app is older. The menu bar's `pageRevision` and the window's `PAGE_REVISION` are each one constant.
- **A peer station is served only on this station's release.** `remote/api.SkewOf` judges a peer's version against this station's, the same way on every side: the Router judges a far Mac's hello, the guest dialer a VM's, and a station serving frames a caller's.
    - A release is `vX.Y.Z` or `vX.Y.Z-<pre-release>`, as the publish workflow stamps it from the tag. A local build, as `git describe` stamps it (`<tag>-<n>-g<sha>`, `-dirty`), is judged as the release it describes. Two of them agree only on the same release.
    - A version that names no release (`dev` from `go run`, `dry-run-<sha>` from a dry run, a bare sha from a clone with no tags) is a build of no release. Such a peer is served, since no install is one. A station that is itself such a build serves only such builds, because it cannot vouch for being any release. So every released station, and every build that names a release, is held to the exact rule, and the only peers that slip past it are builds that a person chose to run.
    - A peer on another release is refused whatever it serves. The side that is behind is told to update. When the far side is older, the refusal is `far-too-old` with the far side's repair. When this station is older, or is a build of no release, it is `remote.caller-too-old` with `update station here`. A release of major 1 or above, while this station is at major 0, is of the numbering before the line restarted (a v3 station), and is the side behind. A station serving frames refuses such a caller as `remote.caller-too-old`. A caller ahead of it refused it first, because a far hello is judged before any call is sent.

This amends [ADR-070](070-the-engine-keeps-one-read-model.md): its per-Mac push or poll choice, its verdict range and its "a newer app keeps drawing an older engine" are withdrawn. It retires `api.Floor`. The rule for 1.0 and later is a decision of its own, taken when 1.0 is released.

## Consequences

- No code carries an older release: no floor to raise, no poll fallback, no range of revisions.
- Every machine updates at each release. A Mac, a VM or an app left behind is refused, or draws the incompatible verdict, until it is updated. A VM's station is the one its image was built with, so every image is rebuilt at each release.
- The calls and abilities a hello lists keep refusing a call the far side lacks. That can happen only between builds of one release, or builds of no release.
- `station machines` keeps listing a Mac on another release, with its version and the side to update; nothing else of it is read, so its VMs are the ones remembered, their state unknown.
- Every Mac's engine holds a watch to every other Mac. That is the cost the owner accepted for a fleet that is always current.
