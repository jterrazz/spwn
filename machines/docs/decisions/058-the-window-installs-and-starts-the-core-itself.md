# ADR-058: The window installs and starts the core itself

**Status:** Proposed
**Date:** 2026-11-19

## Context

The window, named spwn, is a client of the engine's socket: until now a Mac with no engine showed it every page's
"No engine answers", and nothing in the window could change that. On a Mac where the core was never installed, that
left the window useless on its own. The owner decided on 2026-11-19 that the app must work on its own: opened on a Mac
without the core, it takes the owner from nothing to a running engine.

The install already has one public form, `curl -fsSL https://spwn.sh/install.sh | bash`, which writes `station` and
`spwn` to `~/.local/bin` (`scripts/install.sh:7`, [the footprint](../reference/footprint.md)) and starts nothing.

## Decision

While no engine answers, the shell looks for the core on the disk and runs two programs of its own, each on a press:

- It looks for `~/.local/bin/station`, an executable file at that one path (`core_present`,
  `apps/desktop/src-tauri/src/local_core.rs`), every few seconds while the owner installs from a terminal.
- Install runs the public installer as its own line says, `curl -fsSL https://spwn.sh/install.sh | bash`, through
  `/bin/bash -o pipefail -c` (`core_install`), and the window shows the installer's own lines as sent. A failure is
  said in the installer's last error line: the window classifies nothing.
- Start the engine runs `~/.local/bin/station engine start` and nothing else (`core_start`).
- Each run gets an environment of its own, the window's cleared: `HOME`, `PATH` fixed to
  `/usr/bin:/bin:/usr/sbin:/sbin`, `SHELL` set to the user's login shell (the account's entry, else the window's
  `SHELL`, else `/bin/zsh`, so the installer writes the profile of the shell the owner logs into), and `USER`,
  `LOGNAME`, `TMPDIR` passed through. One run at a time, in a process group of its own, which Cancel ends with
  `SIGTERM`.

What it never does:

- It reads no release: no version, no upgrade, no check of what is out. The installer alone fetches.
- It looks nowhere but that one path: no `PATH` lookup, no `which`, no shell profile read.
- It never runs `station init`: the engine start alone, as the installer leaves the core.
- Nothing once the engine answers: the panel goes, and the window is the engine's client as before.

The page fetches nothing and the CSP is unchanged: the shell runs the programs, the page sends four commands.

The install folder has one owner, `scripts/install.sh:7`. Until the engine projects it into `packages/api`, the window
holds it as `CORE_DIR` in `apps/desktop/src-tauri/src/local_core.rs`, its comment citing that owner, and draws it as
`CORE_FOLDER` in `apps/desktop/src/domain/core.ts`.

## Alternatives refused

- Showing the command only, with a Copy, and waiting for a terminal: the app would still not work on its own, and the
  owner asked that it does. The command stays on the panel, beside Install, for a Mac whose owner prefers a terminal.

## Consequences

- A debug build runs no installer and no `station` unless `STATION_WINDOW_CORE` names a folder holding a fake
  `install.sh` and `station`; it refuses with `core.not-run` otherwise, so no test, bench or capture reaches
  `spwn.sh` or a real core. A release build compiles that path out.
- The window now starts programs other than the opener's two, and its README says which.
- The install folder moves to `packages/api` when the engine session projects it; both constants go then.
