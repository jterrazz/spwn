// Package cliproject is the one way a CLI command finds the active spwn
// project. Two entry points:
//
//   - Find()    returns (nil, nil) only when no spwn.yaml sits on the
//     way up, so the command runs in the user's own home.
//   - Require() returns a crisp "no spwn.yaml" error when no
//     project is present, for commands that require one.
package cliproject

import (
	"fmt"
	"os"

	"spwn.sh/packages/project"
)

// Find walks up from cwd looking for a spwn.yaml. It returns (nil, nil)
// only when no spwn.yaml is found; a spwn.yaml found that does not load,
// for any reason, is its error, which the caller returns as it is.
func Find() (*project.Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return project.Find(cwd)
}

// Require is Find with a friendly error when no project exists.
// Commands that can't operate in legacy mode use this so the user
// sees a consistent "run `station project create`" hint.
func Require() (*project.Project, error) {
	p, err := Find()
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("no spwn.yaml found: run `station project create <name>` first")
	}
	return p, nil
}
