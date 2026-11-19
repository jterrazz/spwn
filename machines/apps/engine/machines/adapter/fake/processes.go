package fake

import (
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

var errNoKernel = errors.New("this world reads no process of its own")

type Processes struct {
	// invariant: the processes of one Mac of the world: those the world
	// says run there, and this one's, as its kernel says; a reboot of the
	// Mac ends the world's.
	w      *world.World
	mac    string
	kernel contract.Processes
}

var _ contract.Processes = (*Processes)(nil)

func ProcessesOf(w *world.World, mac string, kernel contract.Processes) *Processes {
	return &Processes{w: w, mac: mac, kernel: kernel}
}

func (p *Processes) Self() (machines.Process, error) {
	if p.kernel == nil {
		return machines.Process{}, errNoKernel
	}
	return p.kernel.Self()
}

func (p *Processes) Runs(q machines.Process) bool {
	p.w.Lock()
	held := slices.Contains(p.w.Mac(p.mac).Processes, q)
	p.w.Unlock()
	return held || p.kernel != nil && p.kernel.Runs(q)
}
