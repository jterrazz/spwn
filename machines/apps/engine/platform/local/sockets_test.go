package local

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAUnixSocketIsMode0600InAFolderOfMode0700AndServesItsOwnUserAlone(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "station")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(dir, "engine.sock")
	l, err := listenUnix(sock, os.Getuid()+1)
	if err != nil {
		t.Fatal(err)
	}
	for at, want := range map[string]os.FileMode{dir: 0o700, sock: 0o600} {
		if info, statErr := os.Stat(at); statErr != nil || info.Mode().Perm() != want {
			t.Fatalf("%s: %v, %v; want mode %o", at, info, statErr, want)
		}
	}
	accepted := make(chan Conn, 1)
	go func() {
		c, _ := l.Accept()
		accepted <- c
	}()
	c, err := sockets{}.DialUnix(context.Background(), sock, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if nc, ok := c.(net.Conn); ok {
		_ = nc.SetReadDeadline(time.Now().Add(5 * time.Second))
	}
	if n, readErr := c.Read(make([]byte, 1)); n != 0 || readErr == nil || errors.Is(readErr, os.ErrDeadlineExceeded) {
		t.Fatalf("a peer of another uid read %d, %v; want it closed at accept", n, readErr)
	}
	_ = c.Close()
	_ = l.Close()
	if got := <-accepted; got != nil {
		t.Fatal("a peer of another uid was handed to the server")
	}
}

func TestALongSocketPathIsServedAndLeavesNothingInTheTempFolder(t *testing.T) {
	home := t.TempDir()
	tmp, err := os.MkdirTemp("/tmp", "sk")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	t.Setenv("TMPDIR", tmp)
	dir := filepath.Join(home, strings.Repeat("d", 60), "station")
	if err = os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(dir, "daemon.sock")
	if len(sock) < sunPathMax {
		t.Fatalf("the path %s is too short to need a link", sock)
	}

	l, err := sockets{}.ListenUnix(sock)
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan error, 1)
	go func() {
		c, acceptErr := l.Accept()
		if acceptErr == nil {
			_ = c.Close()
		}
		accepted <- acceptErr
	}()
	c, err := sockets{}.DialUnix(context.Background(), sock, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
	if err = <-accepted; err != nil {
		t.Fatal(err)
	}
	_ = l.Close()

	if left, _ := os.ReadDir(tmp); len(left) != 0 {
		t.Errorf("the temp folder kept %d entries", len(left))
	}
	if _, err := os.Lstat(sock); !os.IsNotExist(err) {
		t.Errorf("the socket outlived its listener: %v", err)
	}
}

func TestAPipeWhoseReaderLeftEndsWithoutAnError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		c, acceptErr := l.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = c.Close() }()
		for range 64 {
			if _, writeErr := c.Write([]byte("SSH-2.0-far\r\n")); writeErr != nil {
				return
			}
		}
	}()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	t.Cleanup(func() { _ = w.Close() })
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not TCP", l.Addr())
	}
	if err = (sockets{}).Pipe(context.Background(), "127.0.0.1", addr.Port, strings.NewReader(""), w); err != nil {
		t.Fatalf("Pipe = %v, want nil once its reader left", err)
	}
}
