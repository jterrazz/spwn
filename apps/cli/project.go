package cli

import (
	"errors"
	"os"

	"spwn.sh/packages/platform"
	"spwn.sh/packages/project"
)

// invariant: a project of the old spwn/ layout stops every command here,
// before any of them reads a folder or starts a world; any other load error
// leaves the command to report it, or to run without a project.
func discoverProject() error {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	p, err := project.Find(cwd)
	var old *project.OldLayoutError
	if errors.As(err, &old) {
		return err
	}
	if err != nil || p == nil {
		return nil
	}
	platform.SetProject(p.Root, p.Layout.Platform())
	return nil
}
