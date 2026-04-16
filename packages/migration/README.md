# packages/migration

On-disk state migrations for ~/.spwn.

## Role

Advances the `~/.spwn/` directory layout as the CLI evolves. Each schema change is a numbered step with an `Apply(baseDir) error` function; the registry enforces strict ordering (no duplicates, no out-of-order numbers) and the runner persists applied state to `~/.spwn/state.json` so rerunning is idempotent. A full backup of `~/.spwn/` (minus `backups/`) is taken before any migration runs. Split from the former `packages/upgrade` package so state migration and CLI self-update (`packages/update`) are independently composable.

## Key types

- `Migration` — one numbered step: `Number`, `Description`, `Apply(baseDir) error`.
- `Registry` / `NewRegistry` — ordered, collision-checked list. `Register` panics on duplicate or out-of-order numbers.
- `Runner` / `NewRunner(baseDir, migrations)` — applies every pending migration against a base dir, persists state, skips already-applied.
- `BackupBaseDir(baseDir)` — tar the whole `~/.spwn/` (excluding `backups/`) into `backups/` before migration runs.
- `migrations/` sub-package — the built-in migration catalogue: drop in a new `NNN_<name>.go` and register it in `migrations.All()`.

## Related

- **Imported by** — `apps/cli` (via `spwn migrate` + boot-time runMigrations)
- **Imports** — `packages/platform`
