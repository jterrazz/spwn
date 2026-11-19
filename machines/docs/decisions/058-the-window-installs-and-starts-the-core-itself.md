# ADR-058: The window installs and starts the core itself

**Status:** Accepted
**Date:** 2026-11-19

## Context

The window, named spwn, is a client of the engine's socket: until now a Mac with no engine showed it every page's
"No engine answers", and nothing in the window could change that. On a Mac where the core was never installed, that
left the window useless on its own. The owner decided on 2026-11-19 that the app must work on its own: opened on a Mac
without the core, it takes the owner from nothing to a running engine.

The install already has one public form, `curl -fsSL https://spwn.sh/install.sh | bash`, which writes `station` and
`spwn` to `~/.local/bin` (`INSTALL_DIR` in the site's `public/install.sh`, jterrazz/spwn-web, its ADR-014) and starts
nothing.

## Decision

While no engine answers, the shell looks for the core on the disk and runs two programs of its own, each on a press:

- It looks for `station` as an executable file at an absolute path, in three places and in this order
  (`station_of`, `apps/desktop/src-tauri/src/station.rs`): the `executable` the engine's hello names, when an engine
  answers; the engine's record, `station/engine.json` under the state root (`station.engine-record/v1`, written at
  each engine start); then the install folder. `core_present` looks every few seconds while the owner installs from
  a terminal.
- Install runs the public installer as its own line says, `curl -fsSL https://spwn.sh/install.sh | bash`, through
  `/bin/bash -o pipefail -c` (`core_install`), and the window shows the installer's own lines as sent. A failure is
  said in the installer's last error line: the window classifies nothing.
- Start the engine runs `station engine start`, the `station` found there, and nothing else (`core_start`).
- Each run gets an environment of its own, the window's cleared: `HOME`, `PATH` fixed to
  `/usr/bin:/bin:/usr/sbin:/sbin`, `SHELL` set to the user's login shell (the account's entry, else the window's
  `SHELL`, else `/bin/zsh`, so the installer writes the profile of the shell the owner logs into), and `USER`,
  `LOGNAME`, `TMPDIR` passed through. One run at a time, in a process group of its own, which Cancel ends with
  `SIGTERM`.

What it never does:

- It reads no release of its own: what is out is the engine's answer, and the update it offers on a press is
  [ADR-060](060-the-window-offers-the-engines-release-and-updates-on-a-press.md)'s. The installer alone fetches.
- It looks nowhere but those three places: no `PATH` lookup (a window opened from the Finder has launchd's), no
  `which`, no shell profile read.
- It never runs `station init`: the engine start alone, as the installer leaves the core.
- Nothing once the engine answers, but the update ADR-060 offers: the panel goes, and the window is the engine's
  client as before.

The page fetches nothing and the CSP is unchanged: the shell runs the programs, the page sends four commands.

The install line, the install folder, the binary's name and where the engine's record lives have one owner,
`packages/api/install.json`, which `make generate` projects: `src-tauri/build.rs` reads it into the crate, and the
page imports it (`apps/desktop/src/domain/core.ts`).

## Alternatives refused

- Showing the command only, with a Copy, and waiting for a terminal: the app would still not work on its own, and the
  owner asked that it does. The command stays on the panel, beside Install, for a Mac whose owner prefers a terminal.

## Consequences

- A debug build runs no installer and no `station` unless `STATION_WINDOW_CORE` names a folder holding a fake
  `install.sh` and `station`; it refuses with `core.not-run` otherwise, so no test, bench or capture reaches
  `spwn.sh` or a real core. A release build compiles that path out.
- The window now starts programs other than the opener's two, and its README says which.
- A debug build under `STATION_WINDOW_CORE` looks at the fake folder alone, whatever the engine's hello or record
  names, so a bench never starts a real `station`.
