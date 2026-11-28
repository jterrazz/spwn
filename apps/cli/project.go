package cli

import (
	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/packages/platform"
)

// invariant: a spwn.yaml found from the working folder that does not load,
// for any reason, stops every command here, before any of them reads a
// folder, writes under the home or starts a world; only a folder with no
// spwn.yaml on the way up runs without a project.
func discoverProject() error {
	p, err := cliproject.Find()
	if err != nil {
		return err
	}
	if p != nil {
		platform.SetProject(p.Root, p.Layout.Platform())
	}
	return nil
}
