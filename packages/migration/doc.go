// Package migration advances spwn's on-disk schema as the CLI evolves.
//
// Each schema change is a numbered step with an Apply(ctx, baseDir)
// error function. The Registry enforces strict ordering (no duplicates,
// no out-of-order numbers — both are programmer errors that panic at
// init-time). The Runner applies every pending migration against a
// base dir, persists the applied state to <baseDir>/version.json, and
// skips already-applied migrations so rerunning is idempotent.
//
// A full backup of the base dir (excluding backups/) is taken before
// any migration runs; failed migrations roll back from the backup.
//
// Scopes. Spwn state lives in four places — ~/.spwn/ (user),
// ./spwn.yaml + ./spwn/ (project), per-agent directories, per-world
// directories under world-states/. Each scope deserves its own
// migration category because the triggers (CLI boot, project load,
// agent load, world spawn) differ. The Runner + Migration types here
// are scope-agnostic: any scope just drops migrations into its own
// sub-package and wires a runner at the appropriate trigger point.
//
// Today only the user scope is populated — see the user/ sub-package
// for the migration catalogue that advances ~/.spwn/.
package migration
