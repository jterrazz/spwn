# packages/agent

The agent domain — manifests, minds, journals, sessions, evolution.

## Role

Owns everything an "agent" is on disk. The composition manifest (`agent.yaml`) and the mind layers (`identity/`, `skills/`, `knowledge/`, `playbooks/`, `journal/`) all live here. CLI verbs like `spwn agent create`, `spwn agent add`, and `spwn agent dream` all ultimately route through this package. The agent is provider-neutral at rest — runtime-specific translation happens later, in `packages/compile`.

## Key types

- `Manifest` — parsed `agent.yaml`: `Name`, `Role`, `Team`, `Runtime` config, `Deps` list. `LoadManifest` / `SaveManifest` / `AddDependency` / `RemoveDependency` for CLI-level edits.
- `Info` — summary view of an agent's on-disk state (dir, layers present, journal length). Populated by `InspectAgent(name)`.
- `InitMind` / `DeleteAgent` / `ListAgents` / `ValidateMind` / `RepairMind` — lifecycle verbs over the mind tree.
- `Session` / `LoadSession` / `SaveSession` — per-world conversation state, used to drive runtime `--resume`.
- `Dream` / `Reflect` / `Sleep` / `Fork` — evolution operations (promote playbooks, consolidate memory, clone agents).
- `ExportMind` / `ImportMind` — tar.gz export for sharing an agent across machines.

## Related

- **Imported by** — `apps/api`, `apps/cli`, `packages/architect`, `packages/world`
- **Imports** — `packages/platform` (paths), internal sub-packages (`mind/`, `journal/`, `session/`, `evolution/`, `memory/`)
