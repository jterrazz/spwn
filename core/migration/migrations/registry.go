package migrations

import "spwn.sh/core/migration"

// All returns every migration in order.
func All() []migration.Migration {
	r := migration.NewRegistry()
	r.Register(TierToRoleState)       // 001
	r.Register(TierToRoleProfiles)    // 002
	r.Register(EnsureDefaultHierarchy) // 003
	r.Register(WorkspaceToWorkspaces) // 004
	r.Register(RenameDefaultRoles)      // 005
	r.Register(OrgYAMLFieldRename)      // 006
	r.Register(ConsolidateJournal)      // 007
	r.Register(RemoveOrphanedUniverses)      // 008
	r.Register(MergeBlueprintIntoKnowledge) // 009
	return r.All()
}
