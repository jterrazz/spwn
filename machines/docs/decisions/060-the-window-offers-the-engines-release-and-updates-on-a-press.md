# ADR-060: The window offers the engine's release and updates on a press

**Status:** Accepted
**Date:** 2026-11-19

## Context

The engine reads the latest release and words whether this Mac is behind it
([ADR-059](059-the-latest-release-is-read-by-two-calls.md)). The window installs and starts the core with two programs
of its shell ([ADR-058](058-the-window-installs-and-starts-the-core-itself.md)). The owner decided on 2026-11-19 how
the two meet: the update lives on an Engine page opened from the sidebar foot, it is offered and never automatic, and
it tells nothing outside the window.

## Decision

- **The Engine page draws `engine.read-release` as given.** Its head is the reading's `Head`; its "Latest release"
  row says when the engine last checked and its last refusal, each only when the reading carries it; "Check now" is
  `engine.refresh-release`, whose refusal is drawn in the engine's own message. The window compares no versions.
- **The sidebar foot carries a quiet row while the reading is `Behind`**: the reading's `Latest`, "available", and
  "Update", opening the Engine page. It is no notification, and the window adds no timer: the reading is asked when
  the page or the foot mounts, and again on the store's ordinary rhythm.
- **Both are gated by the engine's hello.** An engine whose `calls` do not name `engine.read-release` shows no row and
  no release section, and no error.
- **Update is one press running two programs**: ADR-058's installer, then `station engine start` with the `station`
  the engine's hello names. Each step's lines are shown as the installer's are, and a failure is said in the step's
  own last error line. Cancel ends the installer; a start is not cancelled.
- **"What changed" opens the reading's `Notes`** through the opener, and is drawn only when the reading carries them.

## Alternatives refused

- **An update the window runs by itself.** The owner refused it: an install replaces the binaries the engine runs, and
  is the owner's to start.
- **A system notification.** Refused for the same reason: the window says what is out where it is looked at.

## Consequences

- The update outlives the engine it restarts: its state is the window's `UpdateFlow`, apart from the reads, so the
  quiet window ADR-058 draws while no engine answers still shows it.
- A recorded world's hello is made from the calls it holds answers for, so a proof build over a world without a
  release reading draws neither the row nor the page's release.
