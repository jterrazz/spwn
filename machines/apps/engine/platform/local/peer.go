package local

import (
	"net"
	"syscall"
)

func uidOf(c net.Conn) (uid int, ok bool) {
	sc, is := c.(syscall.Conn)
	if !is {
		return 0, false
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return 0, false
	}
	if err := raw.Control(func(fd uintptr) { uid, ok = peerUID(int(fd)) }); err != nil {
		return 0, false
	}
	return uid, ok
}

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
