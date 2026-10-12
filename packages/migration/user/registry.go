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
	r.Register(SimplifyArchitectDir)
	r.Register(RestructureAgentDirs)
	r.Register(RenameHierarchiesToOrganizations)
	r.Register(EnsureDefaultOrganization)
	r.Register(RetireAgentKnowledge)
	r.Register(EnsureUserConfig)
	r.Register(DropLegacyStateJSON)
	return r.All()
}
