# packages/project

The spwn project manifest — parsing, scaffolding, discovery, validation.

## Role

A spwn project is a directory tree with a top-level `spwn.yaml`, one or more agents under `spwn/agents/`, and optional `spwn/tools/`, `spwn/skills/`, `spwn/hooks/`. This package owns the lifecycle of that tree: `spwn init` scaffolds it, `Find` walks up from any subdirectory to locate the nearest `spwn.yaml`, `Load` parses it, and `Validate` runs every static rule (refs exist, lockfile consistent, world/agent cross-refs resolve, no version conflicts across multi-agent worlds). Everything downstream — build, compile, spawn — starts from a `*Project` produced here.

## Key types

- `Project` — resolved project: root path, parsed manifest, agent records, world records. Returned by `Find` / `Load`.
- `Manifest` — parsed `spwn.yaml`: version, name, deps, worlds map.
- `Find(cwd) → *Project` — walk up from cwd looking for `spwn.yaml`. Returns `nil` when no project is present so callers can fall back to global mode.
- `Init(path, opts)` — scaffold a fresh project (manifest + starter agent + lockfile).
- `Validate(Input) → []Issue` — runs every rule (refs-exist, lockfile-consistent, version-conflict, runtime-supported, markdown-imports). Returns structured issues; CLI `spwn check` renders them.
- `Team`, `Organization`, `Role` + CRUD (`CreateTeam`, `ListTeams`, `CreateOrganization`, `ValidateOrganization`, `SetAgentTeam`, …) — teams group agents, organizations define role hierarchies. Stored as YAML under `~/.spwn/teams/` and `~/.spwn/organizations/` because they span multiple projects but are authored here.

## Related

- **Imported by** — `apps/cli`, `packages/transpile`
- **Imports** — `packages/agent`, `packages/dependency`, `packages/platform`, internal sub-packages (`manifest/`, `discovery/`, `scaffold/`, `validate/`, `resolve/`)
