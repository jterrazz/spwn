//go:build darwin && !cgo

package darwin

import (
	"runtime"

	"golang.org/x/sys/unix"
)

// value: a build without cgo (the release's, cross-built on Linux) reads what sysctl
// answers without IOKit or Mach: the CPU count and the memory size. The per-CPU ticks,
// the load, the pages, the swap, the disks, the links and every process stay empty,
// which are the cgo build's.
func basicCounters() (Counters, error) {
	c := Counters{CPUCount: runtime.NumCPU()}
	if n, err := unix.SysctlUint32("hw.ncpu"); err == nil && n > 0 {
		c.CPUCount = int(n)
	}
	if mem, err := unix.SysctlUint64("hw.memsize"); err == nil {
		c.Memsize = mem
	}
	return c, nil
}
