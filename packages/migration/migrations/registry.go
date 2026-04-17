package migrations

import "spwn.sh/packages/migration"

// All returns every migration in order.
func All() []migration.Migration {
	r := migration.NewRegistry()
	r.Register(TierToRoleState)       // 001
	r.Register(TierToRoleProfiles)    // 002
	r.Register(EnsureDefaultHierarchy) // 003
	r.Register(WorkspaceToWorkspaces) // 004
	r.Register(RenameDefaultRoles)      // 005
	r.Register(ConsolidateJournal)      // 007
	r.Register(RemoveOrphanedUniverses)      // 008
	r.Register(SimplifyArchitectDir)       // 010
	r.Register(RestructureAgentDirs)              // 011
	r.Register(RenameHierarchiesToOrganizations) // 012
	r.Register(EnsureDefaultOrganization)        // 013
	r.Register(RetireAgentKnowledge)             // 014
	r.Register(EnsureUserConfig)                 // 015
	return r.All()
}
