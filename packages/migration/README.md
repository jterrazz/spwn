# packages/migration

On-disk state migrations.

## Role

Schema migrations for every spwn-owned directory on the host. Each
schema change is a numbered `Migration{Number, Description, Apply}`
function; the `Registry` enforces strict ordering; the `Runner`
persists applied state to a version file in the target root and
skips already-applied migrations. A full tar backup is taken before
the first pending migration runs so failed migrations can roll back.

## Scopes (mental model)

Spwn state lives in four places. Each has its own lifecycle and
deserves its own migration category. Only the first is wired today;
the others are placeholders for when a real schema change lands.

| Scope | Root | What lives there | Status |
|---|---|---|---|
| **user** | `~/.spwn/` | config.yaml, organizations/, credentials/, activity.jsonl, world-states/ | live — `user/` sub-package |
| project | `./spwn.yaml` + `./spwn/` | manifest, agents, skills, tools, hooks, knowledge | not populated |
| agent | `./spwn/agents/<name>/` | SOUL.md, agent.yaml, playbooks/, journal/ | not populated |
| world | `~/.spwn/world-states/<id>/` | runtime.json, manifest snapshot, roster | not populated |

The `Runner` + `Migration` types are scope-agnostic. Each category
just needs its own root dir and version file; the mechanics are the
same. When a non-user schema change first lands, create
`packages/migration/<scope>/` beside `user/` and wire a second runner
at the right trigger point (CLI boot, project load, agent load, world
spawn).

## Key types

- `Migration` — one numbered step: `Number`, `Description`, `Apply(ctx, baseDir) error`.
- `Registry` / `NewRegistry` — ordered, collision-checked list. `Register` panics on duplicate or out-of-order numbers.
- `Runner` / `NewRunner(baseDir, migrations)` — applies every pending migration against a base dir, persists state, skips already-applied.
- `BackupBaseDir(baseDir)` — tar the whole base dir (excluding `backups/`) into `backups/` before migration runs.
- `user/` sub-package — the built-in user-scope migration catalogue. Drop in a new `NNN_<name>.go` and register it in `user.All()`.

## Testing pattern

Each scope's sub-package has its own `testdata/` tree that mirrors
the migrations catalogue:

```
packages/migration/<scope>/testdata/<NNN_migration_name>/
  before/   — seeded into t.TempDir() before Apply runs
  after/    — expected state of the tempdir once Apply returns
```

A shared `runFixture(t, m, "NNN_migration_name")` helper (in each
scope's `harness_test.go`) copies `before/` into a fresh temp dir,
runs `m.Apply`, then diffs the temp dir against `after/`
file-by-file. Any missing/extra path or content mismatch fails the
test with the path reported.

`.gitkeep` markers in `after/` are filtered by the harness so
fixtures can preserve otherwise-empty directories in git without
breaking the diff. Migrations whose output is not byte-deterministic
(YAML round-trips, timestamp-suffixed renames) keep their happy-path
tests inline; edge-case tests (no-op paths, idempotency, user-edit
preservation) stay inline across the board. Fixtures express the
*transformation contract*; inline tests express the *behavioural
invariants*.

## History

Migrations 001-008 were deleted in the pre-1.0 migration squash.
They targeted file formats spwn no longer emits (legacy `state.json`,
pre-SOUL.md agent profiles, `universes/`) and silently no-op'd on
any install that had already passed through the labels-as-truth cut.
Live migrations start at 010.

## Related

- **Imported by** — `apps/cli` (via `spwn migrate` + boot-time runMigrations)
- **Imports** — `packages/platform`
