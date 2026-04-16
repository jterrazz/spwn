# packages/upgrade

On-disk state migrations + CLI version updates.

## Role

Two responsibilities behind one package: (1) migrations that advance the `~/.spwn/` directory layout forward as the CLI evolves — numbered scripts applied in strict order, tracked in a registry, with backup-before-apply and applied-state persisted so rerunning is idempotent; (2) the `spwn upgrade` self-update path, which downloads and installs a newer CLI binary. Both live together because "upgrade" from the user's perspective is one action: advance the local install to the latest version of both the binary and the state it depends on.

## Key types

- `Migration` — one numbered step: `Number`, `Description`, `Apply(baseDir) error`. Registered via `migrations.init()`.
- `Registry` — ordered, collision-checked list of migrations. `Register` panics on out-of-order or duplicate numbers.
- `Runner` — applies pending migrations against a base dir, persists the applied state to `~/.spwn/state.json`, skips already-applied ones.
- `BackupBaseDir(baseDir)` — tar the whole `~/.spwn/` (minus `backups/`) into `backups/` before migration runs.
- `update/` sub-package — `spwn upgrade` logic: version check against GitHub releases, binary download, atomic replace.

## Related

- **Imported by** — `apps/api`, `apps/cli`
- **Imports** — `packages/platform`, internal sub-packages (`migrations/`, `update/`)
