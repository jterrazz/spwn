package macos

import (
	"golang.org/x/sys/unix"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

// invariant: read by sysctl(3), never a subprocess: what does not move
// while the Mac runs.
func hardware() (machine.System, machine.Hardware) {
	var system machine.System
	var hw machine.Hardware
	system.Version, _ = unix.Sysctl("kern.osproductversion")
	hw.Model, _ = unix.Sysctl("hw.model")
	hw.Chip, _ = unix.Sysctl("machdep.cpu.brand_string")
	if n, err := unix.SysctlUint64("hw.memsize"); err == nil {
		hw.Memory = int64(n) //nolint:gosec // outside: a Mac's memory fits an int64 by orders of magnitude.
	}
	return system, hw
}
