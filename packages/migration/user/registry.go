package user

import "spwn.sh/packages/migration"

// All returns every migration in order.
//
// Migrations 001-008 used to live here; they targeted file formats
// spwn no longer emits (state.json, pre-SOUL.md agent profiles,
// universes/) and were deleted in the pre-1.0 migration squash. Any
// install that was at version 15 or higher continues working
// unchanged; pre-version-9 installs do not exist in the wild.
func All() []migration.Migration {
	r := migration.NewRegistry()
	r.Register(SimplifyArchitectDir)             // 010
	r.Register(RestructureAgentDirs)             // 011
	r.Register(RenameHierarchiesToOrganizations) // 012
	r.Register(EnsureDefaultOrganization)        // 013
	r.Register(RetireAgentKnowledge)             // 014
	r.Register(EnsureUserConfig)                 // 015
	r.Register(DropLegacyStateJSON)              // 016
	return r.All()
}
