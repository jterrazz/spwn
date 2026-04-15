package paths

// Directory layout constants.
const (
	SpwnBaseDir         = ".spwn"
	WorldsSubDir        = "worlds"
	AgentsSubDir        = "agents"
	StateFileName       = "state.json"
	// PluginsSubDir is the subdirectory for project-local plugins
	// (formerly "skills" + "tools" + "packages"; unified
	// post-migration). Kept under the "Skills" alias for
	// backwards-compatible symbol use from apps/cli/skill — the CLI
	// authors bare-markdown skills at <project>/spwn/plugins/<name>.md.
	PluginsSubDir       = "plugins"
	SkillsSubDir        = PluginsSubDir
	CredentialsSubDir   = "credentials"
	ActivityFileName    = "activity.jsonl"
	TeamsSubDir         = "teams"
	OrganizationsSubDir = "organizations"
)
