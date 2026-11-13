package local

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
