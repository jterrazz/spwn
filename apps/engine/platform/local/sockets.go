package local

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type sockets struct{}

type listener struct {
	l    net.Listener
	sock string
}

func (l listener) Accept() (Conn, error) { return l.l.Accept() }
func (l listener) Close() error {
	err := l.l.Close()
	if l.sock != "" {
		_ = os.Remove(l.sock)
	}
	return err
}
func (l listener) Port() int {
	if a, ok := l.l.Addr().(*net.TCPAddr); ok {
		return a.Port
	}
	return 0
}

const sunPathMax = 100

// hazard: a unix socket path is capped near 104 bytes (sun_path); a longer one is
// bound or dialed through a short symlink to its folder, made in a fresh temp
// folder for that one call and removed once the call returns, so concurrent
// calls never share a link and none is left behind.
func viaShort(path string, call func(string) error) error {
	if len(path) < sunPathMax {
		return call(path)
	}
	dir, name := filepath.Split(path)
	for _, base := range []string{os.TempDir(), "/tmp"} {
		tmp, err := os.MkdirTemp(base, "station-")
		if err != nil {
			continue
		}
		link := filepath.Join(tmp, "d")
		short := filepath.Join(link, name)
		if len(short) >= sunPathMax || os.Symlink(filepath.Clean(dir), link) != nil {
			_ = os.RemoveAll(tmp)
			continue
		}
		err = call(short)
		_ = os.Remove(link)
		_ = os.Remove(tmp)
		return err
	}
	return call(path)
}

func (sockets) ListenUnix(path string) (Listener, error) {
	var l net.Listener
	if err := viaShort(path, func(p string) (err error) {
		l, err = net.Listen("unix", p)
		return err
	}); err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = l.Close()
		return nil, err
	}
	sock := ""
	if len(path) >= sunPathMax {
		sock = path
	}
	return listener{l: l, sock: sock}, nil
}

func (sockets) DialTCP(ctx context.Context, address string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.DialContext(ctx, "tcp", address)
}

func (sockets) DialUnix(ctx context.Context, path string, timeout time.Duration) (Conn, error) {
	d := net.Dialer{Timeout: timeout}
	var conn net.Conn
	err := viaShort(path, func(p string) (err error) {
		conn, err = d.DialContext(ctx, "unix", p)
		return err
	})
	return conn, err
}

// invariant: no code path binds every interface: an empty or unspecified ip is
// refused here, the one place a TCP listener is made.
func (sockets) ListenTCP(ip string, port int) (Listener, error) {
	parsed := net.ParseIP(ip)
	if ip == "" || parsed == nil || parsed.IsUnspecified() {
		return nil, ErrWildcard
	}
	l, err := net.Listen("tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	return listener{l: l}, nil
}

func (sockets) Pipe(ctx context.Context, ip string, port int, in io.Reader, out io.Writer) error {
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	go func() {
		_, _ = io.Copy(conn, in)
		if tcp, ok := conn.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
	}()
	_, err = io.Copy(out, conn)
	_ = conn.Close()
	// invariant: a reader that left (ssh exited, a head took its bytes) ends
	// the pipe as the far side's close does: neither is the sshd failing.
	if errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.EPIPE) {
		return nil
	}
	return err
}

type halfCloser interface{ CloseWrite() error }

// invariant: each direction ends on its own: the end of one side's bytes
// half-closes the other's write, so a request's EOF never cuts its response;
// both close once both directions ended.
func (sockets) Join(a, b Conn) {
	var wg sync.WaitGroup
	carry := func(dst, src Conn) {
		defer wg.Done()
		_, _ = io.Copy(dst, src)
		if h, ok := dst.(halfCloser); ok {
			_ = h.CloseWrite()
		} else {
			_ = dst.Close()
		}
	}
	wg.Add(2)
	go carry(a, b)
	go carry(b, a)
	wg.Wait()
	_ = a.Close()
	_ = b.Close()
}
