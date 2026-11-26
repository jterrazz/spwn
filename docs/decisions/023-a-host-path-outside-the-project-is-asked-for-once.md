# ADR-023: A host path outside the project is asked for once

**Status:** Proposed
**Date:** 2026-11-26

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

Taken by delegation on 2026-11-26: the orchestrator of the security verification of `fix/rooted-reads-are-pinned` made the call, and this record writes it down. The same day, the orchestrator of the third security verification of `fix/project-stays-inside` revised it by delegation: the names and paths checked before the ask, nothing outside touched before it, a yes kept only after the run succeeded, and `--yes` for one run, never kept.

## Context

`spwn up` is run in a project folder that may be a cloned repository, so its `spwn.yaml` is the repository author's, not the user's. A world's `workspaces:` and `knowledge:` named any host path, and an fs automation's watch path any host folder: `workspaces: [../..]` mounted the user's home read-write into a world driven by the project's own prompts, with nothing checked or asked. Plan already refused a `knowledge:` path that leaves the project, but `spwn up` never ran that rule. Refusing every outside path would break honest projects that mount a sibling repository on purpose.

## Decision

1. **A path that stays inside the project needs nothing.** Each workspace, the knowledge folder and each fs watch path is resolved against the project's root, by name and through the links the project holds (each read inside the project, never followed by the system), and used by that resolved path, with the name and access it was asked with, carried as values, never re-parsed.
2. **A path that leaves the project is never used silently, nor touched before the yes.** One that is absolute elsewhere, climbs by `..`, or is reached through a project link pointing outside is shown once, quoted and absolute as far as the project's links tell, with what it grants (a read-write or read-only mount into which world, or a watched folder whose files' names and times are recorded in the receipts and passed to the project's agent), and asked on a terminal. Nothing outside the project is stat'd, read or opened before the answer.
3. **The prompt cannot be written by the project.** Every world, agent and workspace name and every host path of spwn.yaml is checked before any reach is built: a control character refuses it (`spwn-yaml.control-character`), a workspace name must be a slug, and a path starting with `~` is refused (`spwn-yaml.tilde`), never expanded. The ask and the refusal print paths quoted.
4. **A typed yes is the user's, kept outside the project, and only once the run succeeded.** It is remembered in `~/.spwn/grants.yaml`, keyed by the project folder's resolved path, the path and the access, so the next run does not ask; another project folder, another path or another access is asked again, and a read-only yes does not serve a read-write ask. A run that fails after the yes keeps nothing.
5. **Off a terminal, no kept yes refuses; `--yes` allows for that run only.** The command fails with `outside.not-granted`, naming each path and what it grants, before anything runs. `--yes` on `spwn up` (and on `spwn automation daemon`) answers the ask for that run, prints each path it allowed and what it grants on stderr (`allowed for this run: …`) so a CI log shows it, and keeps nothing. One flag, no alias, the root grammar's `--yes` on an action that asks.
6. **A path given on the command line is the user's.** A workspace passed with `-w` is not asked.

## Consequences

- An honest project with everything inside it sees no change; one that mounts a sibling folder is asked once per path.
- A CI run of a project with outside paths needs `--yes` on every run, or a grant kept by a typed yes; a README line telling to run `--yes` grants nothing beyond that run.
- The grant is per path, not per world: a second world mounting the same granted path with the same access is not asked again.
- The grant is tied to the project's FOLDER path, not its content: a different repository later cloned or moved into the same folder inherits it, and so do later edits of the same project. `~/.spwn/grants.yaml` lists every kept grant; it is revoked by editing the file and deleting its entry (no command does it).
- The path shown for an outside link is where the project's link points, not where that outside path itself resolves: the user is shown the path the project names, and the bind follows it as the system does, after the yes.
- This closes the mounts, not the trust: every world still receives the user's credentials read-only and the gh cache read-write, so running a cloned project's agents trusts that project ([Worlds](../concepts/worlds.md)).
- The check is made on a static tree; a path swapped by a racing process between the resolve and the bind is out of the model.
