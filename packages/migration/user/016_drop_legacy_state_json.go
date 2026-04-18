package user

import (
	"context"
	"os"
	"path/filepath"

	"spwn.sh/packages/migration"
)

// DropLegacyStateJSON removes the pre-labels-as-truth ~/.spwn/state.json
// and the short-lived ~/.spwn/runtime/ directory that a mid-refactor
// version produced. Both are superseded by Docker-label enumeration +
// per-world runtime.json files under ~/.spwn/world-states/.
//
// Before this migration, runtimestate.NewStore() performed the same
// eviction as a side effect on every CLI boot — a parallel migration
// system. Folded in here so there's one source of truth for schema
// transitions and version.json tracks this alongside every other.
var DropLegacyStateJSON = migration.Migration{
	Number:      16,
	Description: "drop legacy ~/.spwn/state.json + ~/.spwn/runtime/ (superseded by labels-as-truth)",
	Apply: func(_ context.Context, baseDir string) error {
		// All three removals are best-effort: a fresh install has
		// neither, and an already-evicted install (thanks to the old
		// side-effect path) also has neither. Either way this is a
		// no-op.
		_ = os.Remove(filepath.Join(baseDir, "state.json"))
		_ = os.Remove(filepath.Join(baseDir, "state.json.bak"))
		_ = os.RemoveAll(filepath.Join(baseDir, "runtime"))
		return nil
	},
}
