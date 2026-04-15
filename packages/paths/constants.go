package paths

// Directory layout constants.
const (
	SpwnBaseDir         = ".spwn"
	WorldsSubDir        = "worlds"
	AgentsSubDir        = "agents"
	StateFileName       = "state.json"
	// SkillsSubDir is the subdirectory for project-local packages
	// (formerly "skills" + "tools"; unified post-migration). Kept
	// under the "Skills" name for backwards-compatible symbol use
	// from apps/cli/skill — the CLI authors bare-markdown skills
	// at <project>/spwn/packages/<name>.md.
	SkillsSubDir        = "packages"
	CredentialsSubDir   = "credentials"
	ActivityFileName    = "activity.jsonl"
	TeamsSubDir         = "teams"
	OrganizationsSubDir = "organizations"
)
