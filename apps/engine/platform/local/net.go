package local

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"
)

type network struct {
	// value: the most an answer's body is read to, so one that streams
	// without end fails rather than fills the memory.
	most int64
}

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
	a, err := n.Fetch(ctx, url, header)
	return a.Status, a.Body, err
}

func (n network) Fetch(ctx context.Context, url string, header map[string]string) (Reply, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return Reply{}, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Reply{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	said := map[string]string{}
	for k := range resp.Header {
		said[k] = resp.Header.Get(k)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, n.most+1))
	if err == nil && int64(len(body)) > n.most {
		return Reply{Status: resp.StatusCode, Header: said}, fmt.Errorf("%s answered past %d MiB", url, n.most>>20)
	}
	return Reply{Status: resp.StatusCode, Header: said, Body: body}, err
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
