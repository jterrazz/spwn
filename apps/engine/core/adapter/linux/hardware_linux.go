package linux

import (
	"strings"

	"golang.org/x/sys/unix"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func hardware() (machine.System, machine.Hardware) {
	var system machine.System
	var hw machine.Hardware
	var u unix.Utsname
	if unix.Uname(&u) == nil {
		system.Version = strings.TrimRight(string(u.Release[:]), "\x00")
	}
	var info unix.Sysinfo_t
	if unix.Sysinfo(&info) == nil {
		hw.Memory = int64(info.Totalram) * int64(info.Unit) //nolint:gosec // outside: the kernel's byte count fits an int64.
	}
	return system, hw
}
