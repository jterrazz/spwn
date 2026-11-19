package local

import (
	"strconv"

	"golang.org/x/sys/unix"
)

// value: the zombie state of a darwin process (SZOMB, sys/proc.h).
const darwinZombie = 5

func StartOf(pid int) (string, error) {
	// outside: kern.proc.pid answers the kernel's kinfo_proc; its
	// p_starttime is set once, at fork, to the microsecond.
	if pid <= 0 {
		return "", ErrNoProcess
	}
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || int(kp.Proc.P_pid) != pid || kp.Proc.P_stat == darwinZombie {
		return "", ErrNoProcess
	}
	t := kp.Proc.P_starttime
	return strconv.FormatInt(t.Sec, 10) + "." + strconv.FormatInt(int64(t.Usec), 10), nil
}
