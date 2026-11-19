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

type BuildState string

// value: where one step of a build stands: finished, the one running, or
// still to come.
const (
	BuildDone BuildState = "done"
	BuildNow  BuildState = "now"
	BuildNext BuildState = "next"
)

func (BuildState) Enum() []string {
	return []string{string(BuildDone), string(BuildNow), string(BuildNext)}
}

func (s BuildState) Word() string {
	switch s {
	case BuildDone:
		return "Done"
	case BuildNow:
		return "Now"
	case BuildNext:
		return "Next"
	}
	return string(s)
}

type BuildWalk struct {
	// value: one step of a build as images.list tells it: the step, where it
	// stands, and the word the engine says it by.
	Step  BuildStep
	State BuildState
	Word  string
}

type Building struct {
	// value: where a build is, as images.list tells it: its step, its position
	// among the build's steps and how many there are, the sentence a client
	// shows, "Step 2 of 4", and every step in order with where it stands.
	Step     BuildStep
	Position int
	Steps    int
	Sentence string
	Walk     []BuildWalk
}

func BuildingAt(step BuildStep) *Building {
	all := BuildSteps()
	b := &Building{Step: step, Steps: len(all)}
	for i, s := range all {
		if s == step {
			b.Position = i + 1
		}
	}
	for i, s := range all {
		state := BuildNext
		switch {
		case i+1 < b.Position:
			state = BuildDone
		case i+1 == b.Position:
			state = BuildNow
		}
		b.Walk = append(b.Walk, BuildWalk{Step: s, State: state, Word: s.Word()})
	}
	b.Sentence = "Step " + strconv.Itoa(b.Position) + " of " + strconv.Itoa(b.Steps)
	return b
}
