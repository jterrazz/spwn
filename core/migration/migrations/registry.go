package migrations

import "spwn.sh/core/migration"

// All returns every migration in order.
func All() []migration.Migration {
	r := migration.NewRegistry()
	r.Register(TierToRoleState)       // 001
	r.Register(TierToRoleProfiles)    // 002
	r.Register(EnsureDefaultHierarchy) // 003
	r.Register(WorkspaceToWorkspaces) // 004
	return r.All()
}
