package linux

import (
	"golang.org/x/sys/unix"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func hardware(read func(string) ([]byte, error)) (machine.System, machine.Hardware) {
	var system machine.System
	var hw machine.Hardware
	system.Version = osVersion(read)
	var info unix.Sysinfo_t
	if unix.Sysinfo(&info) == nil {
		hw.Memory = int64(info.Totalram) * int64(info.Unit) //nolint:gosec // outside: the kernel's byte count fits an int64.
	}
	return system, hw
}
