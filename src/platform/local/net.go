package local

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"
)

type network struct{}

func (network) Dial(ctx context.Context, addr string, port int, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(addr, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (n network) Get(ctx context.Context, url string, header map[string]string) (status int, body []byte, err error) {
	return n.do(ctx, http.MethodGet, url, header, nil)
}

func (network) do(ctx context.Context, method, url string, header map[string]string, payload []byte) (status int, body []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err = io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

func (network) Broadcast(ctx context.Context, port int, payload []byte) error {
	lc := net.ListenConfig{Control: func(_, _ string, c syscall.RawConn) error {
		var err error
		if cerr := c.Control(func(fd uintptr) {
			err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		}); cerr != nil {
			return cerr
		}
		return err
	}}
	conn, err := lc.ListenPacket(ctx, "udp4", ":0")
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	_, err = conn.WriteTo(payload, &net.UDPAddr{IP: net.IPv4bcast, Port: port})
	return err
}

func (network) HardwareAddress(ip string) string {
	want := net.ParseIP(ip)
	if want == nil {
		return ""
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil || len(i.HardwareAddr) == 0 {
			continue
		}
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.Equal(want) {
				return i.HardwareAddr.String()
			}
		}
	}
	return ""
}
