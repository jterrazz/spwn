package procnet

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract/contracttest"
)

type proc struct{ fstest.MapFS }

func (p proc) ReadFile(name string) ([]byte, error) {
	return p.MapFS.ReadFile(strings.TrimPrefix(name, "/"))
}

func (p proc) ReadDir(name string) ([]fs.DirEntry, error) {
	return p.MapFS.ReadDir(strings.TrimPrefix(name, "/"))
}

func (proc) Readlink(string) (string, error) { return "", fs.ErrInvalid }

func (proc) Symlink(string, string) error { return fs.ErrInvalid }

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
	for _, bad := range []string{"", "1", "9 x", "9000 8000", "1024 65535", "9999 60999", "32768 70000"} {
		if _, _, ok := read(fstest.MapFS{"proc/sys/net/ipv4/ip_local_port_range": file(bad)}); ok {
			t.Errorf("%q read as a range", bad)
		}
	}
	if _, _, ok := read(fstest.MapFS{}); ok {
		t.Error("a missing file read as a range")
	}
}

type counted struct {
	proc
	walks, ops int
}

func (c *counted) ReadFile(name string) ([]byte, error) {
	c.ops++
	return c.proc.ReadFile(name)
}

func (c *counted) ReadDir(name string) ([]fs.DirEntry, error) {
	c.ops++
	if strings.TrimSuffix(name, "/") == "/proc" {
		c.walks++
	}
	return c.proc.ReadDir(name)
}

func busyHost(processes int) fstest.MapFS {
	host := fstest.MapFS{
		"proc/net/tcp":    file(header + row("00000000:0016", "0A", "100") + row("0100007F:0BB8", "0A", "201")),
		"proc/net/tcp6":   file(header),
		"proc/1/cmdline":  file("/usr/sbin/sshd\x00"),
		"proc/1/fd/3":     file("socket:[100]"),
		"proc/50/cmdline": file("/usr/bin/node\x00server.js\x00"),
		"proc/50/fd/3":    file("socket:[201]"),
	}
	for pid := 100; pid < 100+processes; pid++ {
		for fd := range 12 {
			host[fmt.Sprintf("proc/%d/fd/%d", pid, fd)] = file(fmt.Sprintf("pipe:[%d]", pid*100+fd))
		}
	}
	return host
}

func TestProcnetWalksTheProcessesOnlyWhenAListenerIsNewOrLosesItsOwner(t *testing.T) {
	host := busyHost(40)
	c := &counted{proc: proc{host}}
	p := New(c, "")
	read := func() []ports.Listener {
		got, err := p.Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	owner := func(ls []ports.Listener, port int) string {
		for _, l := range ls {
			if l.Port == port {
				return fmt.Sprintf("%s/%d", l.Process, l.PID)
			}
		}
		return ""
	}

	first := read()
	if c.walks != 1 || owner(first, 3000) != "node/50" {
		t.Fatalf("first read: %d walks, node owner %q; want one walk and node/50", c.walks, owner(first, 3000))
	}
	opsOfWalk := c.ops
	c.ops = 0
	for range 20 {
		read()
	}
	if c.walks != 1 {
		t.Fatalf("20 reads of a steady set walked %d times in all; want the first walk only", c.walks)
	}
	if c.ops > 20*6 {
		t.Fatalf("a steady read costs %d operations (a walk is %d); want a handful", c.ops/20, opsOfWalk)
	}

	host["proc/net/tcp"] = file(header + row("00000000:0016", "0A", "100") + row("0100007F:0BB8", "0A", "201") + row("00000000:1F90", "0A", "202"))
	host["proc/60/cmdline"] = file("/usr/bin/python3\x00-m\x00http.server\x00")
	host["proc/60/fd/4"] = file("socket:[202]")
	if got := read(); c.walks != 2 || owner(got, 8080) != "python3/60" {
		t.Fatalf("a new listener: %d walks, owner %q; want a second walk and python3/60", c.walks, owner(got, 8080))
	}

	delete(host, "proc/50/fd/3")
	host["proc/51/cmdline"] = file("/usr/bin/node\x00child.js\x00")
	host["proc/51/fd/3"] = file("socket:[201]")
	if got := read(); c.walks != 3 || owner(got, 3000) != "node/51" {
		t.Fatalf("a socket handed to a child: %d walks, owner %q; want a third walk and node/51", c.walks, owner(got, 3000))
	}

	host["proc/net/tcp"] = file(header + row("00000000:0016", "0A", "100"))
	read()
	if got := read(); len(got) != 1 || c.walks != 3 {
		t.Fatalf("a listener gone: %d listeners, %d walks; want one listener and no walk", len(got), c.walks)
	}
}

func BenchmarkProcnetReadSteady(b *testing.B) {
	for _, processes := range []int{100, 350} {
		b.Run(fmt.Sprintf("%d-processes", processes), func(b *testing.B) {
			p := New(proc{busyHost(processes)}, "")
			if _, err := p.Read(context.Background()); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := p.Read(context.Background()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkProcnetReadWithAWalk(b *testing.B) {
	for _, processes := range []int{100, 350} {
		b.Run(fmt.Sprintf("%d-processes", processes), func(b *testing.B) {
			host := busyHost(processes)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := New(proc{host}, "").Read(context.Background()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
