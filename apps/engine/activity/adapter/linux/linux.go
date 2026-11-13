package linux

import (
	"context"
	"errors"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/activity/contract"
)

// outside: the kernel counts process times in USER_HZ, 100, and resident
// memory in pages, 4 KiB on the kernel of every image station builds.
const (
	hz   = 100
	page = 4096
)

type Clock interface{ Now() time.Time }

type Meter struct {
	// value: the machine's root, where its /proc is read.
	root  fs.FS
	clock Clock

	mu sync.Mutex
	// value: each process's ticks and the machine's, at the last read: a
	// process's CPU share is what it took of the machine since then.
	ticks map[int]uint64
	total uint64
}

var _ contract.Meter = (*Meter)(nil)

func New(root fs.FS, clock Clock) *Meter {
	return &Meter{root: root, clock: clock, ticks: map[int]uint64{}}
}

func (m *Meter) read(name string) string {
	data, err := fs.ReadFile(m.root, "proc/"+name)
	if err != nil {
		return ""
	}
	return string(data)
}

func (m *Meter) Sample(ctx context.Context) (activity.Sample, error) {
	if err := ctx.Err(); err != nil {
		return activity.Sample{}, err
	}
	now := m.clock.Now().UTC()
	s := activity.Sample{At: now, Thermal: activity.ThermalUnknown}
	s.CPUBusy, s.CPUTotal, s.CPUs = ticks(m.read("stat"))
	if fields := strings.Fields(m.read("loadavg")); len(fields) >= 3 {
		for i := range s.Load {
			s.Load[i], _ = strconv.ParseFloat(fields[i], 64)
		}
	}
	memory := meminfo(m.read("meminfo"))
	s.MemTotal = memory["MemTotal"]
	if avail, ok := memory["MemAvailable"]; ok {
		s.MemUsed = max(s.MemTotal-avail, 0)
	}
	s.SwapUsed = max(memory["SwapTotal"]-memory["SwapFree"], 0)
	s.Pressure = pressure(m.read("pressure/memory"))
	s.DiskRead, s.DiskWritten = disks(m.read("diskstats"))
	s.NetIn, s.NetOut = links(m.read("net/dev"))
	if up := uptime(m.read("uptime")); up > 0 {
		s.Boot = now.Add(-time.Duration(up * float64(time.Second))).Truncate(time.Second)
	}
	return s, nil
}

func (m *Meter) Top(ctx context.Context) (activity.Busiest, error) {
	if err := ctx.Err(); err != nil {
		return activity.Busiest{}, err
	}
	all := m.processes()
	cpu := make([]activity.Process, 0, len(all))
	memory := make([]activity.Process, 0, len(all))
	for _, p := range all {
		cpu = append(cpu, activity.Process{PID: p.pid, Name: p.name, Value: p.cpu})
		memory = append(memory, activity.Process{PID: p.pid, Name: p.name, Value: float64(p.rss)})
	}
	return activity.Busiest{CPU: top(cpu), Memory: top(memory)}, nil
}

func (m *Meter) Process(ctx context.Context, pid int) (activity.Usage, error) {
	if err := ctx.Err(); err != nil {
		return activity.Usage{}, err
	}
	for _, p := range m.processes() {
		if p.pid == pid {
			return activity.Usage{CPU: p.cpu, Memory: p.rss, Started: p.started}, nil
		}
	}
	return activity.Usage{}, ErrNoProcess
}

var ErrNoProcess = errors.New("no such process")

type proc struct {
	pid     int
	name    string
	cpu     float64
	rss     int64
	started time.Time
}

// invariant: a process's CPU share is of the whole machine: the ticks it
// took over the ticks every CPU counted, since the last read, or since it
// started on the first.
func (m *Meter) processes() []proc {
	entries, err := fs.ReadDir(m.root, "proc")
	if err != nil {
		return nil
	}
	_, total, cpus := ticks(m.read("stat"))
	up := uptime(m.read("uptime"))
	now := m.clock.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := map[int]uint64{}
	var out []proc
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		name, used, started, ok := stat(m.read(e.Name() + "/stat"))
		if !ok {
			continue
		}
		p := proc{pid: pid, name: name}
		if up > 0 {
			p.started = now.Add(-time.Duration((up - float64(started)/hz) * float64(time.Second))).Truncate(time.Second)
		}
		if fields := strings.Fields(m.read(e.Name() + "/statm")); len(fields) >= 2 {
			p.rss = parseInt(fields[1]) * page
		}
		seen[pid] = used
		if before, ok := m.ticks[pid]; ok && total > m.total && used >= before {
			p.cpu = 100 * float64(used-before) / float64(total-m.total)
		} else if age := up*hz - float64(started); age > 0 && cpus > 0 {
			p.cpu = 100 * float64(used) / (age * float64(cpus))
		}
		p.cpu = min(p.cpu, 100)
		out = append(out, p)
	}
	m.ticks, m.total = seen, total
	return out
}

// outside: /proc/<pid>/stat is `pid (comm) state ...`; comm may hold spaces
// and parentheses, so it ends at the line's last ")". utime and stime are the
// 14th and 15th fields, starttime the 22nd.
func stat(text string) (name string, used, started uint64, ok bool) {
	open, end := strings.IndexByte(text, '('), strings.LastIndexByte(text, ')')
	if open < 0 || end < open {
		return "", 0, 0, false
	}
	fields := strings.Fields(text[end+1:])
	if len(fields) < 20 {
		return "", 0, 0, false
	}
	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)
	started, _ = strconv.ParseUint(fields[19], 10, 64)
	return text[open+1 : end], utime + stime, started, true
}

// outside: the first line of /proc/stat sums every CPU's ticks: user nice
// system idle iowait irq softirq steal; one cpuN line follows per CPU.
func ticks(text string) (busy, total uint64, cpus int) {
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}
		if fields[0] != "cpu" {
			cpus++
			continue
		}
		for i, f := range fields[1:min(len(fields), 9)] {
			n, _ := strconv.ParseUint(f, 10, 64)
			total += n
			if i != 3 && i != 4 {
				busy += n
			}
		}
	}
	return busy, total, cpus
}

func meminfo(text string) map[string]int64 {
	out := map[string]int64{}
	for _, line := range strings.Split(text, "\n") {
		key, rest, ok := strings.Cut(line, ":")
		if fields := strings.Fields(rest); ok && len(fields) > 0 {
			out[key] = parseInt(fields[0]) * 1024
		}
	}
	return out
}

// outside: PSI's `some avg10=` is the share of the last 10 s some task
// waited on memory: under 10 is normal, under 40 warn, else critical.
func pressure(text string) activity.Pressure {
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "some" {
			continue
		}
		value, ok := strings.CutPrefix(fields[1], "avg10=")
		avg, err := strconv.ParseFloat(value, 64)
		switch {
		case !ok || err != nil:
			return ""
		case avg < 10:
			return activity.PressureNormal
		case avg < 40:
			return activity.PressureWarn
		}
		return activity.PressureCritical
	}
	return ""
}

var notADisk = regexp.MustCompile(`^(loop|ram|zram|dm-|sr|md|nbd)`)

// invariant: only whole block devices count: a partition's bytes are its
// disk's already, and loop, ram and device-mapper devices sit on a disk
// counted once.
func disks(text string) (read, written uint64) {
	type device struct {
		name          string
		read, written uint64
	}
	var all []device
	names := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 || notADisk.MatchString(fields[2]) {
			continue
		}
		r, _ := strconv.ParseUint(fields[5], 10, 64)
		w, _ := strconv.ParseUint(fields[9], 10, 64)
		all = append(all, device{name: fields[2], read: r * 512, written: w * 512})
		names[fields[2]] = true
	}
	for _, d := range all {
		if partition(d.name, names) {
			continue
		}
		read += d.read
		written += d.written
	}
	return read, written
}

// outside: a partition is its disk's name and a number: vda1 of vda, nvme0n1p1
// of nvme0n1, mmcblk0p2 of mmcblk0.
func partition(name string, names map[string]bool) bool {
	digits := strings.TrimRight(name, "0123456789")
	if digits == name || digits == "" {
		return false
	}
	return names[digits] || (strings.HasSuffix(digits, "p") && names[strings.TrimSuffix(digits, "p")])
}

// invariant: only the VM's own links count, eth* and en*: a container's veth,
// a bridge, a CNI overlay or a tunnel carries bytes a real link counts once.
func links(text string) (rx, tx uint64) {
	for _, line := range strings.Split(text, "\n") {
		name, counters, ok := strings.Cut(line, ":")
		name = strings.TrimSpace(name)
		fields := strings.Fields(counters)
		if !ok || len(fields) < 9 || !strings.HasPrefix(name, "eth") && !strings.HasPrefix(name, "en") {
			continue
		}
		r, _ := strconv.ParseUint(fields[0], 10, 64)
		t, _ := strconv.ParseUint(fields[8], 10, 64)
		rx += r
		tx += t
	}
	return rx, tx
}

func uptime(text string) float64 {
	field, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	seconds, err := strconv.ParseFloat(field, 64)
	if err != nil || seconds < 0 {
		return 0
	}
	return seconds
}

func top(ps []activity.Process) []activity.Process {
	sorted := append([]activity.Process{}, ps...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Value > sorted[j].Value })
	if len(sorted) > activity.Top {
		sorted = sorted[:activity.Top]
	}
	return sorted
}

func parseInt(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}
