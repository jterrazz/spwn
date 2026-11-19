package procnet

import (
	"context"
	"encoding/hex"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type Proc interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Readlink(path string) (string, error)
}

type Procnet struct {
	proc Proc
	root string
}

func New(proc Proc, root string) *Procnet { return &Procnet{proc: proc, root: root} }

func (p *Procnet) at(name string) string { return path.Join(p.root, name) }

func (p *Procnet) Read(context.Context) ([]ports.Listener, error) {
	tcp, _ := p.proc.ReadFile(p.at("/proc/net/tcp"))
	tcp6, _ := p.proc.ReadFile(p.at("/proc/net/tcp6"))
	ls, inodes := parse(tcp6, tcp)
	p.owners(ls, inodes)
	return ls, nil
}

func (p *Procnet) Range(context.Context) (from, to int, ok bool) {
	// outside: Linux writes the range as "32768\t60999" in
	// /proc/sys/net/ipv4/ip_local_port_range.
	data, err := p.proc.ReadFile(p.at("/proc/sys/net/ipv4/ip_local_port_range"))
	f := strings.Fields(string(data))
	if err != nil || len(f) != 2 {
		return 0, 0, false
	}
	from, fromErr := strconv.Atoi(f[0])
	to, toErr := strconv.Atoi(f[1])
	return from, to, fromErr == nil && toErr == nil && ports.PlausibleEphemeral(from, to)
}

func parse(tables ...[]byte) (out []ports.Listener, inodes []string) {
	// invariant: /proc/net/tcp's local address is the address in host byte
	// order (little-endian on arm64): 0100007F is 127.0.0.1; an IPv6 address is
	// four such 32-bit words; only rows whose state is 0A (LISTEN) count.
	for _, table := range tables {
		for _, line := range strings.Split(string(table), "\n")[1:] {
			f := strings.Fields(line)
			if len(f) < 10 || f[3] != "0A" {
				continue
			}
			hexIP, hexPort, ok := strings.Cut(f[1], ":")
			if !ok {
				continue
			}
			port, err := strconv.ParseUint(hexPort, 16, 16)
			if err != nil {
				continue
			}
			ip, ok := decodeIP(hexIP)
			if !ok {
				continue
			}
			out = append(out, ports.Listener{Port: int(port), Bind: ip})
			inodes = append(inodes, f[9])
		}
	}
	return out, inodes
}

func (p *Procnet) owners(ls []ports.Listener, inodes []string) {
	byInode := map[string][]int{}
	for i, inode := range inodes {
		if inode != "" && inode != "0" {
			key := "socket:[" + inode + "]"
			byInode[key] = append(byInode[key], i)
		}
	}
	if len(byInode) == 0 {
		return
	}
	entries, err := p.proc.ReadDir(p.at("/proc"))
	if err != nil {
		return
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		fds, err := p.proc.ReadDir(p.at(path.Join("/proc", e.Name(), "fd")))
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := p.proc.Readlink(p.at(path.Join("/proc", e.Name(), "fd", fd.Name())))
			if err != nil {
				data, readErr := p.proc.ReadFile(p.at(path.Join("/proc", e.Name(), "fd", fd.Name())))
				if readErr != nil {
					continue
				}
				link = strings.TrimSpace(string(data))
			}
			for _, i := range byInode[link] {
				ls[i].PID, ls[i].Process = pid, p.program(e.Name())
			}
		}
	}
}

// invariant: a process is named by its program, argv[0]'s base name, since
// comm holds the main thread's name a runtime may rename (node-MainThread);
// an argv[0] rewritten into a title (postgres: checkpointer) leaves comm.
func (p *Procnet) program(pid string) string {
	cmdline, _ := p.proc.ReadFile(p.at(path.Join("/proc", pid, "cmdline")))
	argv0, _, _ := strings.Cut(string(cmdline), "\x00")
	if argv0 != "" && !strings.ContainsAny(argv0, " :") {
		return path.Base(argv0)
	}
	comm, _ := p.proc.ReadFile(p.at(path.Join("/proc", pid, "comm")))
	return strings.TrimSpace(string(comm))
}

func decodeIP(h string) (string, bool) {
	raw, err := hex.DecodeString(h)
	if err != nil || (len(raw) != 4 && len(raw) != 16) {
		return "", false
	}
	out := make([]byte, len(raw))
	for w := 0; w < len(raw); w += 4 {
		for i := range 4 {
			out[w+i] = raw[w+3-i]
		}
	}
	if len(out) == 16 && allZero(out[:10]) && out[10] == 0xff && out[11] == 0xff {
		out = out[12:]
	}
	if len(out) == 4 {
		parts := make([]string, 4)
		for i, b := range out {
			parts[i] = strconv.Itoa(int(b))
		}
		return strings.Join(parts, "."), true
	}
	return formatIPv6(out), true
}

func allZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

func formatIPv6(b []byte) string {
	groups := make([]int, 8)
	for i := range groups {
		groups[i] = int(b[2*i])<<8 | int(b[2*i+1])
	}
	start, length := -1, 0
	for i := 0; i < 8; {
		if groups[i] != 0 {
			i++
			continue
		}
		j := i
		for j < 8 && groups[j] == 0 {
			j++
		}
		if j-i > length && j-i > 1 {
			start, length = i, j-i
		}
		i = j
	}
	var parts []string
	for i := 0; i < 8; i++ {
		if i == start {
			parts = append(parts, "")
			if i == 0 {
				parts = append(parts, "")
			}
			i += length - 1
			if i == 7 {
				parts = append(parts, "")
			}
			continue
		}
		parts = append(parts, strconv.FormatInt(int64(groups[i]), 16))
	}
	return strings.Join(parts, ":")
}
