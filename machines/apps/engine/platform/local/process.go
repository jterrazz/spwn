package local

import (
	"errors"
	"os"
)

var ErrNoProcess = errors.New("no such process")

type Processes struct{}

func (Processes) Self() (pid int, start string, err error) {
	pid = os.Getpid()
	start, err = StartOf(pid)
	return pid, start, err
}

func (Processes) Runs(pid int, start string) bool {
	// invariant: the start a process was recorded with is read again from
	// the same kernel, so a pid taken again by a later process differs.
	now, err := StartOf(pid)
	return err == nil && start != "" && now == start
}
