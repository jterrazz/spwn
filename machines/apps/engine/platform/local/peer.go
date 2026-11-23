package local

import "syscall"

func PeerOf(c Conn) (pid int, executable string) {
	sc, ok := c.(syscall.Conn)
	if !ok {
		return 0, ""
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return 0, ""
	}
	_ = raw.Control(func(fd uintptr) { pid = peerPID(int(fd)) })
	return pid, ExecutableOf(pid)
}
