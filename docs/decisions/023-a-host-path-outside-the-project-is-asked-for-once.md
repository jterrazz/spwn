# ADR-023: A host path outside the project is asked for once

**Status:** Proposed
**Date:** 2026-11-26

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

Taken by delegation on 2026-11-26: the orchestrator of the security verification of `fix/rooted-reads-are-pinned` made the call, and this record writes it down.

## Context

`spwn up` is run in a project folder that may be a cloned repository, so its `spwn.yaml` is the repository author's, not the user's. A world's `workspaces:` and `knowledge:` named any host path, and an fs automation's watch path any host folder: `workspaces: [../..]` mounted the user's home read-write into a world driven by the project's own prompts, with nothing checked or asked. Plan already refused a `knowledge:` path that leaves the project, but `spwn up` never ran that rule. Refusing every outside path would break honest projects that mount a sibling repository on purpose.

## Decision

1. **A path that stays inside the project needs nothing.** Each workspace, the knowledge folder and each fs watch path is resolved against the project's root, through its links, and used by that resolved path.
2. **A path that leaves the project is never used silently.** One that is absolute elsewhere, climbs by `..`, or resolves outside through a link (a dangling link included) is shown once, resolved and absolute, with what it grants (a read-write or read-only mount into which world, or the names and times of a watched folder listed), and asked on a terminal.
3. **The yes is the user's, kept outside the project.** It is remembered in `~/.spwn/grants.yaml`, keyed by the project's resolved root, the resolved path and the access, so the next run does not ask; another project, another path or another access is asked again. A read-write workspace keeps its access once granted, and the ask says read-write.
4. **Off a terminal, no kept yes refuses.** The command fails with `outside.not-granted`, naming each path and what it grants, before anything runs; `--yes` on `spwn up` (and on `spwn automation daemon`) answers the ask as a typed yes does, and is remembered the same way. One flag, no alias, the root grammar's `--yes` on an action that asks.
5. **A path given on the command line is the user's.** A workspace passed with `-w` is not asked.

## Consequences

- An honest project with everything inside it sees no change; one that mounts a sibling folder is asked once per path.
- A CI run of a project with outside paths needs `--yes` once, or a kept grant.
- The grant is per path, not per world: a second world mounting the same granted path with the same access is not asked again.
- This closes the mounts, not the trust: every world still receives the user's credentials read-only and the gh cache read-write, so running a cloned project's agents trusts that project ([Worlds](../concepts/worlds.md)).
- The check is made on a static tree; a path swapped by a racing process between the resolve and the bind is out of the model.
