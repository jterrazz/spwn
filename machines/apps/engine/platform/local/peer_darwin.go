package local

import (
	"bytes"

	"golang.org/x/sys/unix"
)

func peerPID(fd int) int {
	pid, err := unix.GetsockoptInt(fd, unix.SOL_LOCAL, unix.LOCAL_PEERPID)
	if err != nil {
		return 0
	}
	return pid
}

func peerUID(fd int) (int, bool) {
	cred, err := unix.GetsockoptXucred(fd, unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	if err != nil {
		return 0, false
	}
	return int(cred.Uid), true
}

func ExecutableOf(pid int) string {
	if pid <= 0 {
		return ""
	}
	raw, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil || len(raw) < 4 {
		return ""
	}
	exe, _, _ := bytes.Cut(raw[4:], []byte{0})
	return string(exe)
}
