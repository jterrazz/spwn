package cli

import (
	"os"

	"spwn.sh/packages/platform"
	"spwn.sh/packages/project"
)

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
