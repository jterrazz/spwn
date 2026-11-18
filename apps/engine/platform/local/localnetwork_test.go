package local

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"
	"testing"
)

func TestOnlyADarwinUnreachableRouteIsTheLocalNetworkBlock(t *testing.T) {
	dialed := func(errno syscall.Errno) error {
		return &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", errno)}
	}
	for _, c := range []struct {
		name, goos string
		err        error
		want       bool
	}{
		{"darwin, unreachable", "darwin", dialed(syscall.EHOSTUNREACH), true},
		{"linux, unreachable", "linux", dialed(syscall.EHOSTUNREACH), false},
		{"darwin, refused", "darwin", dialed(syscall.ECONNREFUSED), false},
		{"darwin, timed out", "darwin", dialed(syscall.ETIMEDOUT), false},
		{"darwin, network down", "darwin", dialed(syscall.ENETUNREACH), false},
		{"darwin, the dial's end", "darwin", context.DeadlineExceeded, false},
		{"darwin, no error", "darwin", nil, false},
		{"darwin, another error", "darwin", errors.New("handshake failed"), false},
	} {
		if got := BlockedByLocalNetwork(c.goos, c.err); got != c.want {
			t.Errorf("%s: BlockedByLocalNetwork = %v, want %v", c.name, got, c.want)
		}
	}
}
