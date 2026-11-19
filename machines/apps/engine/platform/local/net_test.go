package local

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestTheHardwareAddressIsTheOneOfTheInterfaceThatHoldsTheAddress(t *testing.T) {
	ip, want := "", ""
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range ifaces {
		if len(i.HardwareAddr) == 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil {
				ip, want = n.IP.String(), i.HardwareAddr.String()
			}
		}
	}
	if ip == "" {
		t.Skip("no interface here holds both a hardware and an IPv4 address")
	}
	if got := (network{}).HardwareAddress(ip); got != want {
		t.Fatalf("HardwareAddress(%s) = %q, want %q", ip, got, want)
	}
	if got := (network{}).HardwareAddress("192.0.2.77"); got != "" {
		t.Fatalf("an address no interface holds gave %q", got)
	}
}

// value: an HTTP server on the loopback that answers every request with
// a body of size bytes, written by hand on the socket.
func answering(t *testing.T, size int) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = c.Close() }()
				_, _ = c.Read(make([]byte, 4096))
				_, _ = c.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: " + strconv.Itoa(size) + "\r\nConnection: close\r\n\r\n" + strings.Repeat("x", size)))
			}()
		}
	}()
	return "http://" + l.Addr().String() + "/"
}

func TestAnAnswerPastItsBoundFails(t *testing.T) {
	url := answering(t, 3<<20)

	if _, body, err := (network{most: 2 << 20}).Get(context.Background(), url, nil); err == nil || body != nil {
		t.Errorf("read %d bytes (%v); want the answer refused past 2 MiB", len(body), err)
	}
	if _, body, err := (network{most: 4 << 20}).Get(context.Background(), url, nil); err != nil || len(body) != 3<<20 {
		t.Errorf("read %d bytes (%v); want the whole answer within its bound", len(body), err)
	}
}
