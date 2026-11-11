package procnet

import (
	"context"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/ports/contract"
	"github.com/jterrazz/studio-station/src/ports/contract/contracttest"
)

type proc struct{ fstest.MapFS }

func (p proc) ReadFile(name string) ([]byte, error) {
	return p.MapFS.ReadFile(strings.TrimPrefix(name, "/"))
}

func (p proc) ReadDir(name string) ([]fs.DirEntry, error) {
	return p.MapFS.ReadDir(strings.TrimPrefix(name, "/"))
}

func (proc) Readlink(string) (string, error) { return "", fs.ErrInvalid }

const header = "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n"

func row(local, state, inode string) string {
	return "   0: " + local + " 00000000:0000 " + state + " 00000000:00000000 00:00000000 00000000  1000        0 " + inode + " 1 0000000000000000 100 0 0 10 0\n"
}

func file(text string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(text)} }

func TestProcnetKeepsTheListenersContract(t *testing.T) {
	contracttest.Listeners(t, func(*testing.T) contract.Listeners {
		return New(proc{fstest.MapFS{
			"proc/net/tcp":     file(header + row("00000000:0BB8", "0A", "201") + row("0100007F:1538", "0A", "202")),
			"proc/net/tcp6":    file(header + row("00000000000000000000000000000000:0BB8", "0A", "203")),
			"proc/100/cmdline": file("/usr/bin/node\x00server.js\x00"),
			"proc/100/fd/3":    file("socket:[201]"),
			"proc/100/fd/4":    file("socket:[203]"),
			"proc/101/cmdline": file("postgres: checkpointer\x00"),
			"proc/101/comm":    file("postgres\n"),
			"proc/101/fd/5":    file("socket:[202]"),
		}}, "")
	})
}

func TestProcnetReadsAddressesInHostByteOrder(t *testing.T) {
	tcp := header + row("00000000:1F91", "0A", "101") + row("0100007F:1538", "0A", "0") + row("0100007F:2328", "01", "0")
	tcp6 := header + row("0000000000000000FFFF00000100007F:1F91", "0A", "0") + row("00000000000000000000000001000000:0050", "0A", "0")
	got, err := New(proc{fstest.MapFS{"proc/net/tcp": file(tcp), "proc/net/tcp6": file(tcp6)}}, "").Read(context.Background())
	want := []ports.Listener{{Port: 8081, Bind: "127.0.0.1"}, {Port: 80, Bind: "::1"}, {Port: 8081, Bind: "0.0.0.0"}, {Port: 5432, Bind: "127.0.0.1"}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("Read =\n%+v\nwant\n%+v", got, want)
	}
}

func TestProcnetReadsTheRangeOfOutgoingPorts(t *testing.T) {
	read := func(fsys fstest.MapFS) (int, int, bool) { return New(proc{fsys}, "/").Range(context.Background()) }
	if from, to, ok := read(fstest.MapFS{"proc/sys/net/ipv4/ip_local_port_range": file("15000\t30000\n")}); !ok || from != 15000 || to != 30000 {
		t.Fatalf("range %d-%d %v", from, to, ok)
	}
	for _, bad := range []string{"", "1", "9 x", "9000 8000"} {
		if _, _, ok := read(fstest.MapFS{"proc/sys/net/ipv4/ip_local_port_range": file(bad)}); ok {
			t.Errorf("%q read as a range", bad)
		}
	}
	if _, _, ok := read(fstest.MapFS{}); ok {
		t.Error("a missing file read as a range")
	}
}
