package foundation

import (
	"fmt"
	"os"
	"path/filepath"
)

// BaseDir returns the path to ~/.spwn/.
// If SPWN_HOME is set, it overrides the default (used for test isolation).
// Falls back to UNIVERSE_HOME (deprecated) for backward compatibility.
func BaseDir() string {
	if dir := os.Getenv("SPWN_HOME"); dir != "" {
		return dir
	}
	if dir := os.Getenv("UNIVERSE_HOME"); dir != "" {
		fmt.Fprintln(os.Stderr, "warning: UNIVERSE_HOME is deprecated, use SPWN_HOME instead")
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, SpwnBaseDir)
}

// WorldsDir returns the path to ~/.spwn/worlds/.
func WorldsDir() string {
	return filepath.Join(BaseDir(), WorldsSubDir)
}

// AgentsDir returns the path to ~/.spwn/agents/.
func AgentsDir() string {
	return filepath.Join(BaseDir(), AgentsSubDir)
}

// StatePath returns the path to ~/.spwn/state.json.
func StatePath() string {
	return filepath.Join(BaseDir(), StateFileName)
}

// OrgPath returns the path to the universe manifest (org.yaml).
func OrgPath() string {
	return filepath.Join(BaseDir(), OrgFileName)
}

// ClawStatePath returns the path to the Claw state file.
func ClawStatePath() string {
	return filepath.Join(BaseDir(), ClawSubDir, ClawStateFileName)
}

// SkillsDir returns the path to the skills directory.
func SkillsDir() string {
	return filepath.Join(BaseDir(), SkillsSubDir)
}

// KnowledgeDir returns the path to ~/.spwn/knowledge/.
func KnowledgeDir() string {
	return filepath.Join(BaseDir(), KnowledgeSubDir)
}

// TeamsDir returns the path to ~/.spwn/teams/.
func TeamsDir() string {
	return filepath.Join(BaseDir(), TeamsSubDir)
}

// OrganizationsDir returns the path to ~/.spwn/organizations/.
func OrganizationsDir() string {
	return filepath.Join(BaseDir(), OrganizationsSubDir)
}

// ActivityPath returns the path to ~/.spwn/activity.jsonl.
func ActivityPath() string {
	return filepath.Join(BaseDir(), ActivityFileName)
}
