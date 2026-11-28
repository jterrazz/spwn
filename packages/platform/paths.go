package platform

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

// Folders is where, from a project's root, the folder of each kind of what
// it declares sits, as a slash path: what packages/project's Layout says,
// handed to the packages below it, which never read a manifest.
//
// invariant: a field left empty is the kind's own name, agents for
// Agents and so on, as a project whose spwn.yaml renames no folder lays
// them out; so the zero Folders is that default, never the root itself.
type Folders struct {
	Agents   string
	Skills   string
	Tools    string
	Hooks    string
	Commands string
	Profiles string
}

func (f Folders) named() Folders {
	for _, field := range []struct {
		at   *string
		name string
	}{
		{&f.Agents, "agents"}, {&f.Skills, "skills"}, {&f.Tools, "tools"},
		{&f.Hooks, "hooks"}, {&f.Commands, "commands"}, {&f.Profiles, "profiles"},
	} {
		if *field.at == "" {
			*field.at = field.name
		}
	}
	return f
}

var (
	projectRootMu  sync.RWMutex
	projectRoot    string
	projectFolders Folders
)

// SetProject tells the base package where this process's spwn project
// lives and where its folders sit. Path helpers that are project-aware
// (AgentsDir, SkillsDir, LocalStateDir) then resolve inside root.
//
// Pass "" to clear. Callers typically call this once from a cobra
// PersistentPreRun after discovering the project with project.Find.
func SetProject(root string, folders Folders) {
	projectRootMu.Lock()
	defer projectRootMu.Unlock()
	projectRoot = root
	projectFolders = folders.named()
}

// ProjectFolders returns the folders of the active project, as SetProject
// was given them.
func ProjectFolders() Folders {
	projectRootMu.RLock()
	defer projectRootMu.RUnlock()
	return projectFolders
}

func projectFolder(pick func(Folders) string) (root, folder string) {
	projectRootMu.RLock()
	defer projectRootMu.RUnlock()
	return projectRoot, pick(projectFolders)
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
// UserDir (user-level, always) or a project-aware helper (AgentsDir,
// SkillsDir).
func BaseDir() string {
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

// StateTree is the folder LocalStateDir, AgentsDir and SkillsDir are under, as a Tree:
// the project's root when a project is active, so a write there cannot
// leave the project by a link it carries, else the user's own home,
// made when missing.
func StateTree() *Tree {
	if root := ProjectRoot(); root != "" {
		return Within(root)
	}
	_ = os.MkdirAll(UserDir(), 0o755)
	return Within(UserDir())
}

// WorldsDir returns the user-level directory of the named world configs,
// ~/.spwn/worlds/: a project declares its worlds in spwn.yaml, never in a
// folder of its root.
func WorldsDir() string {
	return filepath.Join(UserDir(), WorldsSubDir)
}

// AgentsDir returns the agents directory: the project's agents folder
// when a project is active, else ~/.spwn/agents/.
func AgentsDir() string {
	if root, folder := projectFolder(func(f Folders) string { return f.Agents }); root != "" {
		return filepath.Join(root, filepath.FromSlash(folder))
	}
	return filepath.Join(UserDir(), AgentsSubDir)
}

// SkillsDir returns the skills directory: the project's skills folder
// when a project is active, else ~/.spwn/skills/.
func SkillsDir() string {
	if root, folder := projectFolder(func(f Folders) string { return f.Skills }); root != "" {
		return filepath.Join(root, filepath.FromSlash(folder))
	}
	return filepath.Join(UserDir(), SkillsSubDir)
}

// CredentialsDir always returns the user-level credentials directory.
// Credentials are intentionally never project-scoped - projects must
// never commit or reference auth material.
func CredentialsDir() string {
	return filepath.Join(UserDir(), CredentialsSubDir)
}

// GateDir is the host-side state directory bind-mounted into the
// gate container at /gate. Holds installed catalog tools (under
// tools/), per-tool persistence, and any other gate-owned state.
func GateDir() string {
	return filepath.Join(UserDir(), "gate")
}

// GateToolsDir is where `spwn install spwn:<name>` materialises
// catalog tools that declare a `gate:` section. The gate scans this
// dir at startup and registers each tool's CookieProvider + MCP
// subprocess. Returns the host path; inside the gate container the
// same dir is visible at /gate/tools/.
func GateToolsDir() (string, error) {
	dir := filepath.Join(GateDir(), "tools")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
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

// invariant: one act holds the project root from the root it sets to its
// last read, so two acts in one process never see each other's root.
var actMu sync.Mutex

// WithProject runs act with root and folders as the process's project,
// under a lock every caller of it shares, and restores the project it found
// when act returns.
func WithProject[T any](root string, folders Folders, act func() (T, error)) (T, error) {
	// invariant: an act never calls WithProject, which is not re-entrant.
	actMu.Lock()
	defer actMu.Unlock()
	projectRootMu.RLock()
	previousRoot, previousFolders := projectRoot, projectFolders
	projectRootMu.RUnlock()
	SetProject(root, folders)
	defer SetProject(previousRoot, previousFolders)
	return act()
}
