package kernel

import (
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type Processes struct {
	// invariant: this Mac's processes as its kernel tells them: a process
	// runs while its pid lives with the start it was read with.
}

var _ contract.Processes = Processes{}

func (Processes) Self() (model.Process, error) {
	pid, start, err := local.Processes{}.Self()
	return model.Process{PID: pid, Start: start}, err
}

func (Processes) Runs(p model.Process) bool { return local.Processes{}.Runs(p.PID, p.Start) }
