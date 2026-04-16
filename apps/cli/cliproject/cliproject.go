// Package cliproject centralises the "find the active spwn project"
// dance every CLI subcommand otherwise reimplements. Two entry points:
//
//   - Find()    — returns (nil, nil) when no project is present so
//                 callers can fall back to legacy global mode.
//   - Require() — returns a crisp "no spwn.yaml" error when no
//                 project is present, for commands that require one.
package cliproject

import (
	"fmt"
	"os"

	"spwn.sh/packages/project"
)

// Find walks up from cwd looking for a spwn.yaml. Returns (nil, nil)
// when no project is found (legacy global-mode fallback) and an error
// only when cwd itself cannot be resolved or the discovered manifest
// fails to load.
func Find() (*project.Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return project.Find(cwd)
}

// Require is Find with a friendly error when no project exists.
// Commands that can't operate in legacy mode use this so the user
// sees a consistent "run `spwn init`" hint.
func Require() (*project.Project, error) {
	p, err := Find()
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("no spwn.yaml found — run `spwn init` first")
	}
	return p, nil
}
