package contract

import "github.com/jterrazz/studio-station/apps/engine/machines"

type Processes interface {
	// value: the process this code runs in.
	Self() (machines.Process, error)
	// invariant: a process runs while its pid is alive and started when it
	// says, both read from this Mac's kernel; a pid ended, taken again by
	// another process, or a zombie, runs no more.
	Runs(p machines.Process) bool
}
