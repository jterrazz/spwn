package macos

import (
	"context"
	"runtime"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const sysctlWait = 2 * time.Second

type Commands interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Self struct {
	run      Commands
	hostname string
	hardware func() (machine.System, machine.Hardware)
}

func New(run Commands, hostname string) *Self {
	return &Self{run: run, hostname: hostname, hardware: hardware}
}

func (s *Self) Read(ctx context.Context) (machine.Ref, machine.Kind, machine.System, machine.Hardware, error) {
	// invariant: a macOS machine is a workspace when the kernel says it runs
	// under a hypervisor, and a Mac otherwise: anything but a clear "1" is a Mac,
	// a missing or failing sysctl included, so a sandbox with no system tool
	// still answers a kind.
	kind := machine.Computer
	out, err := s.run.Output(ctx, local.Cmd{Name: "sysctl", Args: []string{"-n", "kern.hv_vmm_present"}, Timeout: sysctlWait})
	if err == nil && out.ExitCode == 0 && strings.TrimSpace(out.Stdout) == "1" {
		kind = machine.Workspace
	}
	system, hw := s.hardware()
	system.OS, system.Arch = machine.MacOS, runtime.GOARCH
	hw.CPUs = runtime.NumCPU()
	return machine.Ref{Name: s.hostname}, kind, system, hw, nil
}
