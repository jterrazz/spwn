package local

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAConnectionNamesTheProcessThatListensAtItsOtherEnd(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "held.sock")
	l, err := (sockets{}).ListenUnix(sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	c, err := (sockets{}).DialUnix(t.Context(), sock, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	self, _ := os.Executable()
	if pid, exe := PeerOf(c); pid != os.Getpid() || exe != self {
		t.Fatalf("the peer read %d %q, want %d %q", pid, exe, os.Getpid(), self)
	}
}
