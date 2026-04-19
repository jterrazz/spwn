# packages/platform

Paths, IDs, and host-platform constants.

## Role

The foundation layer: every other package asks `platform` for "where does this thing live on disk?" and "what should I call a new world/agent?". Centralises `~/.spwn/` layout, container-side paths, random name/ID generation, and mind-layer definitions. Zero external dependencies — everything that might need these primitives imports from here, and `platform` never imports any other spwn package, so it's import-cycle-proof.

## Key types

- `BaseDir()`, `AgentsDir()`, `WorldsDir()`, `CredentialsDir()`, `SkillsDir()`, … — host-side directory resolvers. Respect `SPWN_HOME`, default to `~/.spwn`.
- `ProjectRoot()`, `SetProjectRoot(path)` — optional project mode; set by the CLI at startup when a `spwn.yaml` is discovered.
- `ArchitectContainerName()`, `ArchitectImage`, `ArchitectImageVersion` — constants for the always-on Architect container.
- `GenerateWorldID(name)` → `world-<slug>-<5-hex>`, `GenerateAgentID(name)` → `agent-<name>-<5-hex>`.
- `RandomPlanetName()`, `RandomAgentName()` — bundled name lists for scaffolds.
- `MindLayers` — canonical ordered list of the two mind-layer directories (playbooks, journal). Identity was collapsed into a single SOUL.md at the agent root; knowledge is world-scoped, not agent-scoped; skills moved to build-time dependencies at `/world/skills/`.

## Related

- **Imported by** — `apps/api`, `apps/cli`, `catalog`, `packages/activity`, `packages/agent`, `packages/architect`, `packages/auth`, `packages/migration`, `packages/upgrade`, plus most other packages
- **Imports** — stdlib only
