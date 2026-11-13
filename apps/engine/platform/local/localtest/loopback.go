package localtest

import (
	"net"
	"testing"
)

func Loopback(t testing.TB) net.Listener {
	// value: a TCP listener on 127.0.0.1 at a free port, closed with the
	// test: the far side of a protocol an adapter speaks over platform/local's
	// dial.
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}
