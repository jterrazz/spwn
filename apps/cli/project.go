package cli

import (
	"os"

	"spwn.sh/packages/project"
	"spwn.sh/packages/platform"
)

// discoverProject walks up from the current directory looking for a
// spwn.yaml. When found, the project root is recorded on base so
// every path helper that is project-aware (AgentsDir, WorldsDir,
// SkillsDir, LocalStateDir) resolves inside the project.
//
// Called from root.PersistentPreRunE. Silently no-ops when no
// spwn.yaml is present - legacy user-home mode still works.
func discoverProject() {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	p, err := project.Find(cwd)
	if err != nil || p == nil {
		return
	}
	platform.SetProjectRoot(p.Root)
}
