// Package migration advances the ~/.spwn/ directory layout as the
// CLI evolves.
//
// Each schema change is a numbered step with an
// Apply(ctx, baseDir) error function. The Registry enforces strict
// ordering (no duplicates, no out-of-order numbers — both are
// programmer errors that panic at init-time). The Runner applies
// every pending migration against a base dir, persists the
// applied state to ~/.spwn/state.json, and skips already-applied
// ones so rerunning is idempotent.
//
// A full backup of ~/.spwn/ (excluding backups/) is taken before
// any migration runs; failed migrations roll back from the backup.
//
// The built-in migration catalogue lives in the migrations/
// sub-package — one file per numbered step.
package migration
