package calls

import (
	"context"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const (
	stoppedBudget = 60 * time.Second
	hardStopWait  = 10 * time.Second
)

const diskMissing contract.DiskState = ""

func (s *Machinist) engineState(ctx context.Context, name string) contract.DiskState {
	state, _ := s.hypervisorState(ctx, name)
	return state
}

// invariant: a hypervisor that does not answer its listing says nothing of
// a VM's state: it is that the hypervisor does not answer, never that the
// VM is stopped, since a VM it ran keeps running without it.
func (s *Machinist) hypervisorState(ctx context.Context, name string) (contract.DiskState, error) {
	disks, err := s.engine.Disks(ctx)
	if err != nil {
		return diskMissing, &model.NoEngineError{Hypervisor: s.engine.Name(), Package: s.engine.Package(), Cause: err}
	}
	for _, d := range disks {
		if d.VM == name {
			return d.State, nil
		}
	}
	return diskMissing, nil
}

// invariant: a stop the engine cannot ask of its guest is the hypervisor's
// graceful shutdown: the guest powers itself off within the budget, its
// station-intent unit running as for its own poweroff, and only then is it cut.
func (s *Machinist) powerOff(ctx context.Context, name string) error {
	if s.engineState(ctx, name) != contract.DiskRunning {
		return nil
	}
	return s.engine.Shutdown(ctx, name, stoppedBudget)
}

func (s *Machinist) PowerOff(ctx context.Context, name string) error {
	return s.powerOff(ctx, name)
}

func (s *Machinist) EngineRunning(ctx context.Context, name string) (bool, error) {
	state, err := s.hypervisorState(ctx, name)
	return state == contract.DiskRunning, err
}
