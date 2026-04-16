package paths

// Directory layout constants.
const (
	SpwnBaseDir         = ".spwn"
	WorldsSubDir        = "worlds"
	AgentsSubDir        = "agents"
	StateFileName       = "state.json"
	// PacksSubDir is the subdirectory for project-local packs
	// (formerly "skills" + "tools" + "packages"; unified
	// post-migration). Kept under the "Skills" alias for
	// backwards-compatible symbol use from apps/cli/skill — the CLI
	// authors bare-markdown skills at <project>/spwn/packs/<name>.md.
	PacksSubDir       = "plugins"
	SkillsSubDir        = PacksSubDir
	CredentialsSubDir   = "credentials"
	ActivityFileName    = "activity.jsonl"
	TeamsSubDir         = "teams"
	OrganizationsSubDir = "organizations"
)
