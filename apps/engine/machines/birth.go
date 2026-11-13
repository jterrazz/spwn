package machines

import (
	"slices"
	"strconv"
	"time"
)

const Creating VMState = "creating"

type Birth struct {
	// value: how far its create is, per mille; the step it runs, its position
	// among the create's steps and how many it has, the image's build one
	// of them only when the create builds it; what that step does now, the
	// build's own step while its image is built; when it reached that step.
	Progress int       `json:"progress"`
	Step     BirthStep `json:"step"`
	Position int       `json:"position"`
	Steps    int       `json:"steps"`
	Detail   string    `json:"detail,omitempty"`
	At       time.Time `json:"at"`
}

type BirthStep string

// value: the create's steps, in order: its image built when it is missing,
// its disk cloned, its first run, its address, its ssh.
const (
	BornBuilding BirthStep = "building"
	BornCloning  BirthStep = "cloning"
	BornBooting  BirthStep = "booting"
	BornAddress  BirthStep = "address"
	BornSSH      BirthStep = "ssh"
)

func (BirthStep) Enum() []string {
	return []string{string(BornBuilding), string(BornCloning), string(BornBooting), string(BornAddress), string(BornSSH)}
}

func (BirthStep) Word() string {
	// value: a VM reads Creating through every step of its create.
	return "Creating"
}

func (s BirthStep) Noun() string {
	switch s {
	case BornBuilding:
		return "image"
	case BornCloning:
		return "disk"
	case BornBooting:
		return "boot"
	case BornAddress:
		return "network"
	case BornSSH:
		return "SSH"
	}
	return string(s)
}

// value: the position of a step among a create's steps, from 1; 0 for a step
// a create of that many steps does not have.
func positionOf(step BirthStep, steps int) int {
	all := step.Enum()
	i := slices.Index(all, string(step))
	if i < 0 || steps < len(all)-1 || steps > len(all) {
		return 0
	}
	return max(i+1-(len(all)-steps), 0)
}

func (b Birth) Known() bool {
	return b.Position > 0 && b.Position == positionOf(b.Step, b.Steps)
}

func (b Birth) Doing() string {
	// value: what a reader shows after the VM's Creating: its step's noun,
	// its position and how many there are, "image 1 of 5".
	return b.Step.Noun() + " " + strconv.Itoa(b.Position) + " of " + strconv.Itoa(b.Steps)
}

func BirthAt(step BirthStep, builds bool, at time.Time) *Birth {
	steps := len(step.Enum()) - 1
	if builds {
		steps++
	}
	progress := map[BirthStep]int{BornBuilding: 20, BornCloning: 50, BornBooting: 400, BornAddress: 600, BornSSH: 800}[step]
	return &Birth{Progress: progress, Step: step, Position: positionOf(step, steps), Steps: steps, At: at.UTC().Truncate(time.Second)}
}

const HalfMade = "was left half made"

const FreshFor = 8 * time.Second

func Fresh(vm VM, now time.Time) bool {
	return !vm.ReadyAt.IsZero() && !now.Before(vm.ReadyAt) && now.Sub(vm.ReadyAt) < FreshFor
}

type Process struct {
	// value: a process of a Mac: its pid, and when it started as that Mac's
	// kernel says, so a pid taken again names another process.
	PID   int    `json:"pid"`
	Start string `json:"start"`
}

func Born(vm VM, b *Birth, runs bool) VM {
	// invariant: a birth wins over what the disk and the engine say, so a VM
	// reads creating until its create ends. A create runs in the process
	// that asked it, a command or the engine, and dies with it: a birth
	// whose process no longer runs (killed, its Mac restarted) has no create
	// left, nor has one whose step and position this station does not know;
	// it reads stopped, half made, to start or delete, its memory given back.
	// A create that runs reads creating however long a step takes, an engine
	// restart included.
	switch {
	case b == nil:
	case b.Known() && runs:
		vm.State, vm.Birth = Creating, b
	case vm.State == Running:
	default:
		vm.State, vm.Detail = Stopped, HalfMade
	}
	return vm
}
