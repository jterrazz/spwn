package paths

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PrettyHome collapses the user's home directory prefix to `~` so
// user-facing paths do not leak absolute host locations. Returns the
// input unchanged when the home dir cannot be resolved or when the
// path is not under $HOME. See finding #30.
func PrettyHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}

var (
	projectRootMu sync.RWMutex
	projectRoot   string
)

// SetProjectRoot tells the base package where this process's spwn
// project lives. Path helpers that are project-aware (AgentsDir, WorldsDir,
// SkillsDir, LocalStateDir) then resolve inside <projectRoot>.
//
// Pass "" to clear. Callers typically call this once from a cobra
// PersistentPreRun after discovering the project with manifest.Find.
func SetProjectRoot(path string) {
	projectRootMu.Lock()
	defer projectRootMu.Unlock()
	projectRoot = path
}

// ProjectRoot returns the currently active project root, or "" if no
// project is active (legacy global mode).
func ProjectRoot() string {
	projectRootMu.RLock()
	defer projectRootMu.RUnlock()
	return projectRoot
}

// HasProject reports whether a project root is active.
func HasProject() bool {
	return ProjectRoot() != ""
}

// UserDir is the user-global home: ~/.spwn/. If SPWN_HOME is set, it
// overrides the default (used for test isolation). Always user-level -
// credentials, daemon state, and the activity log live here regardless
// of whether a project is active.
func UserDir() string {
	if dir := os.Getenv("SPWN_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, SpwnBaseDir)
}

// BaseDir is a historical alias for UserDir. New code should prefer
// UserDir (user-level, always) or DataDir (project-aware).
func BaseDir() string {
	return UserDir()
}

// DataDir is the project-aware root for project assets - agents,
// worlds, skills, custom tool packs. Returns <projectRoot>/spwn/ when
// a project is active, or UserDir() otherwise (legacy global mode).
func DataDir() string {
	if root := ProjectRoot(); root != "" {
		return filepath.Join(root, "spwn")
	}
	return UserDir()
}

// LocalStateDir is the project's gitignored state directory (.spwn/).
// Used for transient runtime state like world-states and caches.
// Returns <projectRoot>/.spwn/ when a project is active, or UserDir()
// otherwise.
func LocalStateDir() string {
	if root := ProjectRoot(); root != "" {
		return filepath.Join(root, ".spwn")
	}
	return UserDir()
}

// --- Project-aware data paths ---

// WorldsDir returns the worlds config directory.
func WorldsDir() string {
	return filepath.Join(DataDir(), WorldsSubDir)
}

// AgentsDir returns the agents directory.
func AgentsDir() string {
	return filepath.Join(DataDir(), AgentsSubDir)
}

// SkillsDir returns the skills directory.
func SkillsDir() string {
	return filepath.Join(DataDir(), SkillsSubDir)
}

// --- User-level paths (never project-aware) ---

// CredentialsDir always returns the user-level credentials directory.
// Credentials are intentionally never project-scoped - projects must
// never commit or reference auth material.
func CredentialsDir() string {
	return filepath.Join(UserDir(), CredentialsSubDir)
}

// StatePath returns the legacy state.json path (user-level).
func StatePath() string {
	return filepath.Join(UserDir(), StateFileName)
}

// ActivityPath returns the user-level activity log.
func ActivityPath() string {
	return filepath.Join(UserDir(), ActivityFileName)
}

// TeamsDir returns the user-level teams directory.
func TeamsDir() string {
	return filepath.Join(UserDir(), TeamsSubDir)
}

// OrganizationsDir returns the user-level organizations directory.
func OrganizationsDir() string {
	return filepath.Join(UserDir(), OrganizationsSubDir)
}

// OrgPath returns the path to the legacy org.yaml file.
// Deprecated: org.yaml is no longer created or read. Kept only for
// migration 006 compatibility.
func OrgPath() string {
	return filepath.Join(UserDir(), "org.yaml")
}
