package images

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type BuildStep string

// value: the steps of an image's build, in order: the base pulled, cloned
// and booted; the station binary and the plumbing recipe that installs
// what it needs; the Machinefile's image phase; the cleanup, the power
// off and the record.
const (
	BuildBaseSystem BuildStep = "base-system"
	BuildPackages   BuildStep = "packages"
	BuildSettings   BuildStep = "settings"
	BuildSeal       BuildStep = "seal"
)

func BuildSteps() []BuildStep {
	return []BuildStep{BuildBaseSystem, BuildPackages, BuildSettings, BuildSeal}
}

func (BuildStep) Enum() []string {
	steps := BuildSteps()
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = string(s)
	}
	return out
}

func (s BuildStep) Word() string {
	switch s {
	case BuildBaseSystem:
		return "Base system"
	case BuildPackages:
		return "Packages"
	case BuildSettings:
		return "Settings"
	case BuildSeal:
		return "Seal"
	}
	return string(s)
}

type Making struct {
	// value: an image's build as it stands: the version it will be, and the
	// step it is at.
	Image   string    `json:"image"`
	Version string    `json:"version"`
	Step    BuildStep `json:"step"`
	// value: the process the build runs in: a build whose process no longer
	// runs has ended without saying so, and is no build.
	Owner *machines.Process `json:"owner,omitempty"`
}

type Building struct {
	// value: where a build is, as images.list tells it: its step, its position
	// among the build's steps and how many there are, and the sentence a client
	// shows, "Step 2 of 4".
	Step     BuildStep
	Position int
	Steps    int
	Sentence string
}

func BuildingAt(step BuildStep) *Building {
	all := BuildSteps()
	b := &Building{Step: step, Steps: len(all)}
	for i, s := range all {
		if s == step {
			b.Position = i + 1
		}
	}
	b.Sentence = "Step " + strconv.Itoa(b.Position) + " of " + strconv.Itoa(b.Steps)
	return b
}
