# ADR-024: A project's folders sit at its root

**Status:** Proposed
**Date:** 2026-11-28

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

The owner's word, 2026-11-28, on whether a project's folders sit at the repository's root or under `spwn/`: "1 ta reco, on peut supprimer entierement l'ancien mode". The orchestrator's recommendation was the root, and this record writes it down.

## Context

A project is a `spwn.yaml`, found by walking up from the working folder; its name never came from a folder. Its agents, skills, tools, hooks, commands and profiles sat under one `spwn/` subfolder, so a repository that is itself a fleet of agents (the owner's `studio-fleet`) held `spwn/agents/` beside the `agents` it is about. Releases v0.20.0 and v0.21.0-alpha.1 to alpha.4 shipped that layout, with small download counts.

## Decision

1. **The folders sit at the root.** From the folder that holds `spwn.yaml`: `agents/<name>/`, `skills/`, `tools/<name>/tool.yaml`, `hooks/<name>.yaml`, `commands/<name>.md` and `profiles/`. One type of `packages/project`, `Layout`, owns these names; every loader of the root module takes its folder from it, through `platform.Folders`, and Machines imports it.
2. **`folders:` renames one kind's folder.** A key of `spwn.yaml`'s `folders:` is a kind, its value the slash path of that kind's folder from the root: lowercase letters, digits, `-` and `_` in each name, never `.`, `..`, a leading dot (so never `.git` or `.spwn`), an absolute path, `~`, a colon, a control character or `spwn/`; no two kinds share a folder or sit one under the other. A refused entry is `spwn-yaml.folders`, naming its key; Machines calls it `project.folders-invalid`.
3. **The old layout is refused, never read.** Before `spwn.yaml` is read, a `spwn` at the root that is a folder holding one of `agents`, `skills`, `tools`, `hooks`, `commands`, `profiles` or `hooks.yaml`, or a `spwn` that is a link of any kind, which is never followed, refuses the project with `project.old-layout` and one sentence: "spwn/ is the old project layout: its folders now sit at the project's root. Move each one up (mv spwn/agents agents, and the same for skills, tools, hooks, commands, profiles), point knowledge: at its new path, change any .gitignore line naming spwn/, then run again." When a kind's folder sits both under `spwn/` and at the root, the refusal names those folders and says to merge them. A file named `spwn`, or a `spwn/` folder holding none of those names, is not the old layout. There is no alias, no fallback and no second mode.
4. **A `spwn.yaml` that does not load stops every command.** The old layout, a refused `folders:` entry, YAML that does not parse (`spwn-yaml.parse`): each stops the command with its error and its code, before it reads a folder, writes under the home or starts a world. Only a folder with no `spwn.yaml` on the way up runs without a project.
5. **A skill is named, never inferred from where a file sits.** The six names are common words, so a repository's root may already use one for its own purpose: an `agents/` of documentation, a `tools/` of scripts, a `hooks/` of git hooks, a `commands/` of shell snippets, a `profiles/` of anything, a `skills/` of notes. Of the six, skills were the one kind loaded without being named: every `skills/*.md` and every `skills/<d>/` holding a `SKILL.md` became a prompt of every agent. So a folder under `skills/` is a skill only when its `SKILL.md` opens with front matter that parses and carries a `name:`, and a bare `skills/<name>.md`, which the docs present as a supported form, only when its own front matter does and it is not a `README.md`; anything else is passed over in silence. A named skill ships every file of its folder, at most 64 entries and 256 KiB a file, none a link, or it is refused as `project.skill-refused`, naming the entry. One function, `project.LoadSkills`, serves both binaries. A repository whose root uses one of the six names for something else that a kind would still read declares `folders:` to move that kind elsewhere.
6. **What does not move.** A world's knowledge stays a path of `spwn.yaml` (`knowledge: ./knowledge` by default); the runtime state stays in `.spwn/`; the user's named world configs stay in `~/.spwn/worlds/`, whatever project is active.

## Consequences

- A project of a released version stops at its first command with the sentence above, and the release that carries this is a breaking minor, v0.22.0, whose notes carry the sentence.
- Machines' rooted walk of a project visits `spwn.yaml`, `spwn.lock`, then the folders of the layout alone; nothing else at the root is walked ([machines ADR-112](../../machines/docs/decisions/112-the-declaration-is-spwn-yaml-spwn-lock-and-the-folders-its-layout-names.md)).
- The sibling repository spwn-web is itself an old-layout project and moves after the tag.
