# ADR-055: Station reads only its current state, and retires no older one

**Status:** Proposed
**Date:** 2026-11-18

## Context

Each round of station left code that moved a Mac from the round before. `station init` moved
`~/.studio/` to the XDG folders ([ADR-007](007-files-live-in-the-xdg-folders.md)). It moved a v2
`settings.json` to `preferences.json`, and the preferences read the old group names `here`,
`hosts` and `services` ([ADR-015](015-preferences-json.md)). It stopped the userspace tailscaled
and deleted its keychain accounts ([ADR-010](010-vms-are-reached-through-their-mac.md)), and it
cleaned the studio-workstation lines out of `~/.zshrc`. `station engine start` retired v2's
`com.jterrazz.station.daemon` job and round 3's per-VM agents, and `station daemon` and
`station machines supervise` stayed as hidden commands for them
([ADR-011](011-one-engine-per-mac.md)). Each store converted the records an older station wrote:
the ledger ([ADR-034](034-apply-carries-a-machinefile-the-ledger-bounds-removal.md)), and the VM
records written in v2's lifetime words ([ADR-032](032-the-words-of-a-vms-lifetime-and-network.md)).
`install.sh` deleted `~/.studio/bin/workstation`.

None of this runs on a fresh machine, yet every spec and every change paid for it. On 2026-11-18
the owner ruled: no legacy in the tests or the app, fresh state only, and a host with old
leftovers is cleaned by hand, never by station. Asked about the stores, the owner kept the
version stamp and its refusal, and dropped the converters. Asked about the third-party files
`init` relocates (bun, go, mise, npmrc, tart), the owner kept them: they are what a fresh machine
holds by default.

## Decision

- **Station reads only the layout and the records it writes today.** No step of `init`,
  `engine start` or `install.sh` moves, converts, stops or deletes what an earlier station or
  the studio-workstation era left: `~/.studio/`, `settings.json`, the old group names, the
  userspace tailscaled, its keychain accounts, the old launchd jobs, the old `~/.zshrc` lines.
- **A store keeps its version and refuses every other one.** A record written at another
  version than the one this station knows is refused, never read nor rewritten: a newer one as
  before, an older one with the hint to move the file away. No store converts.
- **The hidden commands that served an old job go.** `station daemon` and
  `station machines supervise` are no commands.
- **Leftovers are the owner's.** A host that still has them is cleaned by hand.
- **Out of this decision:** the `migrate/alias` words and the `status --json` contract of
  [ADR-004](004-station-status-json-is-a-contract.md), which are a separate decision, and the relocation
  of third-party files, which stays.

This amends ADR-007, ADR-010, ADR-011, ADR-015, ADR-032 (its record half) and ADR-034 where they
say that station migrates, converts or retires; their other decisions stand.

## Consequences

- `init` has no `layout` row, and the setup check none either.
- A Mac upgraded from an earlier round keeps its leftovers until the owner removes them; station
  neither reads them nor names them.
- A record an older station wrote stops the command that reads it, with its path; the owner
  moves it away.
- A new round that changes a layout or a record writes no converter: it ships the new form, and
  the owner cleans the old one.
