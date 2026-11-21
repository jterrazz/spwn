# ADR-073: Spwn.app carries the engine on a Mac with a screen; install.sh is the headless install

**Status:** Proposed
**Date:** 2026-11-21

## Context

macOS lists the product several times on one Mac, and the owner wants it listed once (2026-11-21). Login Items shows "Spwn" and the owner's name; Local Network shows five lowercase "station" rows and one "Station"; the Dock shows two apps. The engine must stay usable without the window, so one engine binary needs two install modes, exactly one active per Mac. The design and its lots are the item "One Spwn on a Mac" in [`TODO.md`](../../TODO.md).

Before any code, lot L0 measured what macOS does, on two hosts:

- a disposable clone of the Xcode image on development-macos (macOS 26.6.2), with `sfltool dumpbtm` as its admin;
- the old Mac mini test bench (macOS 27.0.1), whose System Settings were driven and read with agent-desktop.

The probe was a tiny ad hoc signed bundle: one Swift binary as the main executable and again as `Contents/MacOS/station`, two `BundleProgram` agents in `Contents/Library/LaunchAgents`, a login item in `Contents/Library/LoginItems`. Its LAN probe is a `connect()` to another host of the mini's LAN, every 2 s: `errno 65` (No route to host) means Local Network refused, `connected` means allowed. No Developer ID was used, since no secret is handled by an agent. So every Local Network fact below is about ad hoc code, which [TN3179](https://developer.apple.com/documentation/technotes/tn3179-understanding-local-network-privacy) calls unreliable for this privacy check.

### What Local Network keeps

The rules are readable without root, in `/Library/Preferences/com.apple.networkextension.plist`. That file is an NSKeyedArchiver plist holding one `com.apple.preferences.networkprivacy-<user>` configuration per user, and each configuration holds `NEPathRule`s.

- **An app** is one rule, keyed by its bundle id, with no path. An installed app gets a hidden placeholder (`DenyMulticast` true, no preference) and becomes a row only once it is decided. The row is gone from the list once the app is deleted.
- **A bare executable** is one rule keyed by its path and its signing identifier. The row is labelled with the file's name, and it stays after the file is deleted, with a blank icon.
- **No rule records a designated requirement**: `DesignatedRequirement` is empty and `AllowEmptyDesignatedRequirement` true on every rule, Developer ID apps included.

### H1 and H2, the rows on the mini

- **H1, proved.** On the mini, each lowercase "station" row is one (path, signing identifier) pair:
    - `~/.local/bin/station` with `a.out`, the identifier the Go linker gives every ad hoc build;
    - `~/.local/bin/station` with `station-darwin-arm64`, the identifier the Developer ID release carries, since it is signed under its artifact's name;
    - three e2e run folders, deleted since, with one of the two identifiers each.
- **H2, half proved.** The one "Station" row is the rule of `com.jterrazz.station.menubar`, allowed and with no path. Which process raised it is not settled. An ad hoc bare agent whose plist names an installed app in `AssociatedBundleIdentifiers` was still asked about as "station" and given its own path rule, so the key did not move the attribution there. Ad hoc code has no team, and the key's association needs a team match, so the Developer ID case stays a HYPOTHESIS (2026-11-21). The read that settles it: a Developer-ID engine with the key, on a Mac where neither app has a Local Network rule yet.

### The mode A hypotheses

- **HA1, proved** on both hosts. `Contents/MacOS/station` registers `SMAppService.agent(plistName:)`, and BTM records the job under the app (`Parent Identifier: 2.<bundle id>`). It works when started through a `~/.local/bin` link too, although Foundation's `Bundle.main` then names the link's folder: an engine that picks its mode must resolve the link itself. The nested binary can also register the nested login item (`SMAppService.loginItem`). The probe was Swift; the engine calls it through cgo, which L3 proves.
- **HA2, proved.** Over SSH, while the user is logged in at the screen, the register works. For a user with no GUI session it throws `SMAppServiceErrorDomain` 125, "Domain does not support specified action". `status` then reads `enabled` although launchd holds no job, so the engine judges the job by launchd, never by `status`.
- **HA4, proved** on development-macos:
    - `launchctl kickstart -k` of a registered label restarts it, and `KeepAlive` restarts it after a kill;
    - `bootout` removes the job while `status` stays `enabled`, and `launchctl bootstrap` of the bundled plist fails with error 5, since `BundleProgram` belongs to SMAppService alone;
    - `register()` submits a booted-out job again.
- **HA5, from the source alone.** In tauri-cli 2.12.4 (`crates/tauri-bundler/src/bundle/macos/app.rs`), `externalBin` is copied into `Contents/MacOS`, and `bundle.macOS.files` copies any file or folder anywhere under `Contents`. Tauri signs neither unless Tauri signs the app, and this repository signs with `scripts/sign-macos.sh`, whose nested pass finds `.app`, `.framework`, `.dylib`, `.xpc` and `.appex` but no bare Mach-O. No bundle was built with them; L2 builds one.

### Mode A refutes the one-row hope

- **What mode A shows.**
    - Login Items reads the app's name, under Background App Activity, and its notification says "L0A can run in the background".
    - Local Network asks twice for one agent: "Allow “L0A”…", which allows the app's rule, and "Allow “station”…", which writes a path rule for `Contents/MacOS/station` under its own signing identifier.
    - The agent reached the LAN only after the second Allow. launchd names the agent itself as its responsible process (`launchctl procinfo`: `responsible path = …/Contents/MacOS/station`), so TN3179's "for the whole app" holds for a helper the app starts, not for an agent SMAppService starts.
    - So mode A shows two rows, the app's and the nested binary's.
- **What mode B shows.** Login Items reads "station" for an ad hoc job; a Developer-ID job reads the certificate's organisation name (6dce2c34d). Local Network shows one "station" row per (path, identifier).
- **HA3, refuted for ad hoc** on the mini. The registered bundle was replaced by a new build, with new UUIDs and a new default nested identifier `station-<uuid>`, and then:
    - `kickstart -k` failed with `OS_REASON_CODESIGNING`. On development-macos, macOS 26.6.2, the same kickstart ran the new binary;
    - `register()` again, and `bootout` then `register()`, still failed;
    - `unregister()` then `register()` ran it once, but failed after a second replacement whose nested binary was signed with a stable identifier, three times;
    - the updated agent was refused the LAN, with no new alert and no new rule in 30 s.

    A Developer-ID update, whose designated requirement survives releases, is a HYPOTHESIS (2026-11-21), settled by two signed builds of L2.

- **One grant across the two modes: no, by path.** A bare binary's grant is its path's, and the bundled agent is another path, so the B to A move asks again.
- **One grant across the two modes: yes, by UUID, once.** A bare binary whose Mach-O UUID equals an executable of an app LaunchServices knows was asked about under the app's name, "Allow “L0Probe”…", and that Allow set the app's rule. Whether it then reached the LAN was not read.
- **The LAN's default gateway was reached** by an agent that had no Local Network rule at all. That was one host, read once.

## Decision

On a Mac with a screen, Spwn.app carries the engine, and install.sh stays the headless install:

- **Mode A, a Mac with Spwn.app.** `Contents/MacOS/station` is the same engine binary install.sh ships. It is started by two `BundleProgram` agents in `Contents/Library/LaunchAgents`, which SMAppService registers from the engine itself. The menu bar is a nested login item, and `~/.local/bin` holds links into the bundle.
- **Mode B, a Mac without the app.** The bare binary and its two `ProgramArguments` plists stay, without `AssociatedBundleIdentifiers`.
- **Exactly one mode per Mac.** The app's first start moves a Mac from B to A, and keeps nothing.
- **The nested engine is signed with one stable identifier, never a default.** The identifier is `com.jterrazz.station`, and it is set by `scripts/sign-macos.sh` (L1), so the nested binary's Local Network rule survives releases, if a Developer ID behaves as the rules' empty requirement suggests.
- **An update restarts the agents by `unregister()` then `register()`.** `kickstart -k` is never trusted across a replaced binary.
- **The engine judges its job by launchd.** `launchctl print gui/<uid>/<label>` is read, never `SMAppService.status`.

What this changes in earlier records:

- **[ADR-058](058-the-window-installs-and-starts-the-core-itself.md)**: in mode A the panel's Install becomes Start.
- **[ADR-060](060-the-window-offers-the-engines-release-and-updates-on-a-press.md)**: in mode A the engine updates with the window, through the Tauri updater ([ADR-065](065-the-window-updates-itself-through-the-tauri-updater.md)), so there is one update.

## Consequences

- Login Items lists one "Spwn" in mode A, with the two agents and the menu bar under it.
- Local Network lists two rows in mode A, "Spwn" and "station", each asked once, as measured with ad hoc code. One row is out of reach with an agent SMAppService starts. Only an engine started by the app itself would give one, and that would tie the engine to the window, which the owner refused.
- The B to A move asks for Local Network once more, and leaves the bare "station" row. macOS 27.2's (-) button removes it, and the window's migration panel says so once.
- Every release must keep the nested identifier, the bundle id and the team. A change of any one is a new row and a new question.
- An agent registered with no GUI session is not running, although `status` reads `enabled`. Mode A needs a logged-in user, as mode B's `launchctl bootstrap gui/<uid>` already did.
- Before L3 lands, L2 owes one proof with two Developer-ID builds (a `workflow_dispatch` on main): an update keeps the job starting, and keeps the nested binary's Local Network grant. If either fails, this record goes back to the owner before any engine code moves.
