package linux

import (
	"context"
	"runtime"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Self struct {
	hostname string
	hardware func() (machine.System, machine.Hardware)
}

func New(hostname string) *Self { return &Self{hostname: hostname, hardware: hardware} }

func (s *Self) Read(context.Context) (machine.Ref, machine.Kind, machine.System, machine.Hardware, error) {
	// invariant: station runs on Linux only inside a VM a Mac created, so a
	// Linux machine is a workspace.
	system, hw := s.hardware()
	system.OS, system.Arch = machine.Linux, runtime.GOARCH
	hw.CPUs = runtime.NumCPU()
	return machine.Ref{Name: s.hostname}, machine.Workspace, system, hw, nil
}
