# ADR-076: Spwn on a machine: one active app, one background job, the engine updates

**Status:** Accepted
**Date:** 2026-11-22

Accepted by the owner on 2026-11-22, directly, answering the target architecture's proposal. His words are quoted beside each decision.

Amended by [ADR-096](096-spwn-is-one-app-its-menu-bar-notifications-and-engine-live-while-it-runs.md) (Proposed): no background job on a Mac; the engine is Spwn's child and lives while Spwn runs, which opens at login.

## Context

[ADR-073](073-spwn-app-carries-the-engine-on-a-mac-with-a-screen.md) proposed two install modes on a Mac, the app (mode A) and the bare binary (mode B), and [ADR-065](065-the-window-updates-itself-through-the-tauri-updater.md) gave the window its own updater. [ADR-064](064-a-key-never-leaves-its-device.md) put the key agent in a second launchd job. Together they leave two update channels, two background jobs and two ways to install on one Mac. The owner answered the proposal point by point and added rules for the install and the command.

## Decision

1. **A Mac runs Spwn.app and nothing else, arm64 and macOS 26 or later.** "1 ok". Mode B, the bare binary on macOS, goes. The move from B to A of ADR-073 lives one release, then is deleted ([ADR-072](072-before-1-0-every-station-runs-the-same-release.md): no legacy before a major release).
2. **A headless Mac is a session, not a daemon.** A LaunchAgent in a user session with automatic login, no root daemon. "ouverture de session auto mais uniquement par ma machine file à moi": the product never turns automatic login on by default; only the owner's own Machinefile sets it.
3. **The engine is the one updater.** "3 ok", after the Chrome and Google Updater comparison. The window, the menu bar, `spwn update` and the night all ask the engine; it swaps the bundle atomically, verifies it, rolls back on failure, and updates the guests. The Tauri updater of ADR-065 goes.
4. **One background job.** "4. oui fusionne": the key agent of ADR-064 is served by the engine, which is the one job.
5. **A one-time cleanup, per Mac, by the owner.** "5 oui": `sudo sfltool resetbtm`, then (-) on the old Local Network rows. Neither is ever done by the product.

### Install

- **No installer for a GUI user.** "je veux seulement pouvoir dl l'app dans un dmg et glisser dans le tradi application aussi si je veux". The app is downloaded as a disk image and dragged to Applications, or anywhere.
- **One active copy, anywhere**, in `/Applications` or `~/Applications`. The first launch sets up the command and registers the engine. A copy opened elsewhere asks "Use this one?" and takes over, the old copy going to the Trash. Two copies never carry different names.
- **Everything works without admin**: `~/Applications`, `~/.local/bin`, `~/.local/share`, per-user jobs.
- **A standalone command for people without a GUI and for machines.** "faut qu'on soit capable d'avoir une command indépendante, pour les gens qui veulent pas de gui aussi". On Linux it is the bare `spwn` binary under a systemd user unit. On macOS it is option A: the same Spwn.app installed by a command (`curl … | sh` mounts or copies it, links the command, starts the engine), the window never opened and the menu bar off.

### The command is a link, never a copy

`~/.local/bin/spwn` points at the active bundle's engine. On install, launch and update the engine checks it:

| Found                                                    | Action                                             |
| -------------------------------------------------------- | -------------------------------------------------- |
| Absent                                                   | Create it.                                         |
| Ours, to the right bundle                                | Nothing.                                           |
| Ours, to another copy                                    | Repoint it and say so.                             |
| A real file that is ours (signature team and identifier) | Replace it by the link and say so.                 |
| Foreign                                                  | Never touch it: "needs you", with replace or keep. |

Another `spwn` earlier in the login shell's PATH is reported. An Installation section of the Engine page, and `spwn doctor`, show the bundle, the link, the shadowing, the duplicates and the Background Task Management leftovers, each with its fix. There is one command name; the `station` link is removed at the rename. The owner did not object to these rules and did not answer them one by one, so they stand as proposed under his go.

## What was measured

Moving the bundle to another folder makes macOS ask Local Network again for the new path ("Found new path … allowing a new prompt"): the identifier rule does not carry (the Mac mini, 2026-11-22). An unanswered prompt, on a locked or headless screen, blocks every later refusal-recovery of Spwn, even back at the old path, until it is answered; the owner's Allow approved it. An in-place update keeps the grant. So a moved copy says "you moved Spwn: approve it again on this Mac's screen", and a moved copy is never restarted unattended on a headless Mac.

## Consequences

- ADR-065 is superseded; ADR-073 and ADR-064 stand, amended: mode B and its install.sh door leave macOS, and the key agent is no job of its own.
- ADR-073's own status is unchanged: its gate is still the owner's, on a real Mac.
- The window's updater, its manifest and `scripts/window-update.sh` leave; the engine's update, with its rollback, is built first ("Install and upgrade for customers" and "One Spwn on a Mac" in [`TODO.md`](../../TODO.md)).
- A Mac without a screen needs a session that logs in alone, which only a Machinefile turns on.
- Lots are in `TODO.md`, "Spwn on a machine".
